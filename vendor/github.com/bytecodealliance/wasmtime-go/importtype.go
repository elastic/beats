package wasmtime

// #include <wasm.h>
import "C"
import "runtime"

// ImportType is one of the imports component
// A module defines a set of imports that are required for instantiation.
type ImportType struct {
	_ptr   *C.wasm_importtype_t
	_owner interface{}
}

// NewImportType creates a new `ImportType` with the given `module` and `name` and the type
// provided.
func NewImportType(module, name string, ty AsExternType) *ImportType {
	moduleVec := stringToByteVec(module)
	nameVec := stringToByteVec(name)

	// Creating an import type requires taking ownership, so create a copy
	// so we don't have to invalidate pointers here. Shouldn't be too
	// costly in theory anyway.
	extern := ty.AsExternType()
	ptr := C.wasm_externtype_copy(extern.ptr())
	runtime.KeepAlive(extern)

	// And once we've got all that create the import type!
	importPtr := C.wasm_importtype_new(&moduleVec, &nameVec, ptr)

	return mkImportType(importPtr, nil)
}

func mkImportType(ptr *C.wasm_importtype_t, owner interface{}) *ImportType {
	importtype := &ImportType{_ptr: ptr, _owner: owner}
	if owner == nil {
		runtime.SetFinalizer(importtype, func(importtype *ImportType) {
			C.wasm_importtype_delete(importtype._ptr)
		})
	}
	return importtype
}

func (ty *ImportType) ptr() *C.wasm_importtype_t {
	ret := ty._ptr
	maybeGC()
	return ret
}

func (ty *ImportType) owner() interface{} {
	if ty._owner != nil {
		return ty._owner
	}
	return ty
}

// Module returns the name in the module this import type is importing
func (ty *ImportType) Module() string {
	ptr := C.wasm_importtype_module(ty.ptr())
	ret := C.GoStringN(ptr.data, C.int(ptr.size))
	runtime.KeepAlive(ty)
	return ret
}

// Name returns the name in the module this import type is importing.
//
// Note that the returned string may be `nil` with the module linking proposal
// where this field is optional in the import type.
func (ty *ImportType) Name() *string {
	ptr := C.wasm_importtype_name(ty.ptr())
	if ptr == nil {
		return nil
	}
	ret := C.GoStringN(ptr.data, C.int(ptr.size))
	runtime.KeepAlive(ty)
	return &ret
}

// Type returns the type of item this import type expects
func (ty *ImportType) Type() *ExternType {
	ptr := C.wasm_importtype_type(ty.ptr())
	return mkExternType(ptr, ty.owner())
}
