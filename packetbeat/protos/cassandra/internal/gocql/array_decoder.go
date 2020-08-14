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

package cassandra

import (
	"fmt"
	"net"
)

type ByteArrayDecoder struct {
	Data *[]byte
}

func readInt(p []byte) int32 {
	return int32(p[0])<<24 | int32(p[1])<<16 | int32(p[2])<<8 | int32(p[3])
}

func (f ByteArrayDecoder) ReadByte() (byte, error) {
	data := *f.Data
	if len(data) < 1 {
		panic(fmt.Errorf("not enough bytes in buffer to Read byte require 1 got: %d", len(data)))
	}

	b := data[0]
	*f.Data = data[1:]
	return b, nil
}

func (f ByteArrayDecoder) ReadInt() (n int) {
	data := *f.Data
	if len(data) < 4 {
		panic(fmt.Errorf("not enough bytes in buffer to Read int require 4 got: %d", len(data)))
	}

	n = int(int32(data[0])<<24 | int32(data[1])<<16 | int32(data[2])<<8 | int32(data[3]))
	*f.Data = data[4:]

	return
}

func (f ByteArrayDecoder) ReadShort() (n uint16) {
	data := *f.Data
	if len(data) < 2 {
		panic(fmt.Errorf("not enough bytes in buffer to Read short require 2 got: %d", len(data)))
	}
	n = uint16(data[0])<<8 | uint16(data[1])
	*f.Data = data[2:]
	return
}

func (f ByteArrayDecoder) ReadLong() (n int64) {
	data := *f.Data
	if len(data) < 8 {
		panic(fmt.Errorf("not enough bytes in buffer to Read long require 8 got: %d", len(data)))
	}
	n = int64(data[0])<<56 | int64(data[1])<<48 | int64(data[2])<<40 | int64(data[3])<<32 |
		int64(data[4])<<24 | int64(data[5])<<16 | int64(data[6])<<8 | int64(data[7])
	*f.Data = data[8:]
	return
}

func (f ByteArrayDecoder) ReadString() (s string) {
	size := f.ReadShort()
	data := *f.Data
	if len(data) < int(size) {
		panic(fmt.Errorf("not enough bytes in buffer to Read string require %d got: %d", size, len(data)))
	}

	s = string(data[:size])
	*f.Data = data[size:]
	return
}

func (f ByteArrayDecoder) ReadLongString() (s string) {
	size := f.ReadInt()
	data := *f.Data
	if len(data) < size {
		panic(fmt.Errorf("not enough bytes in buffer to Read long string require %d got: %d", size, len(data)))
	}

	s = string(data[:size])
	*f.Data = data[size:]
	return
}

func (f ByteArrayDecoder) ReadUUID() *UUID {
	data := *f.Data

	if len(data) < 16 {
		panic(fmt.Errorf("not enough bytes in buffer to Read uuid require %d got: %d", 16, len(data)))
	}

	u, _ := UUIDFromBytes(data[:16])
	*f.Data = data[16:]
	return &u
}

func (f ByteArrayDecoder) ReadStringList() []string {
	size := f.ReadShort()

	l := make([]string, size)
	for i := 0; i < int(size); i++ {
		l[i] = f.ReadString()
	}

	return l
}

func (f ByteArrayDecoder) ReadBytesInternal() []byte {
	size := f.ReadInt()
	if size < 0 {
		return nil
	}
	data := *f.Data

	if len(data) < size {
		panic(fmt.Errorf("not enough bytes in buffer to Read bytes require %d got: %d", size, len(data)))
	}

	l := data[:size]
	*f.Data = data[size:]

	return l
}

func (f ByteArrayDecoder) ReadBytes() []byte {
	l := f.ReadBytesInternal()

	return l
}

func (f ByteArrayDecoder) ReadShortBytes() []byte {
	size := f.ReadShort()
	data := *f.Data
	if len(data) < int(size) {
		panic(fmt.Errorf("not enough bytes in buffer to Read short bytes: require %d got %d", size, len(data)))
	}

	l := data[:size]
	*f.Data = data[size:]

	return l
}

func (f ByteArrayDecoder) ReadInet() (net.IP, int) {
	data := *f.Data

	if len(data) < 1 {
		panic(fmt.Errorf("not enough bytes in buffer to Read inet size require %d got: %d", 1, len(data)))
	}

	size := data[0]
	*f.Data = data[1:]

	if !(size == 4 || size == 16) {
		panic(fmt.Errorf("invalid IP size: %d", size))
	}

	data = *f.Data
	if len(data) < 1 {
		panic(fmt.Errorf("not enough bytes in buffer to Read inet require %d got: %d", size, len(data)))
	}

	ip := make([]byte, size)
	copy(ip, data[:size])
	*f.Data = data[size:]

	port := f.ReadInt()
	return net.IP(ip), port
}

func (f ByteArrayDecoder) ReadConsistency() Consistency {
	return Consistency(f.ReadShort())
}

func (f ByteArrayDecoder) ReadStringMap() map[string]string {
	size := f.ReadShort()
	m := make(map[string]string)

	for i := 0; i < int(size); i++ {
		k := f.ReadString()
		v := f.ReadString()
		m[k] = v
	}

	return m
}

func (f ByteArrayDecoder) ReadBytesMap() map[string][]byte {
	size := f.ReadShort()
	m := make(map[string][]byte)

	for i := 0; i < int(size); i++ {
		k := f.ReadString()
		v := f.ReadBytes()
		m[k] = v
	}

	return m
}

func (f ByteArrayDecoder) ReadStringMultiMap() map[string][]string {
	size := f.ReadShort()
	m := make(map[string][]string)

	for i := 0; i < int(size); i++ {
		k := f.ReadString()
		v := f.ReadStringList()
		m[k] = v
	}
	return m
}
