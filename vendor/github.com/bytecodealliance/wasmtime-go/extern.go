package wasmtime

// #include "shims.h"
import "C"
import "runtime"

// Extern is an external value, which is the runtime representation of an entity that can be imported or exported.
// It is an address denoting either a function instance, table instance, memory instance, or global instances in the shared store.
// Read more in [spec](https://webassembly.github.io/spec/core/exec/runtime.html#external-values)
//
type Extern struct {
	_ptr *C.wasmtime_extern_t
}

// AsExtern is an interface for all types which can be imported or exported as an Extern
type AsExtern interface {
	AsExtern() C.wasmtime_extern_t
}

func mkExtern(ptr *C.wasmtime_extern_t) *Extern {
	f := &Extern{_ptr: ptr}
	runtime.SetFinalizer(f, func(f *Extern) {
		C.wasmtime_extern_delete(f._ptr)
	})
	return f
}

func (e *Extern) ptr() *C.wasmtime_extern_t {
	ret := e._ptr
	maybeGC()
	return ret
}

// Type returns the type of this export
func (e *Extern) Type(store Storelike) *ExternType {
	ptr := C.wasmtime_extern_type(store.Context(), e.ptr())
	runtime.KeepAlive(e)
	runtime.KeepAlive(store)
	return mkExternType(ptr, nil)
}

// Func returns a Func if this export is a function or nil otherwise
func (e *Extern) Func() *Func {
	ptr := e.ptr()
	if ptr.kind != C.WASMTIME_EXTERN_FUNC {
		return nil
	}
	ret := mkFunc(C.go_wasmtime_extern_func_get(ptr))
	runtime.KeepAlive(e)
	return ret
}

// Global returns a Global if this export is a global or nil otherwise
func (e *Extern) Global() *Global {
	ptr := e.ptr()
	if ptr.kind != C.WASMTIME_EXTERN_GLOBAL {
		return nil
	}
	ret := mkGlobal(C.go_wasmtime_extern_global_get(ptr))
	runtime.KeepAlive(e)
	return ret
}

// Memory returns a Memory if this export is a memory or nil otherwise
func (e *Extern) Memory() *Memory {
	ptr := e.ptr()
	if ptr.kind != C.WASMTIME_EXTERN_MEMORY {
		return nil
	}
	ret := mkMemory(C.go_wasmtime_extern_memory_get(ptr))
	runtime.KeepAlive(e)
	return ret
}

// Table returns a Table if this export is a table or nil otherwise
func (e *Extern) Table() *Table {
	ptr := e.ptr()
	if ptr.kind != C.WASMTIME_EXTERN_TABLE {
		return nil
	}
	ret := mkTable(C.go_wasmtime_extern_table_get(ptr))
	runtime.KeepAlive(e)
	return ret
}

// Module returns a Module if this export is a module or nil otherwise
func (e *Extern) Module() *Module {
	ptr := e.ptr()
	if ptr.kind != C.WASMTIME_EXTERN_MODULE {
		return nil
	}
	module := C.go_wasmtime_extern_module_get(ptr)
	ret := mkModule(C.wasmtime_module_clone(module))
	runtime.KeepAlive(e)
	return ret
}

// Instance returns a Instance if this export is a module or nil otherwise
func (e *Extern) Instance() *Instance {
	ptr := e.ptr()
	if ptr.kind != C.WASMTIME_EXTERN_INSTANCE {
		return nil
	}
	ret := mkInstance(C.go_wasmtime_extern_instance_get(ptr))
	runtime.KeepAlive(e)
	return ret
}

func (e *Extern) AsExtern() C.wasmtime_extern_t {
	return *e.ptr()
}
