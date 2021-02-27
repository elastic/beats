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
)

// XDR maps the External Data Representation
type xdr struct {
	data   []byte
	offset uint32
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

func (r *xdr) getInt() int32 {
	i := int32(binary.BigEndian.Uint32(r.data[r.offset : r.offset+4]))
	r.offset += 4
	return int32(i)
}

func (r *xdr) getUInt() uint32 {
	i := uint32(binary.BigEndian.Uint32(r.data[r.offset : r.offset+4]))
	r.offset += 4
	return i
}

func (r *xdr) getHyper() int64 {
	i := int64(binary.BigEndian.Uint64(r.data[r.offset : r.offset+8]))
	r.offset += 8
	return i
}

func (r *xdr) getUHyper() uint64 {
	i := uint64(binary.BigEndian.Uint64(r.data[r.offset : r.offset+8]))
	r.offset += 8
	return i
}

func (r *xdr) getString() string {
	return string(r.getDynamicOpaque())
}

func (r *xdr) getOpaque(length uint32) []byte {
	padding := (4 - (length & 3)) & 3
	b := r.data[r.offset : r.offset+length]
	r.offset += length + padding
	return b
}

func (r *xdr) getDynamicOpaque() []byte {
	l := r.getUInt()
	return r.getOpaque(l)
}

func (r *xdr) getUIntVector() []uint32 {
	l := r.getUInt()
	v := make([]uint32, int(l))
	for i := 0; i < len(v); i++ {
		v[i] = r.getUInt()
	}
	return v
}
