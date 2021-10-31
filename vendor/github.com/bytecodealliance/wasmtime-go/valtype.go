package wasmtime

// #include <wasm.h>
import "C"
import "runtime"

// ValKind enumeration of different kinds of value types
type ValKind C.wasm_valkind_t

const (
	// KindI32 is the types i32 classify 32 bit integers. Integers are not inherently signed or unsigned, their interpretation is determined by individual operations.
	KindI32 ValKind = C.WASM_I32
	// KindI64 is the types i64 classify 64 bit integers. Integers are not inherently signed or unsigned, their interpretation is determined by individual operations.
	KindI64 ValKind = C.WASM_I64
	// KindF32 is the types f32 classify 32 bit floating-point data. They correspond to the respective binary floating-point representations, also known as single and double precision, as defined by the IEEE 754-2019 standard.
	KindF32 ValKind = C.WASM_F32
	// KindF64 is the types f64 classify 64 bit floating-point data. They correspond to the respective binary floating-point representations, also known as single and double precision, as defined by the IEEE 754-2019 standard.
	KindF64 ValKind = C.WASM_F64
	// TODO: Unknown
	KindExternref ValKind = C.WASM_ANYREF
	// KindFuncref is the infinite union of all function types.
	KindFuncref ValKind = C.WASM_FUNCREF
)

// String renders this kind as a string, similar to the `*.wat` format
func (ty ValKind) String() string {
	switch ty {
	case KindI32:
		return "i32"
	case KindI64:
		return "i64"
	case KindF32:
		return "f32"
	case KindF64:
		return "f64"
	case KindExternref:
		return "externref"
	case KindFuncref:
		return "funcref"
	}
	panic("unknown kind")
}

// ValType means one of the value types, which classify the individual values that WebAssembly code can compute with and the values that a variable accepts.
type ValType struct {
	_ptr   *C.wasm_valtype_t
	_owner interface{}
}

// NewValType creates a new `ValType` with the `kind` provided
func NewValType(kind ValKind) *ValType {
	ptr := C.wasm_valtype_new(C.wasm_valkind_t(kind))
	return mkValType(ptr, nil)
}

func mkValType(ptr *C.wasm_valtype_t, owner interface{}) *ValType {
	valtype := &ValType{_ptr: ptr, _owner: owner}
	if owner == nil {
		runtime.SetFinalizer(valtype, func(valtype *ValType) {
			C.wasm_valtype_delete(valtype._ptr)
		})
	}
	return valtype
}

// Kind returns the corresponding `ValKind` for this `ValType`
func (t *ValType) Kind() ValKind {
	ret := ValKind(C.wasm_valtype_kind(t.ptr()))
	runtime.KeepAlive(t)
	return ret
}

// Converts this `ValType` into a string according to the string representation
// of `ValKind`.
func (t *ValType) String() string {
	return t.Kind().String()
}

func (t *ValType) ptr() *C.wasm_valtype_t {
	ret := t._ptr
	maybeGC()
	return ret
}

func (t *ValType) owner() interface{} {
	if t._owner != nil {
		return t._owner
	}
	return t
}
