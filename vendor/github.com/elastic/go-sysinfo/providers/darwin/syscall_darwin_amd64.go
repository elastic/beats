// Copyright 2018 Elasticsearch Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build darwin,amd64,cgo

package darwin

// #cgo LDFLAGS:-lproc
// #include <sys/sysctl.h>
import "C"

import (
	"bytes"
	"encoding/binary"
	"sync"
	"syscall"
	"unsafe"
)

// Single-word zero for use when we need a valid pointer to 0 bytes.
// See mksyscall.pl.
var _zero uintptr

// Buffer Pool

var bufferPool = sync.Pool{
	New: func() interface{} {
		return &poolMem{
			buf: make([]byte, argMax),
		}
	},
}

type poolMem struct {
	buf  []byte
	pool *sync.Pool
}

func getPoolMem() *poolMem {
	pm := bufferPool.Get().(*poolMem)
	pm.buf = pm.buf[0:cap(pm.buf)]
	pm.pool = &bufferPool
	return pm
}

func (m *poolMem) Release() { m.pool.Put(m) }

// Common errors.

// Do the interface allocations only once for common
// Errno values.
var (
	errEAGAIN error = syscall.EAGAIN
	errEINVAL error = syscall.EINVAL
	errENOENT error = syscall.ENOENT
)

// errnoErr returns common boxed Errno values, to prevent
// allocations at runtime.
func errnoErr(e syscall.Errno) error {
	switch e {
	case 0:
		return nil
	case syscall.EAGAIN:
		return errEAGAIN
	case syscall.EINVAL:
		return errEINVAL
	case syscall.ENOENT:
		return errENOENT
	}
	return e
}

func _sysctl(mib []C.int, old *byte, oldlen *uintptr, new *byte, newlen uintptr) (err error) {
	var _p0 unsafe.Pointer
	if len(mib) > 0 {
		_p0 = unsafe.Pointer(&mib[0])
	} else {
		_p0 = unsafe.Pointer(&_zero)
	}
	_, _, e1 := syscall.Syscall6(syscall.SYS___SYSCTL, uintptr(_p0), uintptr(len(mib)), uintptr(unsafe.Pointer(old)), uintptr(unsafe.Pointer(oldlen)), uintptr(unsafe.Pointer(new)), uintptr(newlen))
	if e1 != 0 {
		err = errnoErr(e1)
	}
	return
}

// Translate "kern.hostname" to []_C_int{0,1,2,3}.
func nametomib(name string) (mib []C.int, err error) {
	const siz = unsafe.Sizeof(mib[0])

	// NOTE(rsc): It seems strange to set the buffer to have
	// size CTL_MAXNAME+2 but use only CTL_MAXNAME
	// as the size. I don't know why the +2 is here, but the
	// kernel uses +2 for its own implementation of this function.
	// I am scared that if we don't include the +2 here, the kernel
	// will silently write 2 words farther than we specify
	// and we'll get memory corruption.
	var buf [C.CTL_MAXNAME + 2]C.int
	n := uintptr(C.CTL_MAXNAME) * siz

	p := (*byte)(unsafe.Pointer(&buf[0]))
	bytes, err := syscall.ByteSliceFromString(name)
	if err != nil {
		return nil, err
	}

	// Magic sysctl: "setting" 0.3 to a string name
	// lets you read back the array of integers form.
	if err = _sysctl([]C.int{0, 3}, p, &n, &bytes[0], uintptr(len(name))); err != nil {
		return nil, err
	}
	return buf[0 : n/siz], nil
}

func sysctl(mib []C.int, value interface{}) error {
	mem := getPoolMem()
	defer mem.Release()

	size := uintptr(len(mem.buf))
	if err := _sysctl(mib, &mem.buf[0], &size, nil, 0); err != nil {
		return err
	}
	data := mem.buf[0:size]

	switch v := value.(type) {
	case *[]byte:
		out := make([]byte, len(data))
		copy(out, data)
		*v = out
		return nil
	default:
		return binary.Read(bytes.NewReader(data), binary.LittleEndian, v)
	}
}

func sysctlByName(name string, out interface{}) error {
	mib, err := nametomib(name)
	if err != nil {
		return err
	}

	return sysctl(mib, out)
}
