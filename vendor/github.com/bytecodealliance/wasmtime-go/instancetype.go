package wasmtime

// #include <wasmtime.h>
import "C"
import "runtime"

// InstanceType describes the exports of an instance.
type InstanceType struct {
	_ptr   *C.wasmtime_instancetype_t
	_owner interface{}
}

func mkInstanceType(ptr *C.wasmtime_instancetype_t, owner interface{}) *InstanceType {
	instancetype := &InstanceType{_ptr: ptr, _owner: owner}
	if owner == nil {
		runtime.SetFinalizer(instancetype, func(instancetype *InstanceType) {
			C.wasmtime_instancetype_delete(instancetype._ptr)
		})
	}
	return instancetype
}

func (ty *InstanceType) ptr() *C.wasmtime_instancetype_t {
	ret := ty._ptr
	maybeGC()
	return ret
}

func (ty *InstanceType) owner() interface{} {
	if ty._owner != nil {
		return ty._owner
	}
	return ty
}

// AsExternType converts this type to an instance of `ExternType`
func (ty *InstanceType) AsExternType() *ExternType {
	ptr := C.wasmtime_instancetype_as_externtype(ty.ptr())
	return mkExternType(ptr, ty.owner())
}

// Exports returns a list of `ExportType` items which are the items that will
// be exported by this instance after instantiation.
func (ty *InstanceType) Exports() []*ExportType {
	exports := &exportTypeList{}
	C.wasmtime_instancetype_exports(ty.ptr(), &exports.vec)
	runtime.KeepAlive(ty)
	return exports.mkGoList()
}
