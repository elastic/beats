package wasmtime

// #include <wasm.h>
import "C"
import "runtime"

// TableType is one of table types which classify tables over elements of element types within a size range.
type TableType struct {
	_ptr   *C.wasm_tabletype_t
	_owner interface{}
}

// NewTableType creates a new `TableType` with the `element` type provided as
// well as limits on its size.
//
// The `min` value is the minimum size, in elements, of this table. The
// `has_max` boolean indicates whether a maximum size is present, and if so
// `max` is used as the maximum size of the table, in elements.
func NewTableType(element *ValType, min uint32, has_max bool, max uint32) *TableType {
	valptr := C.wasm_valtype_new(C.wasm_valtype_kind(element.ptr()))
	runtime.KeepAlive(element)
	if !has_max {
		max = 0xffffffff
	}
	limitsFFI := C.wasm_limits_t{
		min: C.uint32_t(min),
		max: C.uint32_t(max),
	}
	ptr := C.wasm_tabletype_new(valptr, &limitsFFI)

	return mkTableType(ptr, nil)
}

func mkTableType(ptr *C.wasm_tabletype_t, owner interface{}) *TableType {
	tabletype := &TableType{_ptr: ptr, _owner: owner}
	if owner == nil {
		runtime.SetFinalizer(tabletype, func(tabletype *TableType) {
			C.wasm_tabletype_delete(tabletype._ptr)
		})
	}
	return tabletype
}

func (ty *TableType) ptr() *C.wasm_tabletype_t {
	ret := ty._ptr
	maybeGC()
	return ret
}

func (ty *TableType) owner() interface{} {
	if ty._owner != nil {
		return ty._owner
	}
	return ty
}

// Element returns the type of value stored in this table
func (ty *TableType) Element() *ValType {
	ptr := C.wasm_tabletype_element(ty.ptr())
	return mkValType(ptr, ty.owner())
}

// Minimum returns the minimum size, in elements, of this table.
func (ty *TableType) Minimum() uint32 {
	ptr := C.wasm_tabletype_limits(ty.ptr())
	ret := uint32(ptr.min)
	runtime.KeepAlive(ty)
	return ret
}

// Maximum returns the maximum size, in elements, of this table.
//
// If no maximum size is listed then `(false, 0)` is returned, otherwise
// `(true, N)` is returned where `N` is the maximum size.
func (ty *TableType) Maximum() (bool, uint32) {
	ptr := C.wasm_tabletype_limits(ty.ptr())
	ret := uint32(ptr.max)
	runtime.KeepAlive(ty)
	if ret == 0xffffffff {
		return false, 0
	} else {
		return true, ret
	}
}

// AsExternType converts this type to an instance of `ExternType`
func (ty *TableType) AsExternType() *ExternType {
	ptr := C.wasm_tabletype_as_externtype_const(ty.ptr())
	return mkExternType(ptr, ty.owner())
}
