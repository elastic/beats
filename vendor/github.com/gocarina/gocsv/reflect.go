package gocsv

import (
	"reflect"
	"strings"
	"sync"
)

// --------------------------------------------------------------------------
// Reflection helpers

type structInfo struct {
	Fields []fieldInfo
}

// fieldInfo is a struct field that should be mapped to a CSV column, or vica-versa
// Each IndexChain element before the last is the index of an the embedded struct field
// that defines Key as a tag
type fieldInfo struct {
	keys       []string
	IndexChain []int
}

func (f fieldInfo) getFirstKey() string {
	return f.keys[0]
}

func (f fieldInfo) matchesKey(key string) bool {
	for _, k := range f.keys {
		if key == k {
			return true
		}
	}
	return false
}

var structMap = make(map[reflect.Type]*structInfo)
var structMapMutex sync.RWMutex

func getStructInfo(rType reflect.Type) *structInfo {
	structMapMutex.RLock()
	stInfo, ok := structMap[rType]
	structMapMutex.RUnlock()
	if ok {
		return stInfo
	}
	fieldsList := getFieldInfos(rType, []int{})
	stInfo = &structInfo{fieldsList}
	return stInfo
}

func getFieldInfos(rType reflect.Type, parentIndexChain []int) []fieldInfo {
	fieldsCount := rType.NumField()
	fieldsList := make([]fieldInfo, 0, fieldsCount)
	for i := 0; i < fieldsCount; i++ {
		field := rType.Field(i)
		if field.PkgPath != "" {
			continue
		}
		indexChain := append(parentIndexChain, i)
		// if the field is an embedded struct, create a fieldInfo for each of its fields
		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			fieldsList = append(fieldsList, getFieldInfos(field.Type, indexChain)...)
			continue
		}
		fieldInfo := fieldInfo{IndexChain: indexChain}
		fieldTag := field.Tag.Get("csv")
		fieldTags := strings.Split(fieldTag, TagSeparator)
		filteredTags := []string{}
		for _, fieldTagEntry := range fieldTags {
			if fieldTagEntry != "omitempty" {
				filteredTags = append(filteredTags, fieldTagEntry)
			}
		}

		if len(filteredTags) == 1 && filteredTags[0] == "-" {
			continue
		} else if len(filteredTags) > 0 && filteredTags[0] != "" {
			fieldInfo.keys = filteredTags
		} else {
			fieldInfo.keys = []string{field.Name}
		}
		fieldsList = append(fieldsList, fieldInfo)
	}
	return fieldsList
}

func getConcreteContainerInnerType(in reflect.Type) (inInnerWasPointer bool, inInnerType reflect.Type) {
	inInnerType = in.Elem()
	inInnerWasPointer = false
	if inInnerType.Kind() == reflect.Ptr {
		inInnerWasPointer = true
		inInnerType = inInnerType.Elem()
	}
	return inInnerWasPointer, inInnerType
}

func getConcreteReflectValueAndType(in interface{}) (reflect.Value, reflect.Type) {
	value := reflect.ValueOf(in)
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}
	return value, value.Type()
}
