// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0

package ctxfmt

import (
	"reflect"
	"unicode/utf8"
	"unsafe"

	"github.com/urso/diag"
)

func isErrorValue(v interface{}) bool {
	if err, ok := v.(error); ok {
		return err != nil
	}
	return false
}

func unsafeString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

func unsafeBytes(s string) []byte {
	sh := *((*reflect.SliceHeader)(unsafe.Pointer(&s)))
	return *(*[]byte)((unsafe.Pointer)(&reflect.SliceHeader{Data: sh.Data, Len: sh.Len, Cap: sh.Cap}))
}

func isFieldValue(v interface{}) bool {
	_, ok := v.(diag.Field)
	return ok
}

func convRune(v uint64) rune {
	if v > utf8.MaxRune {
		return utf8.RuneError
	}
	return rune(v)
}
