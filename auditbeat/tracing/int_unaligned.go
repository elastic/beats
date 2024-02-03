// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

//go:build linux && (386 || amd64 || amd64p32)

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

func readInt(ptr unsafe.Pointer, len uint8, signed bool) (any, error) {
	var value any

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
	return value, nil
}
