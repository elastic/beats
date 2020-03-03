// +build !go1.10

// Copyright 2017 Tamás Gulácsi
//
//
// SPDX-License-Identifier: UPL-1.0 OR Apache-2.0

package goracle

/*
#include <stdlib.h>
#include "dpiImpl.h"
*/
import "C"
import (
	"bytes"
	"sync"
	"unsafe"
)

const go10 = false

func dpiSetFromString(dv *C.dpiVar, pos C.uint32_t, x string) {
	b := []byte(x)
	C.dpiVar_setFromBytes(dv, pos, (*C.char)(unsafe.Pointer(&b[0])), C.uint32_t(len(b)))
}

var stringBuilders = stringBuilderPool{
	p: &sync.Pool{New: func() interface{} { return bytes.NewBuffer(make([]byte, 0, 1024)) }},
}

type stringBuilderPool struct {
	p *sync.Pool
}

func (sb stringBuilderPool) Get() *bytes.Buffer {
	return sb.p.Get().(*bytes.Buffer)
}
func (sb *stringBuilderPool) Put(b *bytes.Buffer) {
	b.Reset()
	sb.p.Put(b)
}
