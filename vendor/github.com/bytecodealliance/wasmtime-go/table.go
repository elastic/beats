package wasmtime

// #include "shims.h"
import "C"
import (
	"errors"
	"runtime"
)

// Table is a table instance, which is the runtime representation of a table.
//
// It holds a vector of reference types and an optional maximum size, if one was
// specified in the table type at the table’s definition site.
// Read more in [spec](https://webassembly.github.io/spec/core/exec/runtime.html#table-instances)
type Table struct {
	val C.wasmtime_table_t
}

// NewTable creates a new `Table` in the given `Store` with the specified `ty`.
//
// The `ty` must be a reference type (`funref` or `externref`) and `init`
// is the initial value for all table slots and must have the type specified by
// `ty`.
func NewTable(store Storelike, ty *TableType, init Val) (*Table, error) {
	var ret C.wasmtime_table_t
	err := C.wasmtime_table_new(store.Context(), ty.ptr(), init.ptr(), &ret)
	runtime.KeepAlive(store)
	runtime.KeepAlive(ty)
	runtime.KeepAlive(init)
	if err != nil {
		return nil, mkError(err)
	}
	return mkTable(ret), nil
}

func mkTable(val C.wasmtime_table_t) *Table {
	return &Table{val}
}

// Size returns the size of this table in units of elements.
func (t *Table) Size(store Storelike) uint32 {
	ret := C.wasmtime_table_size(store.Context(), &t.val)
	runtime.KeepAlive(store)
	return uint32(ret)
}

// Grow grows this table by the number of units specified, using the
// specified initializer value for new slots.
//
// Returns an error if the table failed to grow, or the previous size of the
// table if growth was successful.
func (t *Table) Grow(store Storelike, delta uint32, init Val) (uint32, error) {
	var prev C.uint32_t
	err := C.wasmtime_table_grow(store.Context(), &t.val, C.uint32_t(delta), init.ptr(), &prev)
	runtime.KeepAlive(store)
	runtime.KeepAlive(init)
	if err != nil {
		return 0, mkError(err)
	}

	return uint32(prev), nil
}

// Get gets an item from this table from the specified index.
//
// Returns an error if the index is out of bounds, or returns a value (which
// may be internally null) if the index is in bounds corresponding to the entry
// at the specified index.
func (t *Table) Get(store Storelike, idx uint32) (Val, error) {
	var val C.wasmtime_val_t
	ok := C.wasmtime_table_get(store.Context(), &t.val, C.uint32_t(idx), &val)
	runtime.KeepAlive(store)
	if !ok {
		return Val{}, errors.New("index out of bounds")
	}
	return takeVal(&val), nil
}

// Set sets an item in this table at the specified index.
//
// Returns an error if the index is out of bounds.
func (t *Table) Set(store Storelike, idx uint32, val Val) error {
	err := C.wasmtime_table_set(store.Context(), &t.val, C.uint32_t(idx), val.ptr())
	runtime.KeepAlive(store)
	runtime.KeepAlive(val)
	if err != nil {
		return mkError(err)
	}
	return nil
}

// Type returns the underlying type of this table
func (t *Table) Type(store Storelike) *TableType {
	ptr := C.wasmtime_table_type(store.Context(), &t.val)
	runtime.KeepAlive(store)
	return mkTableType(ptr, nil)
}

func (t *Table) AsExtern() C.wasmtime_extern_t {
	ret := C.wasmtime_extern_t{kind: C.WASMTIME_EXTERN_TABLE}
	C.go_wasmtime_extern_table_set(&ret, t.val)
	return ret
}
