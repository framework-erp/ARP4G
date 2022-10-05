package arp

import (
	"context"
	"errors"
	"reflect"
)

func Start(ctx context.Context) context.Context {
	return context.WithValue(ctx, procCtxKey, newProcessContext())
}

func Finish(ctx context.Context) error {
	defer func() {
		pc, ok := getProcessContext(ctx)
		if !ok {
			return
		}
		releaseProcessEntities(ctx, pc)
	}()
	pc, ok := getProcessContext(ctx)
	if !ok {
		return nil
	}
	if err := flushProcessEntities(ctx, pc); err != nil {
		return err
	}
	return nil
}

func Abort(ctx context.Context) {
	pc, ok := getProcessContext(ctx)
	if !ok {
		return
	}
	releaseProcessEntities(ctx, pc)
}

func Go(ctx context.Context, f func(ctx context.Context)) (err error) {
	ctx = Start(ctx)
	defer func() {
		if r := recover(); r == nil {
			err = Finish(ctx)
		} else {
			Abort(ctx)
			switch x := r.(type) {
			case string:
				err = errors.New(x)
			case error:
				err = x
			default:
				err = errors.New("unknow panic")
			}
		}
	}()
	f(ctx)
	return
}

//收集，共享，输出一个过程中的数据。包括过程信息，过程中涉及到的实体的状态变化
type ProcessContext struct {
	entities       map[string]*repositoryProcessEntities
	singletonTypes []string
}

func (pc *ProcessContext) addEntityTakenFromRepo(entityType string, id any, entity any) {
	rpes := pc.getRepositoryProcessEntities(entityType)
	rpes.addEntityTaken(entityType, id, entity)
}

func (pc *ProcessContext) getRepositoryProcessEntities(entityType string) *repositoryProcessEntities {
	rpes := pc.entities[entityType]
	if rpes == nil {
		rpes = newRepositoryProcessEntities()
		pc.entities[entityType] = rpes
	}
	return rpes
}

func (pc *ProcessContext) copyEntityInProcess(entityType string, id any) any {
	rpes := pc.getRepositoryProcessEntities(entityType)
	return rpes.copyEntityInProcess(entityType, id)
}

func (pc *ProcessContext) getEntityInProcess(entityType string, id any) *ProcessEntity {
	rpes := pc.getRepositoryProcessEntities(entityType)
	return rpes.getEntityInProcess(id)
}

func (pc *ProcessContext) addNewEntity(entityType string, id any, entity any) *ProcessEntity {
	rpes := pc.getRepositoryProcessEntities(entityType)
	return rpes.addNewEntity(id, entity)
}

func (pc *ProcessContext) addEntityTakenFromSingletonRepo(entityType string) {
	pc.singletonTypes = append(pc.singletonTypes, entityType)
}

func newProcessContext() *ProcessContext {
	return &ProcessContext{entities: make(map[string]*repositoryProcessEntities)}
}

//针对某个仓库收集的，在一个过程中变化的实体
type repositoryProcessEntities struct {
	entities map[any]*ProcessEntity
}

func newRepositoryProcessEntities() *repositoryProcessEntities {
	return &repositoryProcessEntities{make(map[any]*ProcessEntity)}
}

func (rpes *repositoryProcessEntities) addEntityTaken(entityType string, id any, entity any) {
	rpes.entities[id] = &ProcessEntity{CopyEntity(entityType, entity), entity, &TakenFromRepoState{}}
}

func (rpes *repositoryProcessEntities) copyEntityInProcess(entityType string, id any) any {
	processEntity := rpes.entities[id]
	if processEntity == nil {
		return nil
	}
	return processEntity.copyEntity(entityType)
}

func (rpes *repositoryProcessEntities) getEntityInProcess(id any) *ProcessEntity {
	return rpes.entities[id]
}

func (rpes *repositoryProcessEntities) addNewEntity(id any, entity any) *ProcessEntity {
	processEntity := rpes.entities[id]
	if processEntity == nil {
		processEntity = &ProcessEntity{nil, entity, &CreatedInProcState{}}
		rpes.entities[id] = processEntity
		return processEntity
	}
	processEntity.entity = entity
	return processEntity
}

//在当前过程当中的实体
type ProcessEntity struct {
	//刚加载上来时候的实体快照
	snapshot any
	entity   any
	state    ProcessEntityState
}

func (pe *ProcessEntity) State() ProcessEntityState {
	return pe.state
}

