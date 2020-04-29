// Copyright 2012 Google, Inc. All rights reserved.
//
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file in the root of the source
// tree.

// +build linux

package afpacket

import (
	"errors"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

const maxOptSize = 8192

var errSockoptTooBig = errors.New("socket option too big")

// setsockopt provides access to the setsockopt syscall.
func setsockopt(fd, level, name int, val unsafe.Pointer, vallen uintptr) error {
	if vallen > maxOptSize {
		return errSockoptTooBig
	}
	slice := (*[maxOptSize]byte)(val)[:]
	return syscall.SetsockoptString(fd, level, name, string(slice[:vallen]))
}

// getsockopt provides access to the getsockopt syscall.
func getsockopt(fd, level, name int, val unsafe.Pointer, vallen *uintptr) error {
	s, err := unix.GetsockoptString(fd, level, name)
	if err != nil {
		return err
	}
	rcvLen := uintptr(len(s))
	if rcvLen > *vallen {
		return errSockoptTooBig
	}
	copy((*[maxOptSize]byte)(val)[:rcvLen], s)
	*vallen = rcvLen
	return nil
}

// htons converts a short (uint16) from host-to-network byte order.
// Thanks to mikioh for this neat trick:
// https://github.com/mikioh/-stdyng/blob/master/afpacket.go
func htons(i uint16) uint16 {
	return (i<<8)&0xff00 | i>>8
}
