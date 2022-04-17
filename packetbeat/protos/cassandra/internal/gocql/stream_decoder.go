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

	"github.com/menderesk/beats/v7/libbeat/common/streambuf"
)

type StreamDecoder struct {
	r *streambuf.Buffer
}

func (f StreamDecoder) ReadByte() (byte, error) {
	return f.r.ReadByte()
}

func (f StreamDecoder) ReadInt() (n int) {
	data, err := f.r.ReadNetUint32()
	if err != nil {
		panic(err)
	}
	n = int(data)

	return
}

func (f StreamDecoder) ReadShort() (n uint16) {
	data, err := f.r.ReadNetUint16()
	if err != nil {
		panic(err)
	}
	n = data

	return
}

func (f StreamDecoder) ReadLong() (n int64) {
	data, err := f.r.ReadNetUint64()
	if err != nil {
		panic(err)
	}
	n = int64(data)

	return
}

func (f StreamDecoder) ReadString() (s string) {
	size := f.ReadShort()

	str := make([]byte, size)
	_, err := f.r.Read(str)
	if err != nil {
		panic(err)
	}
	s = string(str)

	return
}

func (f StreamDecoder) ReadLongString() (s string) {
	size := f.ReadInt()

	if !f.r.Avail(size) {
		panic(fmt.Errorf("not enough buf to readLongString,need:%d,actual:%d", size, f.r.Len()))
	}
	str := make([]byte, size)
	_, err := f.r.Read(str)
	if err != nil {
		panic(err)
	}
	s = string(str)

	return
}

func (f StreamDecoder) ReadUUID() *UUID {
	bytes := make([]byte, 16)
	_, err := f.r.Read(bytes)
	if err != nil {
		panic(err)
	}

	u, _ := UUIDFromBytes(bytes)
	return &u
}

func (f StreamDecoder) ReadStringList() []string {
	size := f.ReadShort()

	l := make([]string, size)
	for i := 0; i < int(size); i++ {
		l[i] = f.ReadString()
	}

	return l
}

func (f StreamDecoder) ReadBytesInternal() []byte {
	size := f.ReadInt()
	if size < 0 {
		return nil
	}

	bytes := make([]byte, size)
	_, err := f.r.Read(bytes)
	if err != nil {
		panic(err)
	}
	return bytes
}

func (f StreamDecoder) ReadBytes() []byte {
	l := f.ReadBytesInternal()
	return l
}

func (f StreamDecoder) ReadShortBytes() []byte {
	size := f.ReadShort()

	bytes := make([]byte, size)
	_, err := f.r.Read(bytes)
	if err != nil {
		panic(err)
	}
	return bytes
}

func (f StreamDecoder) ReadInet() (net.IP, int) {
	size, err := f.ReadByte()
	if err != nil {
		panic(err)
	}

	if size != 4 && size != 16 {
		panic(fmt.Errorf("invalid IP size: %d", size))
	}

	ip := make([]byte, int(size))
	_, err = f.r.Read(ip)
	if err != nil {
		panic(err)
	}
	port := f.ReadInt()
	return net.IP(ip), port
}

func (f StreamDecoder) ReadConsistency() Consistency {
	return Consistency(f.ReadShort())
}

func (f StreamDecoder) ReadStringMap() map[string]string {
	size := f.ReadShort()
	m := make(map[string]string)

	for i := 0; i < int(size); i++ {
		k := f.ReadString()
		v := f.ReadString()
		m[k] = v
	}

	return m
}

func (f StreamDecoder) ReadBytesMap() map[string][]byte {
	size := f.ReadShort()
	m := make(map[string][]byte)

	for i := 0; i < int(size); i++ {
		k := f.ReadString()
		v := f.ReadBytes()
		m[k] = v
	}

	return m
}

func (f StreamDecoder) ReadStringMultiMap() map[string][]string {
	size := f.ReadShort()
	m := make(map[string][]string)

	for i := 0; i < int(size); i++ {
		k := f.ReadString()
		v := f.ReadStringList()
		m[k] = v
	}
	return m
}
