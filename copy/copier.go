package copy

import (
	"reflect"
)

type EntityCopier struct {
	FieldDeepCopiers []FieldDeepCopier
}

func (copier *EntityCopier) Copy(sourceEntityPtrAny, destEntityPtrAny any) {
	sourceEntityValue := reflect.ValueOf(sourceEntityPtrAny).Elem()
	destEntityValue := reflect.ValueOf(destEntityPtrAny).Elem()
	destEntityValue.Set(sourceEntityValue)
	copier.DeepCopyFields(sourceEntityValue, destEntityValue)
}

func (copier *EntityCopier) DeepCopyFields(sourceEntityValue, destEntityValue reflect.Value) {
	for _, fieldDeepCopier := range copier.FieldDeepCopiers {
		fieldDeepCopier.copyField(sourceEntityValue, destEntityValue)
	}
}

type FieldDeepCopier interface {
	copyField(sourceEntityValue, destEntityValue reflect.Value)
}

type StructFieldDeepCopier struct {
	FieldIndex        int
	FieldEntityCopier *EntityCopier
}

func (copier *StructFieldDeepCopier) copyField(sourceEntityValue, destEntityValue reflect.Value) {
	copier.FieldEntityCopier.DeepCopyFields(sourceEntityValue.Field(copier.FieldIndex), destEntityValue.Field(copier.FieldIndex))
}

type StructPtrFieldDeepCopier struct {
	FieldIndex        int
	FieldEntityType   reflect.Type
	FieldEntityCopier *EntityCopier
}

func (copier *StructPtrFieldDeepCopier) copyField(sourceEntityValue, destEntityValue reflect.Value) {
	newFieldEntityPtr := reflect.New(copier.FieldEntityType)
	newFieldEntity := newFieldEntityPtr.Elem()
	sourceFieldEntity := sourceEntityValue.Field(copier.FieldIndex).Elem()
	destEntityValue.Field(copier.FieldIndex).Set(newFieldEntityPtr)
	newFieldEntity.Set(sourceFieldEntity)
	copier.FieldEntityCopier.DeepCopyFields(sourceFieldEntity, newFieldEntity)
}

type StructArrayFieldDeepCopier struct {
	FieldIndex          int
	ElementEntityCopier *EntityCopier
}

func (copier *StructArrayFieldDeepCopier) copyField(sourceEntityValue, destEntityValue reflect.Value) {
	sourceFieldArray := sourceEntityValue.Field(copier.FieldIndex)
	destFieldArray := destEntityValue.Field(copier.FieldIndex)
	len := sourceFieldArray.Len()
	for i := 0; i < len; i++ {
		copier.ElementEntityCopier.DeepCopyFields(sourceFieldArray.Index(i), destFieldArray.Index(i))
	}
}

type StructPtrArrayFieldDeepCopier struct {
	FieldIndex          int
	ElementEntityType   reflect.Type
	ElementEntityCopier *EntityCopier
}

func (copier *StructPtrArrayFieldDeepCopier) copyField(sourceEntityValue, destEntityValue reflect.Value) {
	sourceFieldArray := sourceEntityValue.Field(copier.FieldIndex)
	destFieldArray := destEntityValue.Field(copier.FieldIndex)
	len := sourceFieldArray.Len()
	for i := 0; i < len; i++ {
		newElementEntityPtr := reflect.New(copier.ElementEntityType)
		destFieldArray.Index(i).Set(newElementEntityPtr)
		copier.ElementEntityCopier.DeepCopyFields(sourceFieldArray.Index(i).Elem(), newElementEntityPtr.Elem())
	}
}

type SimpleSliceFieldDeepCopier struct {
	FieldIndex int
	SliceType  reflect.Type
}

func (copier *SimpleSliceFieldDeepCopier) copyField(sourceEntityValue, destEntityValue reflect.Value) {
	sourceFieldSlice := sourceEntityValue.Field(copier.FieldIndex)
	len := sourceFieldSlice.Len()
	newSlice := reflect.MakeSlice(copier.SliceType, len, sourceFieldSlice.Cap())
	destEntityValue.Field(copier.FieldIndex).Set(newSlice)
	for i := 0; i < len; i++ {
		newSlice.Index(i).Set(sourceFieldSlice.Index(i))
	}
}

type StructSliceFieldDeepCopier struct {
	FieldIndex          int
	SliceType           reflect.Type
	ElementEntityCopier *EntityCopier
}

