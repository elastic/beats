package wasmtime

// #include <wasm.h>
import "C"
import (
	"runtime"
	"unsafe"
)

// FuncType is one of function types which classify the signature of functions, mapping a vector of parameters to a vector of results.
// They are also used to classify the inputs and outputs of instructions.
type FuncType struct {
	_ptr   *C.wasm_functype_t
	_owner interface{}
}

// NewFuncType creates a new `FuncType` with the `kind` provided
func NewFuncType(params, results []*ValType) *FuncType {
	paramVec := mkValTypeList(params)
	resultVec := mkValTypeList(results)

	ptr := C.wasm_functype_new(&paramVec, &resultVec)

	return mkFuncType(ptr, nil)
}

func mkValTypeList(tys []*ValType) C.wasm_valtype_vec_t {
	vec := C.wasm_valtype_vec_t{}
	C.wasm_valtype_vec_new_uninitialized(&vec, C.size_t(len(tys)))
	base := unsafe.Pointer(vec.data)
	for i, ty := range tys {
		ptr := C.wasm_valtype_new(C.wasm_valtype_kind(ty.ptr()))
		*(**C.wasm_valtype_t)(unsafe.Pointer(uintptr(base) + unsafe.Sizeof(ptr)*uintptr(i))) = ptr
	}
	runtime.KeepAlive(tys)
	return vec
}

func mkFuncType(ptr *C.wasm_functype_t, owner interface{}) *FuncType {
	functype := &FuncType{_ptr: ptr, _owner: owner}
	if owner == nil {
		runtime.SetFinalizer(functype, func(functype *FuncType) {
			C.wasm_functype_delete(functype._ptr)
		})
	}
	return functype
}

func (ty *FuncType) ptr() *C.wasm_functype_t {
	ret := ty._ptr
	maybeGC()
	return ret
}

func (ty *FuncType) owner() interface{} {
	if ty._owner != nil {
		return ty._owner
	}
	return ty
}

// Params returns the parameter types of this function type
func (ty *FuncType) Params() []*ValType {
	ptr := C.wasm_functype_params(ty.ptr())
	return ty.convertTypeList(ptr)
}

// Results returns the result types of this function type
func (ty *FuncType) Results() []*ValType {
	ptr := C.wasm_functype_results(ty.ptr())
	return ty.convertTypeList(ptr)
}

func (ty *FuncType) convertTypeList(list *C.wasm_valtype_vec_t) []*ValType {
	ret := make([]*ValType, list.size)

	base := unsafe.Pointer(list.data)
	var ptr *C.wasm_valtype_t
	for i := 0; i < int(list.size); i++ {
		ptr := *(**C.wasm_valtype_t)(unsafe.Pointer(uintptr(base) + unsafe.Sizeof(ptr)*uintptr(i)))
		ty := mkValType(ptr, ty.owner())
		ret[i] = ty
	}
	return ret
}

// AsExternType converts this type to an instance of `ExternType`
func (ty *FuncType) AsExternType() *ExternType {
	ptr := C.wasm_functype_as_externtype_const(ty.ptr())
	return mkExternType(ptr, ty.owner())
}
