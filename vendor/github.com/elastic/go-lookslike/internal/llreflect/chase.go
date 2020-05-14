package llreflect

import (
	"reflect"
)

// ChaseValue takes a value and returns the underlying type even if it is nested inpointers or wrapped in interface{}
func ChaseValue(v reflect.Value) reflect.Value {
	for (v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface) && !v.IsNil() {
		v = v.Elem()
	}
	return v
}
