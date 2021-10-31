package wasmtime

// #include "shims.h"
import "C"
import (
	"runtime"
	"unsafe"
)

// Instance is an instantiated module instance.
// Once a module has been instantiated as an Instance, any exported function can be invoked externally via its function address funcaddr in the store S and an appropriate list val∗ of argument values.
type Instance struct {
	val C.wasmtime_instance_t
}

// NewInstance instantiates a WebAssembly `module` with the `imports` provided.
//
// This function will attempt to create a new wasm instance given the provided
// imports. This can fail if the wrong number of imports are specified, the
// imports aren't of the right type, or for other resource-related issues.
//
// This will also run the `start` function of the instance, returning an error
// if it traps.
func NewInstance(store Storelike, module *Module, imports []AsExtern) (*Instance, error) {
	importsRaw := make([]C.wasmtime_extern_t, len(imports), len(imports))
	for i, imp := range imports {
		importsRaw[i] = imp.AsExtern()
	}
	var val C.wasmtime_instance_t
	err := enterWasm(store, func(trap **C.wasm_trap_t) *C.wasmtime_error_t {
		var imports *C.wasmtime_extern_t
		if len(importsRaw) > 0 {
			imports = (*C.wasmtime_extern_t)(unsafe.Pointer(&importsRaw[0]))
		}
		return C.wasmtime_instance_new(
			store.Context(),
			module.ptr(),
			imports,
			C.size_t(len(importsRaw)),
			&val,
			trap,
		)
	})
	runtime.KeepAlive(store)
	runtime.KeepAlive(module)
	runtime.KeepAlive(imports)
	runtime.KeepAlive(importsRaw)
	if err != nil {
		return nil, err
	}
	return mkInstance(val), nil
}

func mkInstance(val C.wasmtime_instance_t) *Instance {
	return &Instance{val}
}

// Type returns an `InstanceType` that corresponds for this instance.
func (i *Instance) Type(store Storelike) *InstanceType {
	ptr := C.wasmtime_instance_type(store.Context(), &i.val)
	runtime.KeepAlive(store)
	return mkInstanceType(ptr, nil)
}

type externList struct {
	vec C.wasm_extern_vec_t
}

// Exports returns a list of exports from this instance.
//
// Each export is returned as a `*Extern` and lines up with the exports list of
// the associated `Module`.
func (instance *Instance) Exports(store Storelike) []*Extern {
	ret := make([]*Extern, 0)
	var name *C.char
	var name_len C.size_t
	for i := 0; ; i++ {
		var item C.wasmtime_extern_t
		ok := C.wasmtime_instance_export_nth(
			store.Context(),
			&instance.val,
			C.size_t(i),
			&name,
			&name_len,
			&item,
		)
		if !ok {
			break
		}
		ret = append(ret, mkExtern(&item))
	}
	runtime.KeepAlive(store)
	return ret
}

// GetExport attempts to find an export on this instance by `name`
//
// May return `nil` if this instance has no export named `name`
func (i *Instance) GetExport(store Storelike, name string) *Extern {
	var item C.wasmtime_extern_t
	ok := C.wasmtime_instance_export_get(
		store.Context(),
		&i.val,
		C._GoStringPtr(name),
		C._GoStringLen(name),
		&item,
	)
	runtime.KeepAlive(store)
	runtime.KeepAlive(name)
	if ok {
		return mkExtern(&item)
	}
	return nil
}

// GetFunc attemps to find a function on this instance by `name`.
//
// May return `nil` if this instance has no function named `name`,
// it is not a function, etc.
func (i *Instance) GetFunc(store Storelike, name string) *Func {
	f := i.GetExport(store, name)
	if f == nil {
		return nil
	}
	return f.Func()
}

func (i *Instance) AsExtern() C.wasmtime_extern_t {
	ret := C.wasmtime_extern_t{kind: C.WASMTIME_EXTERN_INSTANCE}
	C.go_wasmtime_extern_instance_set(&ret, i.val)
	return ret
}
