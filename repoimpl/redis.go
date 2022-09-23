package repoimpl

import (
	"context"
	"encoding/json"

	"github.com/zhengchengdong/ARP4G/arp"
	"github.com/zhengchengdong/ARP4G/util"

	"github.com/go-redis/redis/v8"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v8"
)

type RedisStore[T any] struct {
	client         *redis.Client
	key            string
	newEmptyEntity arp.NewZeroEntity[T]
}

func (store *RedisStore[T]) Load(ctx context.Context, id any) (*T, error) {
	jsonStr := store.client.HGet(ctx, store.key+util.Strval(id), "json").Val()
	if jsonStr == "" {
		return nil, nil
	}
	entity := store.newEmptyEntity()
	json.Unmarshal([]byte(jsonStr), entity)
	return entity, nil
}

func (store *RedisStore[T]) Save(ctx context.Context, id any, entity *T) error {
	json, err := json.Marshal(entity)
	if err != nil {
		return err
	}
	store.client.HSet(ctx, store.key+util.Strval(id), "json", string(json))
	return nil
}

func (store *RedisStore[T]) SaveAll(ctx context.Context, entitiesToInsert map[any]any, entitiesToUpdate map[any]*arp.ProcessEntity) error {
	for k, v := range entitiesToInsert {
		json, err := json.Marshal(v)
		if err != nil {
			return err
		}
		store.client.HSet(ctx, store.key+util.Strval(k), "json", string(json))
	}
	for k, v := range entitiesToUpdate {
		json, err := json.Marshal(v.Entity())
		if err != nil {
			return err
		}
		store.client.HSet(ctx, store.key+util.Strval(k), "json", string(json))
	}
	return nil
}

func (store *RedisStore[T]) RemoveAll(ctx context.Context, ids []any) error {
	for id := range ids {
		store.client.HDel(ctx, store.key+util.Strval(id), "json")
	}
	return nil
}

func NewRedisStore[T any](client *redis.Client, key string, newEmptyEntity arp.NewZeroEntity[T]) *RedisStore[T] {
	return &RedisStore[T]{client, key, newEmptyEntity}
}

type RedisMutexes struct {
	key string
	rs  *redsync.Redsync
}

func (mutexes *RedisMutexes) Lock(ctx context.Context, id any) (ok bool, absent bool, err error) {
	mutexname := mutexes.key + util.Strval(id)
	mutex := mutexes.rs.NewMutex(mutexname)
	if lockErr := mutex.Lock(); lockErr != nil {
		return false, false, nil
	}
	return true, false, nil
}

func (mutexes *RedisMutexes) NewAndLock(ctx context.Context, id any) (ok bool, err error) {
	mutexname := mutexes.key + util.Strval(id)
	mutex := mutexes.rs.NewMutex(mutexname)
	if lockErr := mutex.Lock(); lockErr != nil {
		return false, nil
	}
	return true, nil
}

func (mutexes *RedisMutexes) UnlockAll(ctx context.Context, ids []any) {
	for _, id := range ids {
		mutexname := mutexes.key + util.Strval(id)
		mutex := mutexes.rs.NewMutex(mutexname)
		mutex.Unlock()
	}
}

func NewRedisMutexes(client *redis.Client, key string) *RedisMutexes {
	pool := goredis.NewPool(client)
	rs := redsync.New(pool)
	return &RedisMutexes{key, rs}
}

type RedisQueryFuncs[T any] struct {
	client        *redis.Client
	key           string
	newZeroEntity arp.NewZeroEntity[T]
}

func NewRedisQueryFuncs[T any](client *redis.Client, key string, newZeroEntity arp.NewZeroEntity[T]) *RedisQueryFuncs[T] {
	return &RedisQueryFuncs[T]{client, key, newZeroEntity}
}

func (qf *RedisQueryFuncs[T]) QueryAllIds(ctx context.Context) (ids []any, err error) {

	var cursor uint64
	var keys []string
	for {
		var someKeys []string
		someKeys, cursor, err = qf.client.Scan(ctx, cursor, qf.key+"*", 100).Result()
		if err != nil {
			return nil, err
		}
		keys = append(keys, someKeys...)
		if cursor == 0 {
			break
		}
	}
	ids = make([]any, len(keys))
	for i, key := range keys {
		id := key[len(qf.key):]
		ids[i] = id
	}
	return
}

func (qf *RedisQueryFuncs[T]) Count(ctx context.Context) (uint64, error) {
	ids, err := qf.QueryAllIds(ctx)
	if err != nil {
		return 0, err
	}
	if ids == nil {
		return 0, nil
	}
	return uint64(len(ids)), nil
}

func NewRedisRepository[T any](client *redis.Client, key string, newEmptyEntity arp.NewZeroEntity[T]) arp.QueryRepository[T] {
	if client == nil {
		return arp.NewMockQueryRepository(newEmptyEntity)
	}
	mutexesimpl := NewRedisMutexes(client, key)
	return NewRedisRepositoryWithMutexesimpl(client, key, newEmptyEntity, mutexesimpl)
}

func NewRedisRepositoryWithMutexesimpl[T any](client *redis.Client, key string, newEmptyEntity arp.NewZeroEntity[T], mutexesimpl arp.Mutexes) arp.QueryRepository[T] {
	if client == nil {
		return arp.NewMockQueryRepository(newEmptyEntity)
	}
	store := NewRedisStore(client, key, newEmptyEntity)
	return arp.NewQueryRepository[T](arp.NewRepository[T](store, mutexesimpl, newEmptyEntity), NewRedisQueryFuncs(client, key, newEmptyEntity))
}
