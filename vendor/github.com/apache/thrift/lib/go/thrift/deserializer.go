/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements. See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership. The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License. You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied. See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package thrift

import (
	"sync"
)

type TDeserializer struct {
	Transport *TMemoryBuffer
	Protocol  TProtocol
}

func NewTDeserializer() *TDeserializer {
	transport := NewTMemoryBufferLen(1024)

	protocol := NewTBinaryProtocolFactoryDefault().GetProtocol(transport)

	return &TDeserializer{
		transport,
		protocol}
}

func (t *TDeserializer) ReadString(msg TStruct, s string) (err error) {
	t.Transport.Reset()

	err = nil
	if _, err = t.Transport.Write([]byte(s)); err != nil {
		return
	}
	if err = msg.Read(t.Protocol); err != nil {
		return
	}
	return
}

func (t *TDeserializer) Read(msg TStruct, b []byte) (err error) {
	t.Transport.Reset()

	err = nil
	if _, err = t.Transport.Write(b); err != nil {
		return
	}
	if err = msg.Read(t.Protocol); err != nil {
		return
	}
	return
}

// TDeserializerPool is the thread-safe version of TDeserializer,
// it uses resource pool of TDeserializer under the hood.
//
// It must be initialized with NewTDeserializerPool.
type TDeserializerPool struct {
	pool sync.Pool
}

// NewTDeserializerPool creates a new TDeserializerPool.
//
// NewTDeserializer can be used as the arg here.
func NewTDeserializerPool(f func() *TDeserializer) *TDeserializerPool {
	return &TDeserializerPool{
		pool: sync.Pool{
			New: func() interface{} {
				return f()
			},
		},
	}
}

func (t *TDeserializerPool) ReadString(msg TStruct, s string) error {
	d := t.pool.Get().(*TDeserializer)
	defer t.pool.Put(d)
	return d.ReadString(msg, s)
}

func (t *TDeserializerPool) Read(msg TStruct, b []byte) error {
	d := t.pool.Get().(*TDeserializer)
	defer t.pool.Put(d)
	return d.Read(msg, b)
}
