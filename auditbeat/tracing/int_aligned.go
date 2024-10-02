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

//go:build linux && !386 && !amd64 && !amd64p32

// Alignment-safe integer reading and writing functions.

package tracing

import (
	"errors"
	"unsafe"
)

var errBadSize = errors.New("bad size for integer")

func copyInt(dst unsafe.Pointer, src unsafe.Pointer, len uint8) error {
	copy(unsafe.Slice((*byte)(dst), len), unsafe.Slice((*byte)(src), len))
	return nil
}

func readInt(ptr unsafe.Pointer, len uint8, signed bool) (any, error) {
	var value any
	asSlice := unsafe.Slice((*byte)(ptr), len)
	switch len {
	case 1:
		if signed {
			value = int8(asSlice[0])
		} else {
			value = asSlice[0]
		}
	case 2:
		if signed {
			value = int16(MachineEndian.Uint16(asSlice))
		} else {
			value = MachineEndian.Uint16(asSlice)
		}

	case 4:
		if signed {
			value = int32(MachineEndian.Uint32(asSlice))
		} else {
			value = MachineEndian.Uint32(asSlice)
		}

	case 8:
		if signed {
			value = int64(MachineEndian.Uint64(asSlice))
		} else {
			value = MachineEndian.Uint64(asSlice)
		}

	default:
		return nil, errBadSize
	}
	return value, nil
}
