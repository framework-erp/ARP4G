package repoimpl

import (
	"context"
	"errors"
	"reflect"
	"sync"

	"github.com/framework-arp/ARP4G/arp"
)

type MemStore[T any] struct {
	data         sync.Map
	typeFullname string
}

func (store *MemStore[T]) Load(ctx context.Context, id any) (entity T, found bool, err error) {
	entitLoad, ok := store.data.Load(id)
	if !ok {
		return entity, false, nil
	}
	return arp.CopyEntity(store.typeFullname, entitLoad).(T), true, nil
}

func (store *MemStore[T]) Save(ctx context.Context, id any, entity T) error {
	if _, ok := store.data.Load(id); ok {
		return errors.New("can not 'Save' since entity already exists")
	}
	store.data.Store(id, entity)
	return nil
}

func (store *MemStore[T]) SaveAll(ctx context.Context, entitiesToInsert map[any]any, entitiesToUpdate map[any]*arp.ProcessEntity) error {
	for k, v := range entitiesToInsert {
		if _, ok := store.data.Load(k); ok {
			return errors.New("can not 'Save' since entity already exists")
		}
		store.data.Store(k, v)
	}
	for k, v := range entitiesToUpdate {
		store.data.Store(k, v.Entity())
	}
	return nil
}

func (store *MemStore[T]) RemoveAll(ctx context.Context, ids []any) error {
	for _, id := range ids {
		store.data.Delete(id)
	}
	return nil
}

type MemMutexes struct {
	mutexes sync.Map
}

func (memMutexes *MemMutexes) Lock(ctx context.Context, id any) (ok bool, absent bool, err error) {
	mutex, loadOk := memMutexes.mutexes.Load(id)
	if !loadOk {
		return false, true, nil
	}
	//将来要支持try次数后失败
	mutex.(*sync.Mutex).Lock()
	return true, false, nil
}

func (memMutexes *MemMutexes) NewAndLock(ctx context.Context, id any) (ok bool, err error) {
	mutex, loaded := memMutexes.mutexes.LoadOrStore(id, &sync.Mutex{})
	if loaded {
		return false, nil
	}
	(mutex.(*sync.Mutex)).Lock()
	return true, nil
}

func (memMutexes *MemMutexes) UnlockAll(ctx context.Context, ids []any) {
	for _, id := range ids {
		mutex, ok := memMutexes.mutexes.Load(id)
		if ok {
			mutex.(*sync.Mutex).Unlock()
		}
	}
}

func NewMemRepository[T any](newZeroEntity arp.NewZeroEntity[T]) arp.Repository[T] {
	zeroEntity := newZeroEntity()
	entityType := reflect.TypeOf(zeroEntity).Elem()
	typeFullname := entityType.PkgPath() + "." + entityType.Name()
	store := &MemStore[T]{typeFullname: typeFullname}
	mutexes := &MemMutexes{}
	return arp.NewRepository[T](store, mutexes, newZeroEntity)
}
