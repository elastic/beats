package wasmtime

// #include <wasm.h>
// #include "shims.h"
import "C"
import (
	"runtime"
	"sync"
	"unsafe"
)

var gExternrefLock sync.Mutex
var gExternrefMap = make(map[int]interface{})
var gExternrefSlab slab

// Val is a primitive numeric value.
// Moreover, in the definition of programs, immutable sequences of values occur to represent more complex data, such as text strings or other vectors.
type Val struct {
	_raw *C.wasmtime_val_t
}

// ValI32 converts a go int32 to a i32 Val
func ValI32(val int32) Val {
	ret := Val{_raw: &C.wasmtime_val_t{kind: C.WASMTIME_I32}}
	C.go_wasmtime_val_i32_set(ret.ptr(), C.int32_t(val))
	return ret
}

// ValI64 converts a go int64 to a i64 Val
func ValI64(val int64) Val {
	ret := Val{_raw: &C.wasmtime_val_t{kind: C.WASMTIME_I64}}
	C.go_wasmtime_val_i64_set(ret.ptr(), C.int64_t(val))
	return ret
}

// ValF32 converts a go float32 to a f32 Val
func ValF32(val float32) Val {
	ret := Val{_raw: &C.wasmtime_val_t{kind: C.WASMTIME_F32}}
	C.go_wasmtime_val_f32_set(ret.ptr(), C.float(val))
	return ret
}

// ValF64 converts a go float64 to a f64 Val
func ValF64(val float64) Val {
	ret := Val{_raw: &C.wasmtime_val_t{kind: C.WASMTIME_F64}}
	C.go_wasmtime_val_f64_set(ret.ptr(), C.double(val))
	return ret
}

// ValFuncref converts a Func to a funcref Val
//
// Note that `f` can be `nil` to represent a null `funcref`.
func ValFuncref(f *Func) Val {
	ret := Val{_raw: &C.wasmtime_val_t{kind: C.WASMTIME_FUNCREF}}
	if f != nil {
		C.go_wasmtime_val_funcref_set(ret.ptr(), f.val)
	}
	return ret
}

// ValExternref converts a go value to a externref Val
//
// Using `externref` is a way to pass arbitrary Go data into a WebAssembly
// module for it to store. Later, when you get a `Val`, you can extract the type
// with the `Externref()` method.
func ValExternref(val interface{}) Val {
	ret := Val{_raw: &C.wasmtime_val_t{kind: C.WASMTIME_EXTERNREF}}

	// If we have a non-nil value then store it in our global map of all
	// externref values. Otherwise there's nothing for us to do since the
	// `ref` field will already be a nil pointer.
	//
	// Note that we add 1 so all non-null externref values are created with
	// non-null pointers.
	if val != nil {
		gExternrefLock.Lock()
		defer gExternrefLock.Unlock()
		index := gExternrefSlab.allocate()
		gExternrefMap[index] = val
		ptr := C.go_externref_new(C.size_t(index + 1))
		C.go_wasmtime_val_externref_set(ret.ptr(), ptr)
		ret.setDtor()
	}
	return ret
}

//export goFinalizeExternref
func goFinalizeExternref(env unsafe.Pointer) {
	idx := int(uintptr(env)) - 1
	gExternrefLock.Lock()
	defer gExternrefLock.Unlock()
	delete(gExternrefMap, idx)
	gExternrefSlab.deallocate(idx)
}

func mkVal(src *C.wasmtime_val_t) Val {
	ret := Val{_raw: &C.wasmtime_val_t{}}
	C.wasmtime_val_copy(ret.ptr(), src)
	ret.setDtor()
	return ret
}

func takeVal(src *C.wasmtime_val_t) Val {
	ret := Val{_raw: &C.wasmtime_val_t{}}
	*ret.ptr() = *src
	ret.setDtor()
	return ret
}

func (v Val) setDtor() {
	runtime.SetFinalizer(v.ptr(), func(ptr *C.wasmtime_val_t) {
		C.wasmtime_val_delete(ptr)
	})
}

func (v Val) ptr() *C.wasmtime_val_t {
	ret := v._raw
	maybeGC()
	return ret
}

// Kind returns the kind of value that this `Val` contains.
func (v Val) Kind() ValKind {
	switch v.ptr().kind {
	case C.WASMTIME_I32:
		return KindI32
	case C.WASMTIME_I64:
		return KindI64
	case C.WASMTIME_F32:
		return KindF32
	case C.WASMTIME_F64:
		return KindF64
	case C.WASMTIME_FUNCREF:
		return KindFuncref
	case C.WASMTIME_EXTERNREF:
		return KindExternref
	}
	panic("failed to get kind of `Val`")
}

// I32 returns the underlying 32-bit integer if this is an `i32`, or panics.
func (v Val) I32() int32 {
	if v.Kind() != KindI32 {
		panic("not an i32")
	}
	return int32(C.go_wasmtime_val_i32_get(v.ptr()))
}

// I64 returns the underlying 64-bit integer if this is an `i64`, or panics.
func (v Val) I64() int64 {
	if v.Kind() != KindI64 {
		panic("not an i64")
	}
	return int64(C.go_wasmtime_val_i64_get(v.ptr()))
}

// F32 returns the underlying 32-bit float if this is an `f32`, or panics.
func (v Val) F32() float32 {
	if v.Kind() != KindF32 {
		panic("not an f32")
	}
	return float32(C.go_wasmtime_val_f32_get(v.ptr()))
}

// F64 returns the underlying 64-bit float if this is an `f64`, or panics.
func (v Val) F64() float64 {
	if v.Kind() != KindF64 {
		panic("not an f64")
	}
	return float64(C.go_wasmtime_val_f64_get(v.ptr()))
}

// Funcref returns the underlying function if this is a `funcref`, or panics.
//
// Note that a null `funcref` is returned as `nil`.
func (v Val) Funcref() *Func {
	if v.Kind() != KindFuncref {
		panic("not a funcref")
	}
	val := C.go_wasmtime_val_funcref_get(v.ptr())
	if val.store_id == 0 {
		return nil
	} else {
		return mkFunc(val)
	}
}

// Externref returns the underlying value if this is an `externref`, or panics.
//
// Note that a null `externref` is returned as `nil`.
func (v Val) Externref() interface{} {
	if v.Kind() != KindExternref {
		panic("not an externref")
	}
	val := C.go_wasmtime_val_externref_get(v.ptr())
	if val == nil {
		return nil
	}
	data := C.wasmtime_externref_data(val)

	gExternrefLock.Lock()
	defer gExternrefLock.Unlock()
	return gExternrefMap[int(uintptr(data))-1]
}

// Get returns the underlying 64-bit float if this is an `f64`, or panics.
func (v Val) Get() interface{} {
	switch v.Kind() {
	case KindI32:
		return v.I32()
	case KindI64:
		return v.I64()
	case KindF32:
		return v.F32()
	case KindF64:
		return v.F64()
	case KindFuncref:
		return v.Funcref()
	case KindExternref:
		return v.Externref()
	}
	panic("failed to get value of `Val`")
}
