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

package nfs

import (
	"encoding/binary"
	"fmt"
)

const (
	maxOpaque int = 1 << 20
	maxVector int = 1 << 15
)

// XDR maps the External Data Representation
type xdr struct {
	data   []byte
	offset int
}

func newXDR(data []byte) *xdr {
	x := makeXDR(data)
	return &x
}

func makeXDR(data []byte) xdr {
	return xdr{data: data, offset: 0}
}

func (r *xdr) size() int {
	return len(r.data)
}

func (r *xdr) getUInt() (uint32, error) {
	if r.offset+4 > len(r.data) {
		return 0, fmt.Errorf("xdr: truncated uint32")
	}
	i := binary.BigEndian.Uint32(r.data[r.offset : r.offset+4])
	r.offset += 4
	return i, nil
}

func (r *xdr) getUHyper() (uint64, error) {
	if r.offset+8 > len(r.data) {
		return 0, fmt.Errorf("xdr: truncated uint64")
	}
	i := binary.BigEndian.Uint64(r.data[r.offset : r.offset+8])
	r.offset += 8
	return i, nil
}

func (r *xdr) getString() (string, error) {
	b, err := r.getDynamicOpaque()
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (r *xdr) getOpaque(length int) ([]byte, error) {
	if length < 0 || length > maxOpaque {
		return nil, fmt.Errorf("xdr: opaque length %d exceeds limit", length)
	}

	start := r.offset
	end := start + length
	if end > len(r.data) {
		return nil, fmt.Errorf("xdr: opaque length %d exceeds buffer", length)
	}

	padding := (4 - (length & 3)) & 3
	if end+padding > len(r.data) {
		return nil, fmt.Errorf("xdr: opaque padding exceeds buffer")
	}

	b := r.data[start:end]
	r.offset = end + padding
	return b, nil
}

func (r *xdr) getDynamicOpaque() ([]byte, error) {
	l, err := r.getUInt()
	if err != nil {
		return nil, err
	}
	return r.getOpaque(int(l))
}

func (r *xdr) getUIntVector() ([]uint32, error) {
	l, err := r.getUInt()
	if err != nil {
		return nil, err
	}
	if int(l) > maxVector {
		return nil, fmt.Errorf("xdr: vector length %d exceeds limit", l)
	}
	v := make([]uint32, int(l))
	for i := 0; i < len(v); i++ {
		vi, err := r.getUInt()
		if err != nil {
			return nil, err
		}
		v[i] = vi
	}
	return v, nil
}
