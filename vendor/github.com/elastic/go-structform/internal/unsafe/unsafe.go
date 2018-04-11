package unsafe

import (
	"reflect"
	"unsafe"
)

type emptyInterface struct {
	typ  unsafe.Pointer
	word unsafe.Pointer
}

func Str2Bytes(s string) []byte {
	sh := (*reflect.StringHeader)(unsafe.Pointer(&s))
	bh := reflect.SliceHeader{Data: sh.Data, Len: sh.Len, Cap: sh.Len}
	b := *(*[]byte)(unsafe.Pointer(&bh))
	return b
}

func Bytes2Str(b []byte) string {
	bh := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	sh := reflect.StringHeader{Data: bh.Data, Len: bh.Len}
	return *((*string)(unsafe.Pointer(&sh)))
}

// IfcValuePtr extracts the underlying values pointer from an empty interface{}
// value.
// Note: this might beome more unsafe in future go-versions,
// if primitive values < pointer size will be stored by value in the
// `interface{}` type.
func IfcValuePtr(v interface{}) unsafe.Pointer {
	ifc := (*emptyInterface)(unsafe.Pointer(&v))
	return ifc.word
}

// ReflValuePtr extracts the pointer value from a reflect.Value instance.
// With reflect.Value basically being similar to `interface{}` augmented with additional
// flags to execute checks, we map the value into an empty interface value (no methods)
// and extract the actual values pointer.
// Note: this might beome more unsafe in future go-versions,
// if primitive values < pointer size will be stored by value in the
// `interface{}` type.
func ReflValuePtr(v reflect.Value) unsafe.Pointer {
	ifc := (*emptyInterface)(unsafe.Pointer(&v))
	return ifc.word
}

// Returns a newly (allocated on heap) function pointer. The unsafe.Pointer returned
// can be used to cast a function type into a function with other(compatible)
// type (e.g. passing pointers only).
func UnsafeFnPtr(fn interface{}) unsafe.Pointer {
	var v reflect.Value
	if tmp, ok := fn.(reflect.Value); ok {
		v = tmp
	} else {
		v = reflect.ValueOf(fn)
	}

	tmp := reflect.New(v.Type())
	tmp.Elem().Set(v)
	return unsafe.Pointer(tmp.Pointer())
}