func (pe *ProcessEntity) Entity() any {
	return pe.entity
}

func (pe *ProcessEntity) changeStateByTake() {
	pe.state = pe.state.transferByTake()
}

func (pe *ProcessEntity) changeStateByPut() {
	pe.state = pe.state.transferByPut()
}

func (pe *ProcessEntity) changeStateByRemove() {
	pe.state = pe.state.transferByRemove()
}

func (pe *ProcessEntity) changeStateByPutIfAbsent() {
	pe.state = pe.state.transferByPutIfAbsen()
}

func (pe *ProcessEntity) copyEntity(entityType string) any {
	if !pe.isAvailable() {
		return nil
	}
	return CopyEntity(entityType, pe.entity)
}

func (pe *ProcessEntity) isAvailable() bool {
	return pe.state.isEntityAvailable()
}

func (pe *ProcessEntity) isAddByTake() bool {
	return pe.state.isAddByTake()
}

//过程当中的实体的可能的几种状态
type ProcessEntityState interface {
	transferByTake() ProcessEntityState
	transferByPut() ProcessEntityState
	transferByPutIfAbsen() ProcessEntityState
	transferByRemove() ProcessEntityState
	isEntityAvailable() bool
	isAddByTake() bool
}

//从仓库中取来的状态
type TakenFromRepoState struct {
}

func (state *TakenFromRepoState) transferByTake() ProcessEntityState {
	return state
}
func (state *TakenFromRepoState) transferByPut() ProcessEntityState {
	return &ErrorState{}
}
func (state *TakenFromRepoState) transferByPutIfAbsen() ProcessEntityState {
	return state
}
func (state *TakenFromRepoState) transferByRemove() ProcessEntityState {
	return &ToRemoveInRepoState{}
}
func (state *TakenFromRepoState) isEntityAvailable() bool {
	return true
}
func (state *TakenFromRepoState) isAddByTake() bool {
	return true
}

//在过程中新建的状态
type CreatedInProcState struct {
}

func (state *CreatedInProcState) transferByTake() ProcessEntityState {
	return state
}
func (state *CreatedInProcState) transferByPut() ProcessEntityState {
	return &ErrorState{}
}
func (state *CreatedInProcState) transferByPutIfAbsen() ProcessEntityState {
	return state
}
func (state *CreatedInProcState) transferByRemove() ProcessEntityState {
	return &TransientInProcState{}
}
func (state *CreatedInProcState) isEntityAvailable() bool {
	return true
}
func (state *CreatedInProcState) isAddByTake() bool {
	return false
}

//瞬时状态，就是在过程中创建之后又在过程中删除，和仓库没有关系
type TransientInProcState struct {
}

func (state *TransientInProcState) transferByTake() ProcessEntityState {
	return &ErrorState{}
}
func (state *TransientInProcState) transferByPut() ProcessEntityState {
	return &CreatedInProcState{}
}
func (state *TransientInProcState) transferByPutIfAbsen() ProcessEntityState {
	return &CreatedInProcState{}
}
func (state *TransientInProcState) transferByRemove() ProcessEntityState {
	return state
}
func (state *TransientInProcState) isEntityAvailable() bool {
	return false
}
func (state *TransientInProcState) isAddByTake() bool {
	return false
}

//需要去仓库中删除的状态
type ToRemoveInRepoState struct {
}

func (state *ToRemoveInRepoState) transferByTake() ProcessEntityState {
	return &ErrorState{}
}
func (state *ToRemoveInRepoState) transferByPut() ProcessEntityState {
	return &TakenFromRepoState{}
}
func (state *ToRemoveInRepoState) transferByPutIfAbsen() ProcessEntityState {
	return &TakenFromRepoState{}
}
func (state *ToRemoveInRepoState) transferByRemove() ProcessEntityState {
	return state
}
func (state *ToRemoveInRepoState) isEntityAvailable() bool {
	return false
}
func (state *ToRemoveInRepoState) isAddByTake() bool {
	return true
}

//错误状态
type ErrorState struct {
}

func (state *ErrorState) transferByTake() ProcessEntityState {
	return state
}
func (state *ErrorState) transferByPut() ProcessEntityState {
	return state
}
func (state *ErrorState) transferByPutIfAbsen() ProcessEntityState {
	return state
}
func (state *ErrorState) transferByRemove() ProcessEntityState {
	return state
}
func (state *ErrorState) isEntityAvailable() bool {
	return false
}
func (state *ErrorState) isAddByTake() bool {
	return false
}

