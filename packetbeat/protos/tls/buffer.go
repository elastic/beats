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

package tls

import (
	"fmt"

	"github.com/menderesk/beats/v7/libbeat/common/streambuf"
)

type bufferView struct {
	buf         *streambuf.Buffer
	base, limit int
}

func newBufferView(buf *streambuf.Buffer, pos int, length int) *bufferView {
	return &bufferView{buf, pos, pos + length}
}

func (r bufferView) read8(pos int, dest *uint8) bool {
	offset := pos + r.base
	if offset+1 > r.limit {
		return false
	}
	val, err := r.buf.ReadNetUint8At(offset)
	*dest = val
	return err == nil
}

func (r bufferView) read16Net(pos int, dest *uint16) bool {
	offset := pos + r.base
	if offset+2 > r.limit {
		return false
	}
	val, err := r.buf.ReadNetUint16At(offset)
	*dest = val
	return err == nil
}

func (r bufferView) read24Net(pos int, dest *uint32) bool {
	offset := pos + r.base
	if offset+3 > r.limit {
		return false
	}
	val8, err := r.buf.ReadNetUint8At(offset)
	if err != nil {
		return false
	}
	val16, err := r.buf.ReadNetUint16At(offset + 1)
	if err != nil {
		return false
	}
	*dest = uint32(val16) | (uint32(val8) << 16)
	return true
}

func (r bufferView) read32Net(pos int, dest *uint32) bool {
	offset := pos + r.base
	if offset+4 > r.limit {
		return false
	}
	val, err := r.buf.ReadNetUint32At(offset)
	*dest = val
	return err == nil
}

func (r bufferView) readString(pos int, len int, dest *string) bool {
	if slice := r.readBytes(pos, len); slice != nil {
		*dest = string(slice)
		return true
	}
	return false
}

func (r bufferView) readBytes(pos int, length int) []byte {
	offset := pos + r.base
	lim := offset + length
	if lim > r.limit {
		return nil
	}
	bytes := r.buf.Bytes()
	if lim <= len(bytes) {
		return r.buf.Bytes()[offset:lim]
	}
	return nil
}

func (r bufferView) subview(start, length int) bufferView {
	if 0 <= start && 0 <= length && start+length <= r.limit {
		return bufferView{
			base:  r.base + start,
			limit: r.base + start + length,
			buf:   r.buf,
		}
	}

	panic(fmt.Sprintf("invalid buffer view requested start:%d len:%d limit:%d",
		start, length, r.limit))
}

func (r bufferView) length() int {
	return r.limit - r.base
}
