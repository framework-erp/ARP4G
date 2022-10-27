package arp

import (
	"context"
	"errors"
	"reflect"
	"sync"
)

//仓库是存放聚合的地方，聚合只会通过它的id来获取。
//仓库是一个接口，是为了清晰地罗列它的功能，除此之外没有其他特别的原因。
//泛型类型T代表聚合的类型，仓库创建的时候决定T的实际类型，
//例如：
//repoimpl.NewMemRepository(func() *Order { return &Order{} }) 这里创建了一个实际类型为“Order”的内存仓库
//repoimpl.NewMemRepository(func() *Product { return &Product{} }) 这里创建了一个实际类型为“Product”的内存仓库
type Repository[T any] interface {
	Find(ctx context.Context, id any) (entity T, found bool)
	Take(ctx context.Context, id any) (entity T, found bool)
	Put(ctx context.Context, id any, entity T)
	PutIfAbsent(ctx context.Context, id any, entity T) (actual T, absent bool)
	Remove(ctx context.Context, id any) (removed T, exists bool)
	TakeOrPutIfAbsent(ctx context.Context, id any, newEntity T) T
}

//对内的仓库操作集合
type innerRepository interface {
	FlushProcessEntities(ctx context.Context, entitiesToInsert map[any]any, entitiesToUpdate map[any]*ProcessEntity, idsToRemoveEntity []any) error
	ReleaseProcessEntities(ctx context.Context, ids []any)
}

type RepositoryImpl[T any] struct {
	entityType string
	store      Store[T]
	mutexes    Mutexes
}

type Store[T any] interface {
	//加载上来的是原始entity的一个副本(copy)，基于数据库的store天然就是copy，而内存store需要实现copy而不能传递原始entity的指针
	Load(ctx context.Context, id any) (entity T, found bool, err error)
	Save(ctx context.Context, id any, entity T) error
	SaveAll(ctx context.Context, entitiesToInsert map[any]any, entitiesToUpdate map[any]*ProcessEntity) error
	RemoveAll(ctx context.Context, ids []any) error
}

type NewZeroEntity[T any] func() T

type Mutexes interface {
	Lock(ctx context.Context, id any) (ok bool, absent bool, err error)
	//返回ok不为true那就是已创建了
	NewAndLock(ctx context.Context, id any) (ok bool, err error)
	UnlockAll(ctx context.Context, ids []any)
}

func (repository *RepositoryImpl[T]) EntityType() string {
	return repository.entityType
}

func (repository *RepositoryImpl[T]) Find(ctx context.Context, id any) (entity T, found bool) {
	entityInProcess := CopyEntityInProcess(ctx, repository.entityType, id)
	if entityInProcess != nil {
		value, _ := entityInProcess.(T)
		return value, true
	}
	entity, found, err := repository.store.Load(ctx, id)
	if err != nil {
		panic("Find error: " + err.Error())
	}
	return entity, found
}

func (repository *RepositoryImpl[T]) Take(ctx context.Context, id any) (entity T, found bool) {
	exists, ent := TakeEntityInProcess(ctx, repository.entityType, id)
	if exists {
		value, _ := ent.(T)
		return value, true
	}
	ok, absent, err := repository.mutexes.Lock(ctx, id)
	if err != nil {
		panic("Take error: " + err.Error())
	}
	var existsEntity T
	if absent {
		//检查entity存在且补锁
		existsEntity, found = repository.Find(ctx, id)
		if !found {
			return entity, false
		}
		ok, err := repository.mutexes.NewAndLock(ctx, id)
		if err != nil {
			panic("Take error: " + err.Error())
		}
		if !ok {
			//补锁不成功那就是有人抢先补锁，那么这里就需要再去获得锁了
			ok, _, err = repository.mutexes.Lock(ctx, id)
			if err != nil {
				panic("Take error: " + err.Error())
			}
			if !ok {
				panic("Take error: can not 'Take' since entity is occupied")
			}
		}
	} else {
		if !ok {
			panic("Take error: can not 'Take' since entity is occupied")
		}
		existsEntity, found = repository.Find(ctx, id)
		if err != nil {
			panic("Take error: " + err.Error())
		}
		if !found {
			return entity, false
		}
	}
	TakenFromRepository(ctx, repository.entityType, id, existsEntity)
	return existsEntity, true
}

