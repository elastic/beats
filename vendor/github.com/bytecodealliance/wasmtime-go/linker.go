package wasmtime

// #include <wasmtime.h>
// #include "shims.h"
import "C"
import (
	"reflect"
	"runtime"
)

// Linker implements a wasmtime Linking module, which can link instantiated modules together.
// More details you can see [examples for C](https://bytecodealliance.github.io/wasmtime/examples-c-linking.html) or
// [examples for Rust](https://bytecodealliance.github.io/wasmtime/examples-rust-linking.html)
type Linker struct {
	_ptr   *C.wasmtime_linker_t
	Engine *Engine
}

func NewLinker(engine *Engine) *Linker {
	ptr := C.wasmtime_linker_new(engine.ptr())
	linker := &Linker{_ptr: ptr, Engine: engine}
	runtime.SetFinalizer(linker, func(linker *Linker) {
		C.wasmtime_linker_delete(linker._ptr)
	})
	return linker
}

func (l *Linker) ptr() *C.wasmtime_linker_t {
	ret := l._ptr
	maybeGC()
	return ret
}

// AllowShadowing configures whether names can be redefined after they've already been defined
// in this linker.
func (l *Linker) AllowShadowing(allow bool) {
	C.wasmtime_linker_allow_shadowing(l.ptr(), C.bool(allow))
	runtime.KeepAlive(l)
}

// Define defines a new item in this linker with the given module/name pair. Returns
// an error if shadowing is disallowed and the module/name is already defined.
func (l *Linker) Define(module, name string, item AsExtern) error {
	extern := item.AsExtern()
	err := C.wasmtime_linker_define(
		l.ptr(),
		C._GoStringPtr(module),
		C._GoStringLen(module),
		C._GoStringPtr(name),
		C._GoStringLen(name),
		&extern,
	)
	runtime.KeepAlive(l)
	runtime.KeepAlive(module)
	runtime.KeepAlive(name)
	runtime.KeepAlive(item)
	if err == nil {
		return nil
	}

	return mkError(err)
}

// DefineFunc acts as a convenience wrapper to calling Define and WrapFunc.
//
// Returns an error if shadowing is disabled and the name is already defined.
func (l *Linker) DefineFunc(store Storelike, module, name string, f interface{}) error {
	return l.Define(module, name, WrapFunc(store, f))
}

// FuncNew defines a function in this linker in the same style as `NewFunc`
//
// Note that this function does not require a `Storelike`, which is
// intentional. This function can be used to insert store-independent functions
// into this linker which allows this linker to be used for instantiating
// modules in multiple different stores.
//
// Returns an error if shadowing is disabled and the name is already defined.
func (l *Linker) FuncNew(module, name string, ty *FuncType, f func(*Caller, []Val) ([]Val, *Trap)) error {
	idx := insertFuncNew(nil, ty, f)
	err := C.go_linker_define_func(
		l.ptr(),
		C._GoStringPtr(module),
		C._GoStringLen(module),
		C._GoStringPtr(name),
		C._GoStringLen(name),
		ty.ptr(),
		0, // this is "new"
		C.size_t(idx),
	)
	runtime.KeepAlive(l)
	runtime.KeepAlive(module)
	runtime.KeepAlive(name)
	runtime.KeepAlive(ty)
	if err == nil {
		return nil
	}

	return mkError(err)
}

// FuncWrap defines a function in this linker in the same style as `WrapFunc`
//
// Note that this function does not require a `Storelike`, which is
// intentional. This function can be used to insert store-independent functions
// into this linker which allows this linker to be used for instantiating
// modules in multiple different stores.
//
// Returns an error if shadowing is disabled and the name is already defined.
func (l *Linker) FuncWrap(module, name string, f interface{}) error {
	val := reflect.ValueOf(f)
	ty := inferFuncType(val)
	idx := insertFuncWrap(nil, val)
	err := C.go_linker_define_func(
		l.ptr(),
		C._GoStringPtr(module),
		C._GoStringLen(module),
		C._GoStringPtr(name),
		C._GoStringLen(name),
		ty.ptr(),
		1, // this is "wrap"
		C.size_t(idx),
	)
	runtime.KeepAlive(l)
	runtime.KeepAlive(module)
	runtime.KeepAlive(name)
	runtime.KeepAlive(ty)
	if err == nil {
		return nil
	}

	return mkError(err)
}