func (copier *StructSliceFieldDeepCopier) copyField(sourceEntityValue, destEntityValue reflect.Value) {
	sourceFieldSlice := sourceEntityValue.Field(copier.FieldIndex)
	len := sourceFieldSlice.Len()
	newSlice := reflect.MakeSlice(copier.SliceType, len, sourceFieldSlice.Cap())
	destEntityValue.Field(copier.FieldIndex).Set(newSlice)
	for i := 0; i < len; i++ {
		sourceEntityElement := sourceFieldSlice.Index(i)
		newEntityElement := newSlice.Index(i)
		newEntityElement.Set(sourceEntityElement)
		copier.ElementEntityCopier.DeepCopyFields(sourceEntityElement, newEntityElement)
	}
}

type StructPtrSliceFieldDeepCopier struct {
	FieldIndex          int
	SliceType           reflect.Type
	ElementEntityCopier *EntityCopier
}

func (copier *StructPtrSliceFieldDeepCopier) copyField(sourceEntityValue, destEntityValue reflect.Value) {
	sourceFieldSlice := sourceEntityValue.Field(copier.FieldIndex)
	len := sourceFieldSlice.Len()
	newSlice := reflect.MakeSlice(copier.SliceType, len, sourceFieldSlice.Cap())
	destEntityValue.Field(copier.FieldIndex).Set(newSlice)
	elementEntityType := copier.SliceType.Elem().Elem()
	for i := 0; i < len; i++ {
		sourceEntityElement := sourceFieldSlice.Index(i).Elem()
		newEntityPtrElement := reflect.New(elementEntityType)
		newEntityElement := newEntityPtrElement.Elem()
		newEntityElement.Set(sourceEntityElement)
		copier.ElementEntityCopier.DeepCopyFields(sourceEntityElement, newEntityElement)
		newSlice.Index(i).Set(newEntityPtrElement)
	}
}

type SimpleMapFieldDeepCopier struct {
	FieldIndex int
	MapType    reflect.Type
}

func (copier *SimpleMapFieldDeepCopier) copyField(sourceEntityValue, destEntityValue reflect.Value) {
	sourceFieldMap := sourceEntityValue.Field(copier.FieldIndex)
	if sourceFieldMap.IsNil() {
		return
	}
	newMap := reflect.MakeMap(copier.MapType)
	destEntityValue.Field(copier.FieldIndex).Set(newMap)
	keys := sourceFieldMap.MapKeys()
	for _, k := range keys {
		value := sourceFieldMap.MapIndex(k)
		newMap.SetMapIndex(k, value)
	}
}

type StructMapFieldDeepCopier struct {
	FieldIndex          int
	MapType             reflect.Type
	ElementEntityCopier *EntityCopier
}

func (copier *StructMapFieldDeepCopier) copyField(sourceEntityValue, destEntityValue reflect.Value) {
	sourceFieldMap := sourceEntityValue.Field(copier.FieldIndex)
	if sourceFieldMap.IsNil() {
		return
	}
	newMap := reflect.MakeMap(copier.MapType)
	destEntityValue.Field(copier.FieldIndex).Set(newMap)
	keys := sourceFieldMap.MapKeys()
	elementType := copier.MapType.Elem()
	for _, k := range keys {
		value := sourceFieldMap.MapIndex(k)
		newValue := reflect.New(elementType).Elem()
		newValue.Set(value)
		copier.ElementEntityCopier.DeepCopyFields(value, newValue)
		newMap.SetMapIndex(k, newValue)
	}
}

type StructPtrMapFieldDeepCopier struct {
	FieldIndex          int
	MapType             reflect.Type
	ElementEntityCopier *EntityCopier
}

func (copier *StructPtrMapFieldDeepCopier) copyField(sourceEntityValue, destEntityValue reflect.Value) {
	sourceFieldMap := sourceEntityValue.Field(copier.FieldIndex)
	if sourceFieldMap.IsNil() {
		return
	}
	newMap := reflect.MakeMap(copier.MapType)
	destEntityValue.Field(copier.FieldIndex).Set(newMap)
	keys := sourceFieldMap.MapKeys()
	elementType := copier.MapType.Elem().Elem()
	for _, k := range keys {
		entity := sourceFieldMap.MapIndex(k).Elem()
		newEntityPtr := reflect.New(elementType)
		newEntity := newEntityPtr.Elem()
		newEntity.Set(entity)
		copier.ElementEntityCopier.DeepCopyFields(entity, newEntity)
		newMap.SetMapIndex(k, newEntityPtr)
	}
}

