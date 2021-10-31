package wasmtime

// #include <wasmtime.h>
import "C"
import "runtime"

// ModuleType describes the imports/exports of a module.
type ModuleType struct {
	_ptr   *C.wasmtime_moduletype_t
	_owner interface{}
}

func mkModuleType(ptr *C.wasmtime_moduletype_t, owner interface{}) *ModuleType {
	moduletype := &ModuleType{_ptr: ptr, _owner: owner}
	if owner == nil {
		runtime.SetFinalizer(moduletype, func(moduletype *ModuleType) {
			C.wasmtime_moduletype_delete(moduletype._ptr)
		})
	}
	return moduletype
}

func (ty *ModuleType) ptr() *C.wasmtime_moduletype_t {
	ret := ty._ptr
	maybeGC()
	return ret
}

func (ty *ModuleType) owner() interface{} {
	if ty._owner != nil {
		return ty._owner
	}
	return ty
}

// AsExternType converts this type to an instance of `ExternType`
func (ty *ModuleType) AsExternType() *ExternType {
	ptr := C.wasmtime_moduletype_as_externtype(ty.ptr())
	return mkExternType(ptr, ty.owner())
}

// Imports returns a list of `ImportType` items which are the items imported by
// this module and are required for instantiation.
func (m *ModuleType) Imports() []*ImportType {
	imports := &importTypeList{}
	C.wasmtime_moduletype_imports(m.ptr(), &imports.vec)
	runtime.KeepAlive(m)
	return imports.mkGoList()
}

// Exports returns a list of `ExportType` items which are the items that will
// be exported by this module after instantiation.
func (m *ModuleType) Exports() []*ExportType {
	exports := &exportTypeList{}
	C.wasmtime_moduletype_exports(m.ptr(), &exports.vec)
	runtime.KeepAlive(m)
	return exports.mkGoList()
}
