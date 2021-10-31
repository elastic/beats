package bin

import (
	"errors"
	"fmt"
	"reflect"
	"unsafe"
)

var (
	errPtrPtrStructRequired = errors.New("pointer to pointer of go structure required")
)

type emptyIfc struct {
	typ, ptr unsafe.Pointer
}

// UnsafeCastStruct casts a byte slice its contents into an arbitrary go-structure.
// The structure passed must be a pointer to a pointer of a struct to be casted too.
//
// If the input buffers length is 0, `to` will be set to nil.
//
// The operation is unsafe, as it does not validate the input value to be a
// pointer of a pointer, plus no length check is executed.
func UnsafeCastStruct(to interface{}, b []byte) {
	ifc := (*emptyIfc)(unsafe.Pointer(&to))

	if len(b) != 0 {
		*(*uintptr)(ifc.ptr) = uintptr(unsafe.Pointer(&b[0]))
	} else {
		*(*uintptr)(ifc.ptr) = 0
	}
}

// CastStruct casts a byte slice its contents into an arbitrary go-structure.
// The structure passed must be a pointer to a pointer of a structed to be casted too.
// An error is returned if the input type is invalid or the buffer is not big
// enough to hold the structure.
// If the input buffers length is 0, `to` will be set to nil.
func CastStruct(to interface{}, b []byte) error {
	v := reflect.ValueOf(to)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Ptr {
		return errPtrPtrStructRequired
	}

	if bl, tl := len(b), int(v.Type().Size()); 0 < bl && bl < tl {
		return fmt.Errorf("buffer of %v byte(s) can not be casted into structure requiring %v byte(s)",
			bl, tl)
	}

	UnsafeCastStruct(to, b)
	return nil
}