func GenerateEntityCopier(entityType reflect.Type, entityCopiers map[string]*EntityCopier) *EntityCopier {
	typeFullname := entityType.PkgPath() + "." + entityType.Name()
	if entityCopiers[typeFullname] != nil {
		return entityCopiers[typeFullname]
	}
	numField := entityType.NumField()
	fieldDeepCopiers := make([]FieldDeepCopier, 0, numField)
	for i := 0; i < numField; i++ {
		field := entityType.Field(i)
		fieldDeepCopier := generateFieldDeepCopier(i, field.Type, entityCopiers)
		if fieldDeepCopier != nil {
			fieldDeepCopiers = append(fieldDeepCopiers, fieldDeepCopier)
		}
	}
	entityCopier := EntityCopier{fieldDeepCopiers}
	entityCopierPtr := &entityCopier
	entityCopiers[typeFullname] = entityCopierPtr
	return entityCopierPtr
}

func generateFieldDeepCopier(fieldIndex int, fieldType reflect.Type, entityCopiers map[string]*EntityCopier) FieldDeepCopier {
	fieldTypeKind := fieldType.Kind()
	if fieldTypeKind == reflect.Map {
		elementTypeKind := fieldType.Elem().Kind()
		if elementTypeKind == reflect.Struct {
			entityCopier := GenerateEntityCopier(fieldType.Elem(), entityCopiers)
			if len(entityCopier.FieldDeepCopiers) == 0 {
				return &SimpleMapFieldDeepCopier{fieldIndex, fieldType}
			}
			return &StructMapFieldDeepCopier{fieldIndex, fieldType, entityCopier}
		} else if elementTypeKind == reflect.Pointer {
			pointToType := fieldType.Elem().Elem()
			if pointToType.Kind() == reflect.Struct {
				entityCopier := GenerateEntityCopier(pointToType, entityCopiers)
				return &StructPtrMapFieldDeepCopier{fieldIndex, fieldType, entityCopier}
			} else {
				return &SimpleMapFieldDeepCopier{fieldIndex, fieldType}
			}
		} else {
			return &SimpleMapFieldDeepCopier{fieldIndex, fieldType}
		}
	} else if fieldTypeKind == reflect.Struct {
		entityCopier := GenerateEntityCopier(fieldType, entityCopiers)
		if len(entityCopier.FieldDeepCopiers) == 0 {
			return nil
		}
		return &StructFieldDeepCopier{fieldIndex, entityCopier}
	} else if fieldTypeKind == reflect.Array {
		elementTypeKind := fieldType.Elem().Kind()
		if elementTypeKind == reflect.Struct {
			entityCopier := GenerateEntityCopier(fieldType.Elem(), entityCopiers)
			if len(entityCopier.FieldDeepCopiers) == 0 {
				return nil
			}
			return &StructArrayFieldDeepCopier{fieldIndex, entityCopier}
		} else if elementTypeKind == reflect.Pointer {
			pointToTypeKind := fieldType.Elem().Elem().Kind()
			if pointToTypeKind == reflect.Struct {
				entityCopier := GenerateEntityCopier(fieldType.Elem().Elem(), entityCopiers)
				return &StructPtrArrayFieldDeepCopier{fieldIndex, fieldType.Elem(), entityCopier}
			} else {
				return nil
			}
		} else {
			return nil
		}
	} else if fieldTypeKind == reflect.Slice {
		elementTypeKind := fieldType.Elem().Kind()
		if elementTypeKind == reflect.Struct {
			entityCopier := GenerateEntityCopier(fieldType.Elem(), entityCopiers)
			if len(entityCopier.FieldDeepCopiers) == 0 {
				return &SimpleSliceFieldDeepCopier{fieldIndex, fieldType}
			}
			return &StructSliceFieldDeepCopier{fieldIndex, fieldType, entityCopier}
		} else if elementTypeKind == reflect.Pointer {
			pointToType := fieldType.Elem().Elem()
			if pointToType.Kind() == reflect.Struct {
				entityCopier := GenerateEntityCopier(pointToType, entityCopiers)
				return &StructPtrSliceFieldDeepCopier{fieldIndex, fieldType, entityCopier}
			} else {
				return &SimpleSliceFieldDeepCopier{fieldIndex, fieldType}
			}
		} else {
			return &SimpleSliceFieldDeepCopier{fieldIndex, fieldType}
		}
	} else if fieldTypeKind == reflect.Pointer {
		pointToTypeKind := fieldType.Elem().Kind()
		if pointToTypeKind == reflect.Struct {
			entityCopier := GenerateEntityCopier(fieldType.Elem(), entityCopiers)
			return &StructPtrFieldDeepCopier{fieldIndex, fieldType.Elem(), entityCopier}
		} else {
			return nil
		}
	} else {
		return nil
	}
}
