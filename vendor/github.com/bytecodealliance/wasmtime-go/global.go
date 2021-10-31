package wasmtime

// #include "shims.h"
import "C"
import "runtime"

// Global is a global instance, which is the runtime representation of a global variable.
// It holds an individual value and a flag indicating whether it is mutable.
// Read more in [spec](https://webassembly.github.io/spec/core/exec/runtime.html#global-instances)
type Global struct {
	val C.wasmtime_global_t
}

// NewGlobal creates a new `Global` in the given `Store` with the specified `ty` and
// initial value `val`.
func NewGlobal(
	store Storelike,
	ty *GlobalType,
	val Val,
) (*Global, error) {
	var ret C.wasmtime_global_t
	err := C.wasmtime_global_new(
		store.Context(),
		ty.ptr(),
		val.ptr(),
		&ret,
	)
	runtime.KeepAlive(store)
	runtime.KeepAlive(ty)
	runtime.KeepAlive(val)
	if err != nil {
		return nil, mkError(err)
	}

	return mkGlobal(ret), nil
}

func mkGlobal(val C.wasmtime_global_t) *Global {
	return &Global{val}
}

// Type returns the type of this global
func (g *Global) Type(store Storelike) *GlobalType {
	ptr := C.wasmtime_global_type(store.Context(), &g.val)
	runtime.KeepAlive(store)
	return mkGlobalType(ptr, nil)
}

// Get gets the value of this global
func (g *Global) Get(store Storelike) Val {
	ret := C.wasmtime_val_t{}
	C.wasmtime_global_get(store.Context(), &g.val, &ret)
	runtime.KeepAlive(store)
	return takeVal(&ret)
}

// Set sets the value of this global
func (g *Global) Set(store Storelike, val Val) error {
	err := C.wasmtime_global_set(store.Context(), &g.val, val.ptr())
	runtime.KeepAlive(store)
	runtime.KeepAlive(val)
	if err == nil {
		return nil
	}

	return mkError(err)
}

func (g *Global) AsExtern() C.wasmtime_extern_t {
	ret := C.wasmtime_extern_t{kind: C.WASMTIME_EXTERN_GLOBAL}
	C.go_wasmtime_extern_global_set(&ret, g.val)
	return ret
}
