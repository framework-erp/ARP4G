package arp

import (
	"ARP4G/copy"
	"reflect"
)

var repositories map[string]innerRepository = make(map[string]innerRepository)

var entityCopiers map[string]*copy.EntityCopier = make(map[string]*copy.EntityCopier)

var newZeroEntityFuncs map[string]newZeroEntity = make(map[string]newZeroEntity)

var singletonRepositories map[string]innerSingletonRepository = make(map[string]innerSingletonRepository)

func registerRepository[T any](repository *RepositoryImpl[T]) {
	repositories[repository.entityType] = repository
}

func getRepository(typeFullname string) innerRepository {
	return repositories[typeFullname]
}

func generateEntityCopier[T any](typeFullname string, entityType reflect.Type, newZeroEntityFunc NewZeroEntity[T]) {
	newZeroEntityFuncs[typeFullname] = func() any {
		return newZeroEntityFunc()
	}
	copy.GenerateEntityCopier(entityType, entityCopiers)
}

//完整复制一个实体（深拷贝），这里约定，实体只能是一个struct，实体的field只能是基本类型或者实体或者集合（Array，Map，Slice），集合的元素只能是基本类型或者实体
func CopyEntity(typeFullname string, entity any) any {
	newEntity := newZeroEntityFuncs[typeFullname]()
	entityCopiers[typeFullname].Copy(entity, newEntity)
	return newEntity
}

type newZeroEntity func() any

func registerSingletonRepository[T any](repository *SingletonRepositoryImpl[T]) {
	singletonRepositories[repository.entityType] = repository
}

func getSingletonRepository(typeFullname string) innerSingletonRepository {
	return singletonRepositories[typeFullname]
}
