package wasmtime

// #include "shims.h"
// #include <stdlib.h>
import "C"
import (
	"io/ioutil"
	"runtime"
	"unsafe"
)

// Module is a module which collects definitions for types, functions, tables, memories, and globals.
// In addition, it can declare imports and exports and provide initialization logic in the form of data and element segments or a start function.
// Modules organized WebAssembly programs as the unit of deployment, loading, and compilation.
type Module struct {
	_ptr *C.wasmtime_module_t
}

// NewModule compiles a new `Module` from the `wasm` provided with the given configuration
// in `engine`.
func NewModule(engine *Engine, wasm []byte) (*Module, error) {
	// We can't create the `wasm_byte_vec_t` here and pass it in because
	// that runs into the error of "passed a pointer to a pointer" because
	// the vec itself is passed by pointer and it contains a pointer to
	// `wasm`. To work around this we insert some C shims above and call
	// them.
	var wasmPtr *C.uint8_t
	if len(wasm) > 0 {
		wasmPtr = (*C.uint8_t)(unsafe.Pointer(&wasm[0]))
	}
	var ptr *C.wasmtime_module_t
	err := C.wasmtime_module_new(engine.ptr(), wasmPtr, C.size_t(len(wasm)), &ptr)
	runtime.KeepAlive(engine)
	runtime.KeepAlive(wasm)

	if err != nil {
		return nil, mkError(err)
	}

	return mkModule(ptr), nil
}

// NewModuleFromFile reads the contents of the `file` provided and interprets them as either the
// text format or the binary format for WebAssembly.
//
// Afterwards delegates to the `NewModule` constructor with the contents read.
func NewModuleFromFile(engine *Engine, file string) (*Module, error) {
	wasm, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	// If this wasm isn't actually wasm, treat it as the text format and
	// parse it as such.
	if len(wasm) > 0 && wasm[0] != 0 {
		wasm, err = Wat2Wasm(string(wasm))
		if err != nil {
			return nil, err
		}
	}
	return NewModule(engine, wasm)

}

// ModuleValidate validates whether `wasm` would be a valid wasm module according to the
// configuration in `store`
func ModuleValidate(engine *Engine, wasm []byte) error {
	var wasmPtr *C.uint8_t
	if len(wasm) > 0 {
		wasmPtr = (*C.uint8_t)(unsafe.Pointer(&wasm[0]))
	}
	err := C.wasmtime_module_validate(engine.ptr(), wasmPtr, C.size_t(len(wasm)))
	runtime.KeepAlive(engine)
	runtime.KeepAlive(wasm)
	if err == nil {
		return nil
	}

	return mkError(err)
}

func mkModule(ptr *C.wasmtime_module_t) *Module {
	module := &Module{_ptr: ptr}
	runtime.SetFinalizer(module, func(module *Module) {
		C.wasmtime_module_delete(module._ptr)
	})
	return module
}

func (m *Module) ptr() *C.wasmtime_module_t {
	ret := m._ptr
	maybeGC()
	return ret
}

// Type returns a `ModuleType` that corresponds for this module.
func (m *Module) Type() *ModuleType {
	ptr := C.wasmtime_module_type(m.ptr())
	runtime.KeepAlive(m)
	return mkModuleType(ptr, nil)
}

type importTypeList struct {
	vec C.wasm_importtype_vec_t
}

func (list *importTypeList) mkGoList() []*ImportType {
	runtime.SetFinalizer(list, func(imports *importTypeList) {
		C.wasm_importtype_vec_delete(&imports.vec)
	})

	ret := make([]*ImportType, int(list.vec.size))
	base := unsafe.Pointer(list.vec.data)
	var ptr *C.wasm_importtype_t
	for i := 0; i < int(list.vec.size); i++ {
		ptr := *(**C.wasm_importtype_t)(unsafe.Pointer(uintptr(base) + unsafe.Sizeof(ptr)*uintptr(i)))
		ty := mkImportType(ptr, list)
		ret[i] = ty
	}
	return ret
}

type exportTypeList struct {
	vec C.wasm_exporttype_vec_t
}

func (list *exportTypeList) mkGoList() []*ExportType {
	runtime.SetFinalizer(list, func(exports *exportTypeList) {
		C.wasm_exporttype_vec_delete(&exports.vec)
	})

	ret := make([]*ExportType, int(list.vec.size))
	base := unsafe.Pointer(list.vec.data)
	var ptr *C.wasm_exporttype_t
	for i := 0; i < int(list.vec.size); i++ {
		ptr := *(**C.wasm_exporttype_t)(unsafe.Pointer(uintptr(base) + unsafe.Sizeof(ptr)*uintptr(i)))
		ty := mkExportType(ptr, list)
		ret[i] = ty
	}
	return ret
}

// NewModuleDeserialize decodes and deserializes in-memory bytes previously
// produced by `module.Serialize()`.
//
// This function does not take a WebAssembly binary as input. It takes
// as input the results of a previous call to `Serialize()`, and only takes
// that as input.
//
// If deserialization is successful then a compiled module is returned,
// otherwise nil and an error are returned.
//
// Note that to deserialize successfully the bytes provided must have beeen
// produced with an `Engine` that has the same commpilation options as the
// provided engine, and from the same version of this library.
func NewModuleDeserialize(engine *Engine, encoded []byte) (*Module, error) {
	var encodedPtr *C.uint8_t
	var ptr *C.wasmtime_module_t
	if len(encoded) > 0 {
		encodedPtr = (*C.uint8_t)(unsafe.Pointer(&encoded[0]))
	}
	err := C.wasmtime_module_deserialize(
		engine.ptr(),
		encodedPtr,
		C.size_t(len(encoded)),
		&ptr,
	)
	runtime.KeepAlive(engine)
	runtime.KeepAlive(encoded)

	if err != nil {
		return nil, mkError(err)
	}

	return mkModule(ptr), nil
}

// NewModuleDeserializeFile is the same as `NewModuleDeserialize` except that
// the bytes are read from a file instead of provided as an argument.
func NewModuleDeserializeFile(engine *Engine, path string) (*Module, error) {
	cs := C.CString(path)
	var ptr *C.wasmtime_module_t
	err := C.wasmtime_module_deserialize_file(engine.ptr(), cs, &ptr)
	runtime.KeepAlive(engine)
	C.free(unsafe.Pointer(cs))

	if err != nil {
		return nil, mkError(err)
	}

	return mkModule(ptr), nil
}

// Serialize will convert this in-memory compiled module into a list of bytes.
//
// The purpose of this method is to extract an artifact which can be stored
// elsewhere from this `Module`. The returned bytes can, for example, be stored
// on disk or in an object store. The `NewModuleDeserialize` function can be
// used to deserialize the returned bytes at a later date to get the module
// back.
func (m *Module) Serialize() ([]byte, error) {
	retVec := C.wasm_byte_vec_t{}
	err := C.wasmtime_module_serialize(m.ptr(), &retVec)
	runtime.KeepAlive(m)

	if err != nil {
		return nil, mkError(err)
	}
	ret := C.GoBytes(unsafe.Pointer(retVec.data), C.int(retVec.size))
	C.wasm_byte_vec_delete(&retVec)
	return ret, nil
}

func (m *Module) AsExtern() C.wasmtime_extern_t {
	ret := C.wasmtime_extern_t{kind: C.WASMTIME_EXTERN_MODULE}
	C.go_wasmtime_extern_module_set(&ret, m.ptr())
	return ret
}