func (repository *RepositoryImpl[T]) Put(ctx context.Context, id any, entity T) {
	if EntityAvailableInProcess(ctx, repository.entityType, id) {
		panic("can not 'Put' since entity already exists")
	}
	PutNewEntityToProcess(ctx, repository.entityType, id, entity)
}

func (repository *RepositoryImpl[T]) PutIfAbsent(ctx context.Context, id any, entity T) (actual T, absent bool) {
	//先要看过程中的，如果有可用的那就拿来做实际值，如果有但是不可用那就取用新值且新值覆盖老值
	entityGetOrPut, get := GetFromOrPutEntityToProcessIfNotAvailable(ctx, repository.entityType, id, entity)
	if entityGetOrPut != nil {
		actual, _ = entityGetOrPut.(T)
		return actual, !get
	}
	ok, err := repository.mutexes.NewAndLock(ctx, id)
	if err != nil {
		panic("PutIfAbsent error: " + err.Error())
	}
	if !ok {
		actual, _ = repository.Take(ctx, id)
		return actual, false
	}
	if err = repository.store.Save(ctx, id, entity); err != nil {
		panic("PutIfAbsent error: " + err.Error())
	}
	TakenFromRepository(ctx, repository.entityType, id, entity)
	return entity, true
}

func (repository *RepositoryImpl[T]) Remove(ctx context.Context, id any) (removed T, exists bool) {
	entity, found := repository.Take(ctx, id)
	if found {
		RemoveEntityInProcess(ctx, repository.entityType, id, entity)
		return entity, true
	}
	return removed, false
}

func (repository *RepositoryImpl[T]) TakeOrPutIfAbsent(ctx context.Context, id any, newEntity T) T {
	entity, found := repository.Take(ctx, id)
	if !found {
		actual, _ := repository.PutIfAbsent(ctx, id, newEntity)
		return actual
	}
	return entity
}

func (repository *RepositoryImpl[T]) FlushProcessEntities(ctx context.Context, entitiesToInsert map[any]any, entitiesToUpdate map[any]*ProcessEntity, idsToRemoveEntity []any) error {
	err := repository.store.SaveAll(ctx, entitiesToInsert, entitiesToUpdate)
	if err != nil {
		return err
	}
	err = repository.store.RemoveAll(ctx, idsToRemoveEntity)
	if err != nil {
		return err
	}
	return nil
}

func (repository *RepositoryImpl[T]) ReleaseProcessEntities(ctx context.Context, ids []any) {
	repository.mutexes.UnlockAll(ctx, ids)
}

func NewRepository[T any](store Store[T], mutexes Mutexes, newZeroEntityFunc NewZeroEntity[T]) Repository[T] {
	zeroEntity := newZeroEntityFunc()
	entityType := reflect.TypeOf(zeroEntity).Elem()
	typeFullname := entityType.PkgPath() + "." + entityType.Name()
	generateEntityCopier(typeFullname, entityType, newZeroEntityFunc)
	repo := &RepositoryImpl[T]{typeFullname, store, mutexes}
	registerRepository(repo)
	return repo
}

//包含常用的查询功能，复杂的查询请通过其他机制实现
type QueryFuncs[T any] interface {
	QueryAllIds(ctx context.Context) (ids []any, err error)
	Count(ctx context.Context) (uint64, error)
	QueryAllByField(ctx context.Context, fieldName string, fieldValue any) ([]T, error)
}

//包含一些常用查询功能的仓库，作为反CQRS（命令查询分离模式）设计，具有现实意义
type QueryRepository[T any] interface {
	Repository[T]
	QueryFuncs[T]
}

type QueryRepositoryImpl[T any] struct {
	*RepositoryImpl[T]
	QueryFuncs[T]
}

func NewQueryRepository[T any](repository Repository[T], queryFuncs QueryFuncs[T]) QueryRepository[T] {
	return &QueryRepositoryImpl[T]{repository.(*RepositoryImpl[T]), queryFuncs}
}

