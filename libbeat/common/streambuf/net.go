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

package streambuf

// read integers in network byte order

import (
	"github.com/elastic/beats/v8/libbeat/common"
)

// Parse 8bit binary value from Buffer.
func (b *Buffer) ReadNetUint8() (uint8, error) {
	if b.Failed() {
		return 0, b.err
	}
	tmp := b.data[b.mark:]
	if err := b.Advance(1); err != nil {
		return 0, err
	}
	value := tmp[0]
	return value, nil
}

// Write 8bit binary value to Buffer.
func (b *Buffer) WriteNetUint8(u uint8) error {
	return b.Append([]byte{u})
}

// Parse 8bit binary value from the buffer at index. Will not advance the read
// buffer
func (b *Buffer) ReadNetUint8At(index int) (uint8, error) {
	if b.Failed() {
		return 0, b.err
	}
	if !b.Avail(1 + index) {
		return 0, b.bufferEndError()
	}
	return b.data[index+b.mark], nil
}

// Write 8bit binary value at index.
func (b *Buffer) WriteNetUint8At(u uint8, index int) error {
	if b.err != nil {
		return b.err
	}
	b.sliceAt(index, 1)[0] = u
	return nil
}

// Parse 16bit binary value in network byte order from Buffer
// (converted to Host order).
func (b *Buffer) ReadNetUint16() (uint16, error) {
	if b.Failed() {
		return 0, b.err
	}
	tmp := b.data[b.mark:]
	if err := b.Advance(2); err != nil {
		return 0, err
	}
	value := common.BytesNtohs(tmp)
	return value, nil
}

// Write 16bit binary value in network byte order to buffer.
func (b *Buffer) WriteNetUint16(u uint16) error {
	return b.WriteNetUint16At(u, b.available)
}

// Parse 16bit binary value from the buffer at index. Will not advance the read
// buffer
func (b *Buffer) ReadNetUint16At(index int) (uint16, error) {
	if b.Failed() {
		return 0, b.err
	}
	if !b.Avail(2 + index) {
		return 0, b.bufferEndError()
	}
	return common.BytesNtohs(b.data[index+b.mark:]), nil
}

// Write 16bit binary value at index in network byte order to buffer.
func (b *Buffer) WriteNetUint16At(u uint16, index int) error {
	if b.err != nil {
		return b.err
	}
	tmp := b.sliceAt(index, 2)
	tmp[0] = uint8(u >> 8)
	tmp[1] = uint8(u)
	return nil
}

// Parse 32bit binary value in network byte order from Buffer
// (converted to Host order).
func (b *Buffer) ReadNetUint32() (uint32, error) {
	if b.Failed() {
		return 0, b.err
	}
	tmp := b.data[b.mark:]
	if err := b.Advance(4); err != nil {
		return 0, err
	}
	value := common.BytesNtohl(tmp)
	return value, nil
}

// Write 32bit binary value in network byte order to buffer.
func (b *Buffer) WriteNetUint32(u uint32) error {
	return b.WriteNetUint32At(u, b.available)
}

// Parse 32bit binary value from the buffer at index. Will not advance the read
// buffer
func (b *Buffer) ReadNetUint32At(index int) (uint32, error) {
	if b.Failed() {
		return 0, b.err
	}
	if !b.Avail(4 + index) {
		return 0, b.bufferEndError()
	}
	return common.BytesNtohl(b.data[index+b.mark:]), nil
}

// Write 32bit binary value at index in network byte order to buffer.
func (b *Buffer) WriteNetUint32At(u uint32, index int) error {
	if b.err != nil {
		return b.err
	}
	tmp := b.sliceAt(index, 4)
	tmp[0] = uint8(u >> 24)
	tmp[1] = uint8(u >> 16)
	tmp[2] = uint8(u >> 8)
	tmp[3] = uint8(u)
	return nil
}

// Parse 64bit binary value in network byte order from Buffer
// (converted to Host order).
func (b *Buffer) ReadNetUint64() (uint64, error) {
	if b.Failed() {
		return 0, b.err
	}
	tmp := b.data[b.mark:]
	if err := b.Advance(8); err != nil {
		return 0, err
	}
	value := common.BytesNtohll(tmp)
	return value, nil
}

// Write 64bit binary value in network byte order to buffer.
func (b *Buffer) WriteNetUint64(u uint64) error {
	return b.WriteNetUint64At(u, b.available)
}

// Parse 64bit binary value from the buffer at index. Will not advance the read
// buffer
func (b *Buffer) ReadNetUint64At(index int) (uint64, error) {
	if b.Failed() {
		return 0, b.err
	}
	if !b.Avail(8 + index) {
		return 0, b.bufferEndError()
	}
	return common.BytesNtohll(b.data[index+b.mark:]), nil
}

// Write 64bit binary value at index in network byte order to buffer.
func (b *Buffer) WriteNetUint64At(u uint64, index int) error {
	if b.err != nil {
		return b.err
	}
	tmp := b.sliceAt(index, 8)
	tmp[0] = uint8(u >> 56)
	tmp[1] = uint8(u >> 48)
	tmp[2] = uint8(u >> 40)
	tmp[3] = uint8(u >> 32)
	tmp[4] = uint8(u >> 24)
	tmp[5] = uint8(u >> 16)
	tmp[6] = uint8(u >> 8)
	tmp[7] = uint8(u)
	return nil
}