type key int

var procCtxKey key

func getProcessContext(ctx context.Context) (*ProcessContext, bool) {
	pc, ok := ctx.Value(procCtxKey).(*ProcessContext)
	return pc, ok
}

func TakenFromRepository(ctx context.Context, entityType string, id any, entity any) {
	pc, ok := getProcessContext(ctx)
	if !ok {
		return
	}
	pc.addEntityTakenFromRepo(entityType, id, entity)
}

func flushProcessEntities(ctx context.Context, pc *ProcessContext) error {
	for entityType, repoPes := range pc.entities {
		entitiesToInsert := make(map[any]any)
		entitiesToUpdate := make(map[any]*ProcessEntity)
		idsToRemoveEntity := make([]any, 0, len(repoPes.entities))
		for k, v := range repoPes.entities {
			switch v.state.(type) {
			case *TakenFromRepoState:
				if !reflect.DeepEqual(v.snapshot, v.entity) {
					entitiesToUpdate[k] = v
				}
			case *CreatedInProcState:
				entitiesToInsert[k] = v.entity
			case *ToRemoveInRepoState:
				idsToRemoveEntity = append(idsToRemoveEntity, k)
			default:
			}
		}
		if err := getRepository(entityType).FlushProcessEntities(ctx, entitiesToInsert, entitiesToUpdate, idsToRemoveEntity); err != nil {
			return err
		}
	}
	return nil
}

func releaseProcessEntities(ctx context.Context, pc *ProcessContext) {
	for entityType, repoPes := range pc.entities {
		ids := make([]any, 0, len(repoPes.entities))
		for id, processEntity := range repoPes.entities {
			if processEntity.isAddByTake() {
				ids = append(ids, id)
			}
		}
		getRepository(entityType).ReleaseProcessEntities(ctx, ids)
	}
	for _, entityType := range pc.singletonTypes {
		getSingletonRepository(entityType).ReleaseProcessEntity(ctx)
	}
}

func CopyEntityInProcess(ctx context.Context, entityType string, id any) any {
	pc, ok := getProcessContext(ctx)
	if !ok {
		return nil
	}
	return pc.copyEntityInProcess(entityType, id)
}

func TakeEntityInProcess(ctx context.Context, entityType string, id any) (exists bool, entity any) {
	pc, ok := getProcessContext(ctx)
	if !ok {
		return false, nil
	}
	processEntity := pc.getEntityInProcess(entityType, id)
	if processEntity == nil {
		return false, nil
	}
	if processEntity.isAvailable() {
		processEntity.changeStateByTake()
		return true, processEntity.entity
	} else {
		return true, nil
	}
}

func EntityAvailableInProcess(ctx context.Context, entityType string, id any) bool {
	pc, ok := getProcessContext(ctx)
	if !ok {
		return false
	}
	processEntity := pc.getEntityInProcess(entityType, id)
	if processEntity == nil {
		return false
	}
	return processEntity.isAvailable()
}

func PutNewEntityToProcess(ctx context.Context, entityType string, id any, entity any) {
	pc, ok := getProcessContext(ctx)
	if !ok {
		return
	}
	processEntity := pc.addNewEntity(entityType, id, entity)
	if !processEntity.isAvailable() {
		processEntity.changeStateByPut()
	}
}

func RemoveEntityInProcess(ctx context.Context, entityType string, id any, entity any) {
	pc, ok := getProcessContext(ctx)
	if !ok {
		return
	}
	processEntity := pc.getEntityInProcess(entityType, id)
	processEntity.changeStateByRemove()
}

func GetFromOrPutEntityToProcessIfNotAvailable(ctx context.Context, entityType string, id any, entity any) (actual any, get bool) {
	pc, ok := getProcessContext(ctx)
	if !ok {
		return nil, true
	}
	processEntity := pc.getEntityInProcess(entityType, id)
	if processEntity == nil {
		return nil, true
	}
	if processEntity.isAvailable() {
		actual = processEntity.entity
		get = true
	} else {
		processEntity.entity = entity
		actual = entity
		get = false
	}
	processEntity.changeStateByPutIfAbsent()
	return
}

func TakenFromSingletonRepository(ctx context.Context, entityType string) {
	pc, ok := getProcessContext(ctx)
	if !ok {
		return
	}
	pc.addEntityTakenFromSingletonRepo(entityType)
}