// DefineInstance defines all exports of an instance provided under the module name provided.
//
// Returns an error if shadowing is disabled and names are already defined.
func (l *Linker) DefineInstance(store Storelike, module string, instance *Instance) error {
	err := C.wasmtime_linker_define_instance(
		l.ptr(),
		store.Context(),
		C._GoStringPtr(module),
		C._GoStringLen(module),
		&instance.val,
	)
	runtime.KeepAlive(l)
	runtime.KeepAlive(module)
	runtime.KeepAlive(store)
	if err == nil {
		return nil
	}

	return mkError(err)
}

// DefineModule defines automatic instantiations of the module in this linker.
//
// The `name` of the module is the name within the linker, and the `module` is
// the one that's being instantiated. This function automatically handles
// WASI Commands and Reactors for instantiation and initialization. For more
// information see the Rust documentation --
// https://docs.wasmtime.dev/api/wasmtime/struct.Linker.html#method.module.
func (l *Linker) DefineModule(store Storelike, name string, module *Module) error {
	err := C.wasmtime_linker_module(
		l.ptr(),
		store.Context(),
		C._GoStringPtr(name),
		C._GoStringLen(name),
		module.ptr(),
	)
	runtime.KeepAlive(l)
	runtime.KeepAlive(name)
	runtime.KeepAlive(module)
	runtime.KeepAlive(store)
	if err == nil {
		return nil
	}

	return mkError(err)
}

// DefineWasi links a WASI module into this linker, ensuring that all exported functions
// are available for linking.
//
// Returns an error if shadowing is disabled and names are already defined.
func (l *Linker) DefineWasi() error {
	err := C.wasmtime_linker_define_wasi(l.ptr())
	runtime.KeepAlive(l)
	if err == nil {
		return nil
	}

	return mkError(err)
}

// Instantiate instantates a module with all imports defined in this linker.
//
// Returns an error if the instance's imports couldn't be satisfied, had the
// wrong types, or if a trap happened executing the start function.
func (l *Linker) Instantiate(store Storelike, module *Module) (*Instance, error) {
	var ret C.wasmtime_instance_t
	err := enterWasm(store, func(trap **C.wasm_trap_t) *C.wasmtime_error_t {
		return C.wasmtime_linker_instantiate(l.ptr(), store.Context(), module.ptr(), &ret, trap)
	})
	runtime.KeepAlive(l)
	runtime.KeepAlive(module)
	runtime.KeepAlive(store)
	if err != nil {
		return nil, err
	}
	return mkInstance(ret), nil
}

// GetDefault acquires the "default export" of the named module in this linker.
//
// If there is no default item then an error is returned, otherwise the default
// function is returned.
//
// For more information see the Rust documentation --
// https://docs.wasmtime.dev/api/wasmtime/struct.Linker.html#method.get_default.
func (l *Linker) GetDefault(store Storelike, name string) (*Func, error) {
	var ret C.wasmtime_func_t
	err := C.wasmtime_linker_get_default(
		l.ptr(),
		store.Context(),
		C._GoStringPtr(name),
		C._GoStringLen(name),
		&ret,
	)
	runtime.KeepAlive(l)
	runtime.KeepAlive(name)
	runtime.KeepAlive(store)
	if err != nil {
		return nil, mkError(err)
	}
	return mkFunc(ret), nil

}

// GetOneByName loads an item by name from this linker.
//
// If the item isn't defined then nil is returned, otherwise the item is
// returned.
func (l *Linker) Get(store Storelike, module, name string) *Extern {
	var ret C.wasmtime_extern_t
	ok := C.wasmtime_linker_get(
		l.ptr(),
		store.Context(),
		C._GoStringPtr(module),
		C._GoStringLen(module),
		C._GoStringPtr(name),
		C._GoStringLen(name),
		&ret,
	)
	runtime.KeepAlive(l)
	runtime.KeepAlive(name)
	runtime.KeepAlive(module)
	runtime.KeepAlive(store)
	if ok {
		return mkExtern(&ret)
	}
	return nil

}
