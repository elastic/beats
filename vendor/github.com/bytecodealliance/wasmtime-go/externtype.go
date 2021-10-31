package wasmtime

// #include <wasm.h>
import "C"
import "runtime"

// ExternType means one of external types which classify imports and external values with their respective types.
type ExternType struct {
	_ptr   *C.wasm_externtype_t
	_owner interface{}
}

// AsExternType is an interface for all types which can be ExternType.
type AsExternType interface {
	AsExternType() *ExternType
}

func mkExternType(ptr *C.wasm_externtype_t, owner interface{}) *ExternType {
	externtype := &ExternType{_ptr: ptr, _owner: owner}
	if owner == nil {
		runtime.SetFinalizer(externtype, func(externtype *ExternType) {
			C.wasm_externtype_delete(externtype._ptr)
		})
	}
	return externtype
}

func (ty *ExternType) ptr() *C.wasm_externtype_t {
	ret := ty._ptr
	maybeGC()
	return ret
}

func (ty *ExternType) owner() interface{} {
	if ty._owner != nil {
		return ty._owner
	}
	return ty
}

// FuncType returns the underlying `FuncType` for this `ExternType` if it's a function
// type. Otherwise returns `nil`.
func (ty *ExternType) FuncType() *FuncType {
	ptr := C.wasm_externtype_as_functype(ty.ptr())
	if ptr == nil {
		return nil
	}
	return mkFuncType(ptr, ty.owner())
}

// GlobalType returns the underlying `GlobalType` for this `ExternType` if it's a *global* type.
// Otherwise returns `nil`.
func (ty *ExternType) GlobalType() *GlobalType {
	ptr := C.wasm_externtype_as_globaltype(ty.ptr())
	if ptr == nil {
		return nil
	}
	return mkGlobalType(ptr, ty.owner())
}

// TableType returns the underlying `TableType` for this `ExternType` if it's a *table* type.
// Otherwise returns `nil`.
func (ty *ExternType) TableType() *TableType {
	ptr := C.wasm_externtype_as_tabletype(ty.ptr())
	if ptr == nil {
		return nil
	}
	return mkTableType(ptr, ty.owner())
}

// MemoryType returns the underlying `MemoryType` for this `ExternType` if it's a *memory* type.
// Otherwise returns `nil`.
func (ty *ExternType) MemoryType() *MemoryType {
	ptr := C.wasm_externtype_as_memorytype(ty.ptr())
	if ptr == nil {
		return nil
	}
	return mkMemoryType(ptr, ty.owner())
}

// AsExternType returns this type itself
func (ty *ExternType) AsExternType() *ExternType {
	return ty
}
