// This file has been generated from 'fold_map_inline.yml', do not edit
package gotype

import (
	"reflect"
	"unsafe"
)

var _mapInlineMapping = map[reflect.Type]reFoldFn{
	tBool:    foldMapInlineBool,
	tString:  foldMapInlineString,
	tUint:    foldMapInlineUint,
	tUint8:   foldMapInlineUint8,
	tUint16:  foldMapInlineUint16,
	tUint32:  foldMapInlineUint32,
	tUint64:  foldMapInlineUint64,
	tInt:     foldMapInlineInt,
	tInt8:    foldMapInlineInt8,
	tInt16:   foldMapInlineInt16,
	tInt32:   foldMapInlineInt32,
	tInt64:   foldMapInlineInt64,
	tFloat32: foldMapInlineFloat32,
	tFloat64: foldMapInlineFloat64,
}

func getMapInlineByPrimitiveElem(t reflect.Type) reFoldFn {
	if t == tInterface {
		return foldMapInlineInterface
	}
	return _mapInlineMapping[t]
}

func foldMapInlineInterface(C *foldContext, v reflect.Value) (err error) {
	ptr := unsafe.Pointer(v.Pointer())
	if ptr == nil {
		return nil
	}

	m := *((*map[string]interface{})(unsafe.Pointer(&ptr)))
	for k, v := range m {
		if err = C.OnKey(k); err != nil {
			return err
		}
		if err = foldInterfaceValue(C, v); err != nil {
			return err
		}
	}
	return
}

func foldMapInlineBool(C *foldContext, v reflect.Value) (err error) {
	ptr := unsafe.Pointer(v.Pointer())
	if ptr == nil {
		return nil
	}

	m := *((*map[string]bool)(unsafe.Pointer(&ptr)))
	for k, v := range m {
		if err = C.OnKey(k); err != nil {
			return err
		}
		if err = C.OnBool(v); err != nil {
			return err
		}
	}
	return
}

func foldMapInlineString(C *foldContext, v reflect.Value) (err error) {
	ptr := unsafe.Pointer(v.Pointer())
	if ptr == nil {
		return nil
	}

	m := *((*map[string]string)(unsafe.Pointer(&ptr)))
	for k, v := range m {
		if err = C.OnKey(k); err != nil {
			return err
		}
		if err = C.OnString(v); err != nil {
			return err
		}
	}
	return
}

func foldMapInlineUint(C *foldContext, v reflect.Value) (err error) {
	ptr := unsafe.Pointer(v.Pointer())
	if ptr == nil {
		return nil
	}

	m := *((*map[string]uint)(unsafe.Pointer(&ptr)))
	for k, v := range m {
		if err = C.OnKey(k); err != nil {
			return err
		}
		if err = C.OnUint(v); err != nil {
			return err
		}
	}
	return
}

func foldMapInlineUint8(C *foldContext, v reflect.Value) (err error) {
	ptr := unsafe.Pointer(v.Pointer())
	if ptr == nil {
		return nil
	}

	m := *((*map[string]uint8)(unsafe.Pointer(&ptr)))
	for k, v := range m {
		if err = C.OnKey(k); err != nil {
			return err
		}
		if err = C.OnUint8(v); err != nil {
			return err
		}
	}
	return
}

func foldMapInlineUint16(C *foldContext, v reflect.Value) (err error) {
	ptr := unsafe.Pointer(v.Pointer())
	if ptr == nil {
		return nil
	}

	m := *((*map[string]uint16)(unsafe.Pointer(&ptr)))
	for k, v := range m {
		if err = C.OnKey(k); err != nil {
			return err
		}
		if err = C.OnUint16(v); err != nil {
			return err
		}
	}
	return
}

func foldMapInlineUint32(C *foldContext, v reflect.Value) (err error) {
	ptr := unsafe.Pointer(v.Pointer())
	if ptr == nil {
		return nil
	}

	m := *((*map[string]uint32)(unsafe.Pointer(&ptr)))
	for k, v := range m {
		if err = C.OnKey(k); err != nil {
			return err
		}
		if err = C.OnUint32(v); err != nil {
			return err
		}
	}
	return
}

func foldMapInlineUint64(C *foldContext, v reflect.Value) (err error) {
	ptr := unsafe.Pointer(v.Pointer())
	if ptr == nil {
		return nil
	}

	m := *((*map[string]uint64)(unsafe.Pointer(&ptr)))
	for k, v := range m {
		if err = C.OnKey(k); err != nil {
			return err
		}
		if err = C.OnUint64(v); err != nil {
			return err
		}
	}
	return
}

func foldMapInlineInt(C *foldContext, v reflect.Value) (err error) {
	ptr := unsafe.Pointer(v.Pointer())
	if ptr == nil {
		return nil
	}

	m := *((*map[string]int)(unsafe.Pointer(&ptr)))
	for k, v := range m {
		if err = C.OnKey(k); err != nil {
			return err
		}
		if err = C.OnInt(v); err != nil {
			return err
		}
	}
	return
}

func foldMapInlineInt8(C *foldContext, v reflect.Value) (err error) {
	ptr := unsafe.Pointer(v.Pointer())
	if ptr == nil {
		return nil
	}

	m := *((*map[string]int8)(unsafe.Pointer(&ptr)))
	for k, v := range m {
		if err = C.OnKey(k); err != nil {
			return err
		}
		if err = C.OnInt8(v); err != nil {
			return err
		}
	}
	return
}

func foldMapInlineInt16(C *foldContext, v reflect.Value) (err error) {
	ptr := unsafe.Pointer(v.Pointer())
	if ptr == nil {
		return nil
	}

	m := *((*map[string]int16)(unsafe.Pointer(&ptr)))
	for k, v := range m {
		if err = C.OnKey(k); err != nil {
			return err
		}
		if err = C.OnInt16(v); err != nil {
			return err
		}
	}
	return
}

func foldMapInlineInt32(C *foldContext, v reflect.Value) (err error) {
	ptr := unsafe.Pointer(v.Pointer())
	if ptr == nil {
		return nil
	}

	m := *((*map[string]int32)(unsafe.Pointer(&ptr)))
	for k, v := range m {
		if err = C.OnKey(k); err != nil {
			return err
		}
		if err = C.OnInt32(v); err != nil {
			return err
		}
	}
	return
}

func foldMapInlineInt64(C *foldContext, v reflect.Value) (err error) {
	ptr := unsafe.Pointer(v.Pointer())
	if ptr == nil {
		return nil
	}

	m := *((*map[string]int64)(unsafe.Pointer(&ptr)))
	for k, v := range m {
		if err = C.OnKey(k); err != nil {
			return err
		}
		if err = C.OnInt64(v); err != nil {
			return err
		}
	}
	return
}

func foldMapInlineFloat32(C *foldContext, v reflect.Value) (err error) {
	ptr := unsafe.Pointer(v.Pointer())
	if ptr == nil {
		return nil
	}

	m := *((*map[string]float32)(unsafe.Pointer(&ptr)))
	for k, v := range m {
		if err = C.OnKey(k); err != nil {
			return err
		}
		if err = C.OnFloat32(v); err != nil {
			return err
		}
	}
	return
}

func foldMapInlineFloat64(C *foldContext, v reflect.Value) (err error) {
	ptr := unsafe.Pointer(v.Pointer())
	if ptr == nil {
		return nil
	}

	m := *((*map[string]float64)(unsafe.Pointer(&ptr)))
	for k, v := range m {
		if err = C.OnKey(k); err != nil {
			return err
		}
		if err = C.OnFloat64(v); err != nil {
			return err
		}
	}
	return
}
