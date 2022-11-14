package repoext

import (
	"context"
	"sync"

	"github.com/framework-arp/ARP4G/arp"
)

//用于只读查询类场景加速
type ViewCachedRepository[T any] struct {
	*arp.RepositoryImpl[T]
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

func (vcr *ViewCachedRepository[T]) Find(ctx context.Context, id any) (entity T, found bool) {
	entityLoad, ok := vcr.cache.Load(id)
	if ok {
		if _, ok := entityLoad.(*NullEntity); ok {
			return entity, false
		}
		return entityLoad.(T), true
	}
	entityFromStore, found := vcr.RepositoryImpl.Find(ctx, id)
	vcr.UpdateCacheForEntity(id, entityFromStore)
	entityLoad, _ = vcr.cache.Load(id)
	if _, ok := entityLoad.(*NullEntity); ok {
		return entity, false
	}
	//实际上ViewCachedRepository的目的是只读的，所以查出来需要复制一份，保护一下
	return arp.CopyEntity(vcr.RepositoryImpl.EntityType(), entityLoad).(T), true
}

func (vcr *ViewCachedRepository[T]) Take(ctx context.Context, id any) (entity T, found bool) {
	entity, found = vcr.RepositoryImpl.Take(ctx, id)
	vcr.UpdateCacheForEntity(id, entity)
	return
}

func (vcr *ViewCachedRepository[T]) Put(ctx context.Context, id any, entity T) {
	vcr.RepositoryImpl.Put(ctx, id, entity)
	vcr.UpdateCacheForEntity(id, entity)
}

func (vcr *ViewCachedRepository[T]) PutIfAbsent(ctx context.Context, id any, entity T) (actual T, absent bool) {
	actual, absent = vcr.RepositoryImpl.PutIfAbsent(ctx, id, entity)
	vcr.UpdateCacheForEntity(id, actual)
	return
}

func (vcr *ViewCachedRepository[T]) Remove(ctx context.Context, id any) (removed T, exists bool) {
	removed, exists = vcr.RepositoryImpl.Remove(ctx, id)
	if exists {
		vcr.UpdateCacheForEntity(id, nil)
	}
	return
}

func (vcr *ViewCachedRepository[T]) TakeOrPutIfAbsent(ctx context.Context, id any, newEntity T) T {
	entity := vcr.RepositoryImpl.TakeOrPutIfAbsent(ctx, id, newEntity)
	vcr.UpdateCacheForEntity(id, entity)
	return entity
}

func (vcr *ViewCachedRepository[T]) updateCount(count uint64) {
	vcr.mutex.Lock()
	vcr.count = count
	vcr.mutex.Unlock()
}

func NewViewCachedRepository[T any](repository arp.Repository[T]) arp.Repository[T] {
	return &ViewCachedRepository[T]{RepositoryImpl: repository.(*arp.RepositoryImpl[T])}
}
