package wasmtime

// #include <stdlib.h>
// #include <wasm.h>
// #include <wasmtime.h>
import "C"

import (
	"runtime"
	"unsafe"
)

// Trap is the trap instruction which represents the occurrence of a trap.
// Traps are bubbled up through nested instruction sequences, ultimately reducing the entire program to a single trap instruction, signalling abrupt termination.
type Trap struct {
	_ptr *C.wasm_trap_t
}

// Frame is one of activation frames which carry the return arity n of the respective function,
// hold the values of its locals (including arguments) in the order corresponding to their static local indices,
// and a reference to the function’s own module instance
type Frame struct {
	_ptr   *C.wasm_frame_t
	_owner interface{}
}

// TrapCode is the code of an instruction trap.
type TrapCode uint8

const (
	// StackOverflow: the current stack space was exhausted.
	StackOverflow TrapCode = iota
	// MemoryOutOfBounds: out-of-bounds memory access.
	MemoryOutOfBounds
	// HeapMisaligned: a wasm atomic operation was presented with a not-naturally-aligned linear-memory address.
	HeapMisaligned
	// TableOutOfBounds: out-of-bounds access to a table.
	TableOutOfBounds
	// IndirectCallToNull: indirect call to a null table entry.
	IndirectCallToNull
	// BadSignature: signature mismatch on indirect call.
	BadSignature
	// IntegerOverflow: an integer arithmetic operation caused an overflow.
	IntegerOverflow
	// IntegerDivisionByZero: integer division by zero.
	IntegerDivisionByZero
	// BadConversionToInteger: failed float-to-int conversion.
	BadConversionToInteger
	// UnreachableCodeReached: code that was supposed to have been unreachable was reached.
	UnreachableCodeReached
	// Interrupt: execution has been interrupted.
	Interrupt
)

// NewTrap creates a new `Trap` with the `name` and the type provided.
func NewTrap(message string) *Trap {
	ptr := C.wasmtime_trap_new(C._GoStringPtr(message), C._GoStringLen(message))
	runtime.KeepAlive(message)
	return mkTrap(ptr)
}

func mkTrap(ptr *C.wasm_trap_t) *Trap {
	trap := &Trap{_ptr: ptr}
	runtime.SetFinalizer(trap, func(trap *Trap) {
		C.wasm_trap_delete(trap._ptr)
	})
	return trap
}

func (t *Trap) ptr() *C.wasm_trap_t {
	ret := t._ptr
	maybeGC()
	return ret
}

// Message returns the message of the `Trap`
func (t *Trap) Message() string {
	message := C.wasm_byte_vec_t{}
	C.wasm_trap_message(t.ptr(), &message)
	ret := C.GoStringN(message.data, C.int(message.size-1))
	runtime.KeepAlive(t)
	C.wasm_byte_vec_delete(&message)
	return ret
}

// Code returns the code of the `Trap` if it exists, nil otherwise.
func (t *Trap) Code() *TrapCode {
	var code C.uint8_t
	var ret *TrapCode
	ok := C.wasmtime_trap_code(t.ptr(), &code)
	if ok == C._Bool(true) {
		ret = (*TrapCode)(&code)
	}
	runtime.KeepAlive(t)
	return ret
}

func (t *Trap) Error() string {
	return t.Message()
}

func unwrapStrOr(s *string, other string) string {
	if s == nil {
		return other
	}

	return *s
}

type frameList struct {
	vec C.wasm_frame_vec_t
}

// Frames returns the wasm function frames that make up this trap
func (t *Trap) Frames() []*Frame {
	frames := &frameList{}
	C.wasm_trap_trace(t.ptr(), &frames.vec)
	runtime.KeepAlive(t)
	runtime.SetFinalizer(frames, func(frames *frameList) {
		C.wasm_frame_vec_delete(&frames.vec)
	})

	ret := make([]*Frame, int(frames.vec.size))
	base := unsafe.Pointer(frames.vec.data)
	var ptr *C.wasm_frame_t
	for i := 0; i < int(frames.vec.size); i++ {
		ptr := *(**C.wasm_frame_t)(unsafe.Pointer(uintptr(base) + unsafe.Sizeof(ptr)*uintptr(i)))
		ret[i] = &Frame{
			_ptr:   ptr,
			_owner: frames,
		}
	}
	return ret
}

func (f *Frame) ptr() *C.wasm_frame_t {
	ret := f._ptr
	maybeGC()
	return ret
}

// FuncIndex returns the function index in the wasm module that this frame represents
func (f *Frame) FuncIndex() uint32 {
	ret := C.wasm_frame_func_index(f.ptr())
	runtime.KeepAlive(f)
	return uint32(ret)
}

// FuncName returns the name, if available, for this frame's function
func (f *Frame) FuncName() *string {
	ret := C.wasmtime_frame_func_name(f.ptr())
	if ret == nil {
		runtime.KeepAlive(f)
		return nil
	}
	str := C.GoStringN(ret.data, C.int(ret.size))
	runtime.KeepAlive(f)
	return &str
}

// ModuleName returns the name, if available, for this frame's module
func (f *Frame) ModuleName() *string {
	ret := C.wasmtime_frame_module_name(f.ptr())
	if ret == nil {
		runtime.KeepAlive(f)
		return nil
	}
	str := C.GoStringN(ret.data, C.int(ret.size))
	runtime.KeepAlive(f)
	return &str
}

// ModuleOffset returns offset of this frame's instruction into the original module
func (f *Frame) ModuleOffset() uint {
	ret := uint(C.wasm_frame_module_offset(f.ptr()))
	runtime.KeepAlive(f)
	return ret
}

// FuncOffset returns offset of this frame's instruction into the original function
func (f *Frame) FuncOffset() uint {
	ret := uint(C.wasm_frame_func_offset(f.ptr()))
	runtime.KeepAlive(f)
	return ret
}
