package wasmtime

// #include "shims.h"
import "C"
import (
	"runtime"
	"unsafe"
)

// Memory instance is the runtime representation of a linear memory.
// It holds a vector of bytes and an optional maximum size, if one was specified at the definition site of the memory.
// Read more in [spec](https://webassembly.github.io/spec/core/exec/runtime.html#memory-instances)
// In wasmtime-go, you can get the vector of bytes by the unsafe pointer of memory from `Memory.Data()`, or go style byte slice from `Memory.UnsafeData()`
type Memory struct {
	val C.wasmtime_memory_t
}

// NewMemory creates a new `Memory` in the given `Store` with the specified `ty`.
func NewMemory(store Storelike, ty *MemoryType) (*Memory, error) {
	var ret C.wasmtime_memory_t
	err := C.wasmtime_memory_new(store.Context(), ty.ptr(), &ret)
	runtime.KeepAlive(store)
	runtime.KeepAlive(ty)
	if err != nil {
		return nil, mkError(err)
	}
	return mkMemory(ret), nil
}

func mkMemory(val C.wasmtime_memory_t) *Memory {
	return &Memory{val}
}

// Type returns the type of this memory
func (mem *Memory) Type(store Storelike) *MemoryType {
	ptr := C.wasmtime_memory_type(store.Context(), &mem.val)
	runtime.KeepAlive(store)
	return mkMemoryType(ptr, nil)
}

// Data returns the raw pointer in memory of where this memory starts
func (mem *Memory) Data(store Storelike) unsafe.Pointer {
	ret := unsafe.Pointer(C.wasmtime_memory_data(store.Context(), &mem.val))
	runtime.KeepAlive(store)
	return ret
}

// UnsafeData returns the raw memory backed by this `Memory` as a byte slice (`[]byte`).
//
// This is not a safe method to call, hence the "unsafe" in the name. The byte
// slice returned from this function is not managed by the Go garbage collector.
// You need to ensure that `m`, the original `Memory`, lives longer than the
// `[]byte` returned.
//
// Note that you may need to use `runtime.KeepAlive` to keep the original memory
// `m` alive for long enough while you're using the `[]byte` slice. If the
// `[]byte` slice is used after `m` is GC'd then that is undefined behavior.
func (mem *Memory) UnsafeData(store Storelike) []byte {
	// see https://github.com/golang/go/wiki/cgo#turning-c-arrays-into-go-slices
	const MaxLen = 1 << 32
	length := mem.DataSize(store)
	if length >= MaxLen {
		panic("memory is too big")
	}
	return (*[MaxLen]byte)(mem.Data(store))[:length:length]
}

// DataSize returns the size, in bytes, that `Data()` is valid for
func (mem *Memory) DataSize(store Storelike) uintptr {
	ret := uintptr(C.wasmtime_memory_data_size(store.Context(), &mem.val))
	runtime.KeepAlive(store)
	return ret
}

// Size returns the size, in wasm pages, of this memory
func (mem *Memory) Size(store Storelike) uint64 {
	ret := uint64(C.wasmtime_memory_size(store.Context(), &mem.val))
	runtime.KeepAlive(store)
	return ret
}

// Grow grows this memory by `delta` pages
func (mem *Memory) Grow(store Storelike, delta uint64) (uint64, error) {
	prev := C.uint64_t(0)
	err := C.wasmtime_memory_grow(store.Context(), &mem.val, C.uint64_t(delta), &prev)
	runtime.KeepAlive(store)
	if err != nil {
		return 0, mkError(err)
	}
	return uint64(prev), nil
}

func (mem *Memory) AsExtern() C.wasmtime_extern_t {
	ret := C.wasmtime_extern_t{kind: C.WASMTIME_EXTERN_MEMORY}
	C.go_wasmtime_extern_memory_set(&ret, mem.val)
	return ret
}
