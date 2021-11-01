// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux && (386 || amd64 || amd64p32)
// +build linux
// +build 386 amd64 amd64p32

// Integer reading and writing functions for platforms where alignment is not a problem.

package tracing

import (
	"errors"
	"unsafe"
)

var errBadSize = errors.New("bad size for integer")

func copyInt(dst unsafe.Pointer, src unsafe.Pointer, len uint8) error {
	switch len {
	case 1:
		*(*uint8)(dst) = *(*uint8)(src)

	case 2:
		*(*uint16)(dst) = *(*uint16)(src)

	case 4:
		*(*uint32)(dst) = *(*uint32)(src)

	case 8:
		*(*uint64)(dst) = *(*uint64)(src)

	default:
		return errBadSize
	}
	return nil
}

func readInt(ptr unsafe.Pointer, len uint8, signed bool) (value interface{}, err error) {
	switch len {
	case 1:
		if signed {
			value = *(*int8)(ptr)
		} else {
			value = *(*uint8)(ptr)
		}
	case 2:
		if signed {
			value = *(*int16)(ptr)
		} else {
			value = *(*uint16)(ptr)
		}

	case 4:
		if signed {
			value = *(*int32)(ptr)
		} else {
			value = *(*uint32)(ptr)
		}

	case 8:
		if signed {
			value = *(*int64)(ptr)
		} else {
			value = *(*uint64)(ptr)
		}

	default:
		return nil, errBadSize
	}
	return
}