//保存的是一个不存在于某个集合当中的独立的实体。只在内存中，如需从数据库加载初始数据，则在系统启动时完成加载
type SingletonRepository[T any] interface {
	Get(ctx context.Context) (*T, error)
	Take(ctx context.Context) (*T, error)
	Put(ctx context.Context, entity *T) error
}

//对内的独立实体仓库操作集合
type innerSingletonRepository interface {
	ReleaseProcessEntity(ctx context.Context)
}

type SingletonRepositoryImpl[T any] struct {
	entityType string
	entity     *T
	mutex      *sync.Mutex
}

func (repo *SingletonRepositoryImpl[T]) Get(ctx context.Context) (*T, error) {
	return repo.entity, nil
}

func (repo *SingletonRepositoryImpl[T]) Take(ctx context.Context) (*T, error) {
	repo.mutex.Lock()
	TakenFromSingletonRepository(ctx, repo.entityType)
	return repo.entity, nil
}

func (repo *SingletonRepositoryImpl[T]) Put(ctx context.Context, entity *T) error {
	repo.entity = entity
	return nil
}
func (repo *SingletonRepositoryImpl[T]) ReleaseProcessEntity(ctx context.Context) {
	repo.mutex.Unlock()
}

func NewSingletonRepository[T any](entity *T) SingletonRepository[T] {
	entityType := reflect.TypeOf(entity)
	typeFullname := entityType.PkgPath() + "." + entityType.Name()
	repo := &SingletonRepositoryImpl[T]{typeFullname, entity, &sync.Mutex{}}
	registerSingletonRepository(repo)
	return repo
}

type MockStore[T any] struct {
	data map[any]*T
}

func NewMockStore[T any]() *MockStore[T] {
	return &MockStore[T]{make(map[any]*T)}
}

func (store *MockStore[T]) Load(ctx context.Context, id any) (entity T, found bool, err error) {
	entityPtr := store.data[id]
	if entityPtr == nil {
		return entity, false, nil
	}
	return *entityPtr, true, nil
}

func (store *MockStore[T]) Save(ctx context.Context, id any, entity T) error {
	store.data[id] = &entity
	return nil
}

func (store *MockStore[T]) SaveAll(ctx context.Context, entitiesToInsert map[any]any, entitiesToUpdate map[any]*ProcessEntity) error {
	for k, v := range entitiesToInsert {
		t := v.(T)
		store.data[k] = &t
	}
	for k, v := range entitiesToUpdate {
		t := v.entity.(T)
		store.data[k] = &t
	}
	return nil
}

func (store *MockStore[T]) RemoveAll(ctx context.Context, ids []any) error {
	for id := range ids {
		delete(store.data, id)
	}
	return nil
}

type MockMutexes struct {
}

func NewMockMutexes() *MockMutexes {
	return &MockMutexes{}
}

func (mutexes *MockMutexes) Lock(ctx context.Context, id any) (ok bool, absent bool, err error) {
	return true, false, nil
}

func (mutexes *MockMutexes) NewAndLock(ctx context.Context, id any) (ok bool, err error) {
	return true, nil
}

func (mutexes *MockMutexes) UnlockAll(ctx context.Context, ids []any) {
}

type MockQueryFuncs[T any] struct {
}

func (qf *MockQueryFuncs[T]) QueryAllIds(ctx context.Context) (ids []any, err error) {
	return nil, errors.New("unsupported")
}

func (qf *MockQueryFuncs[T]) Count(ctx context.Context) (uint64, error) {
	return 0, errors.New("unsupported")
}

func (qf *MockQueryFuncs[T]) QueryAllByField(ctx context.Context, fieldName string, fieldValue any) ([]T, error) {
	return nil, errors.New("unsupported")
}

func NewMockRepository[T any](newZeroEntityFunc NewZeroEntity[T]) Repository[T] {
	return NewRepository[T](NewMockStore[T](), NewMockMutexes(), newZeroEntityFunc)
}

func NewMockQueryRepository[T any](newZeroEntityFunc NewZeroEntity[T]) QueryRepository[T] {
	mockRepo := (NewMockRepository(newZeroEntityFunc)).(*RepositoryImpl[T])
	return &QueryRepositoryImpl[T]{mockRepo, &MockQueryFuncs[T]{}}
}
