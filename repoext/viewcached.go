package repoext

import (
	"ARP4G/arp"
	"context"
	"sync"
)

//用于只读查询类场景加速
type ViewCachedRepository[T any] struct {
	*arp.QueryRepositoryImpl[T]
	cache sync.Map
	count uint64
	mutex sync.Mutex
}

type NullEntity struct {
}

func (vcr *ViewCachedRepository[T]) UpdateCacheForEntity(id any, entity any) {
	if entity == nil {
		vcr.cache.Store(id, &NullEntity{})
	} else {
		vcr.cache.Store(id, entity)
	}
}

func (vcr *ViewCachedRepository[T]) Find(ctx context.Context, id any) *T {
	entity, ok := vcr.cache.Load(id)
	if ok {
		if _, ok := entity.(*NullEntity); ok {
			return nil
		}
		return entity.(*T)
	}
	entityFromStore := vcr.QueryRepositoryImpl.Find(ctx, id)
	vcr.UpdateCacheForEntity(id, entityFromStore)
	entity, _ = vcr.cache.Load(id)
	if _, ok := entity.(*NullEntity); ok {
		return nil
	}
	//实际上ViewCachedRepository的目的是只读的，所以查出来需要复制一份，保护一下
	return arp.CopyEntity(vcr.QueryRepositoryImpl.EntityType(), entity).(*T)
}

func (vcr *ViewCachedRepository[T]) Take(ctx context.Context, id any) *T {
	entity := vcr.QueryRepositoryImpl.Take(ctx, id)
	vcr.UpdateCacheForEntity(id, entity)
	return entity
}

func (vcr *ViewCachedRepository[T]) Put(ctx context.Context, entity *T) {
	vcr.QueryRepositoryImpl.Put(ctx, entity)
	vcr.UpdateCacheForEntity(vcr.QueryRepositoryImpl.GetEntityId(entity), entity)
}

func (vcr *ViewCachedRepository[T]) PutIfAbsent(ctx context.Context, entity *T) (actual *T, absent bool) {
	actual, absent = vcr.QueryRepositoryImpl.PutIfAbsent(ctx, entity)
	vcr.UpdateCacheForEntity(vcr.QueryRepositoryImpl.GetEntityId(actual), actual)
	return
}

func (vcr *ViewCachedRepository[T]) Remove(ctx context.Context, id any) *T {
	removed := vcr.QueryRepositoryImpl.Remove(ctx, id)
	vcr.UpdateCacheForEntity(id, nil)
	return removed
}

func (vcr *ViewCachedRepository[T]) TakeOrPutIfAbsent(ctx context.Context, id any, newEntity *T) *T {
	entity := vcr.QueryRepositoryImpl.TakeOrPutIfAbsent(ctx, id, newEntity)
	vcr.UpdateCacheForEntity(id, entity)
	return entity
}

func (vcr *ViewCachedRepository[T]) updateCount(count uint64) {
	vcr.mutex.Lock()
	vcr.count = count
	vcr.mutex.Unlock()
}

func (vcr *ViewCachedRepository[T]) Count(ctx context.Context) (uint64, error) {
	if vcr.count != 0 {
		return vcr.count, nil
	}
	count, err := vcr.QueryRepositoryImpl.Count(ctx)
	if err != nil {
		return 0, err
	}
	vcr.updateCount(count)
	return count, nil
}

func NewViewCachedRepository[T any](repository arp.QueryRepository[T]) arp.QueryRepository[T] {
	return &ViewCachedRepository[T]{QueryRepositoryImpl: repository.(*arp.QueryRepositoryImpl[T])}
}
