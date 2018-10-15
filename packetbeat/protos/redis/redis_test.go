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

// +build !integration

package redis

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func newTestStream(content []byte) *stream {
	st := newStream(time.Now(), nil)
	st.Append(content)
	return st
}

func parse(content []byte) (*redisMessage, bool, bool) {
	st := newTestStream(content)
	ok, complete := st.parser.parse(&st.Buf)
	return st.parser.message, ok, complete
}

var noArgsRequest = []byte("*1\r\n" +
	"$4\r\n" +
	"INFO\r\n")

func TestRedisParser_NoArgsRequest(t *testing.T) {
	msg, ok, complete := parse(noArgsRequest)

	assert.True(t, ok)
	assert.True(t, complete)
	assert.True(t, msg.isRequest)
	assert.Equal(t, "INFO", string(msg.message))
	assert.Equal(t, len(noArgsRequest), msg.size)
}

var arrayRequest = []byte("*3\r\n" +
	"$3\r\n" +
	"SET\r\n" +
	"$4\r\n" +
	"key1\r\n" +
	"$5\r\n" +
	"Hello\r\n")

func TestRedisParser_ArrayRequest(t *testing.T) {
	msg, ok, complete := parse(arrayRequest)

	assert.True(t, ok)
	assert.True(t, complete)
	assert.True(t, msg.isRequest)
	assert.Equal(t, "SET key1 Hello", string(msg.message))
	assert.Equal(t, len(arrayRequest), msg.size)
}

var arrayResponse = []byte("*4\r\n" +
	"$3\r\n" +
	"foo\r\n" +
	"$-1\r\n" +
	"$3\r\n" +
	"bar\r\n" +
	":23\r\n")

func TestRedisParser_ArrayResponse(t *testing.T) {
	msg, ok, complete := parse(arrayResponse)

	assert.True(t, ok)
	assert.True(t, complete)
	assert.False(t, msg.isRequest)
	assert.Equal(t, "[foo, nil, bar, 23]", string(msg.message))
	assert.Equal(t, len(arrayResponse), msg.size)
}

var arrayNestedMessage = []byte("*3\r\n" +
	"*-1\r\n" +
	"+foo\r\n" +
	"*2\r\n" +
	":1\r\n" +
	"+bar\r\n")

func TestRedisParser_ArrayNested(t *testing.T) {
	msg, ok, complete := parse(arrayNestedMessage)

	assert.True(t, ok)
	assert.True(t, complete)
	assert.False(t, msg.isRequest)
	assert.Equal(t, "[nil, foo, [1, bar]]", string(msg.message))
	assert.Equal(t, len(arrayNestedMessage), msg.size)
}

func TestRedisParser_SimpleString(t *testing.T) {
	message := []byte("+OK\r\n")
	msg, ok, complete := parse(message)

	assert.True(t, ok)
	assert.True(t, complete)
	assert.False(t, msg.isRequest)
	assert.Equal(t, "OK", string(msg.message))
	assert.Equal(t, len(message), msg.size)
}

func TestRedisParser_NilString(t *testing.T) {
	message := []byte("$-1\r\n")
	msg, ok, complete := parse(message)

	assert.True(t, ok)
	assert.True(t, complete)
	assert.False(t, msg.isRequest)
	assert.Equal(t, "nil", string(msg.message))
	assert.Equal(t, len(message), msg.size)
}

func TestRedisParser_EmptyString(t *testing.T) {
	message := []byte("$0\r\n\r\n")
	msg, ok, complete := parse(message)

	assert.True(t, ok)
	assert.True(t, complete)
	assert.False(t, msg.isRequest)
	assert.Equal(t, "", string(msg.message))
	assert.Equal(t, len(message), msg.size)
}

func TestRedisParser_LenString(t *testing.T) {
	message := []byte("$5\r\n" +
		"12345\r\n")
	msg, ok, complete := parse(message)

	assert.True(t, ok)
	assert.True(t, complete)
	assert.False(t, msg.isRequest)
	assert.Equal(t, "12345", string(msg.message))
	assert.Equal(t, len(message), msg.size)
}

func TestRedisParser_LenStringWithCRLF(t *testing.T) {
	message := []byte("$7\r\n" +
		"123\r\n45\r\n")
	msg, ok, complete := parse(message)

	assert.True(t, ok)
	assert.True(t, complete)
	assert.False(t, msg.isRequest)
	assert.Equal(t, "123\r\n45", string(msg.message))
	assert.Equal(t, len(message), msg.size)
}

func TestRedisParser_EmptyArray(t *testing.T) {
	message := []byte("*0\r\n")
	msg, ok, complete := parse(message)

	assert.True(t, ok)
	assert.True(t, complete)
	assert.False(t, msg.isRequest)
	assert.Equal(t, "[]", string(msg.message))
	assert.Equal(t, len(message), msg.size)
}

var (
	array2PassesPart1 = []byte("*3\r\n" +
		"$3\r\n" +
		"SET\r\n" +
		"$4\r\n")
	array2PassesPart2 = []byte("key1\r\n" +
		"$5\r\n" +
		"Hello\r\n")
)

func TestRedisParser_Array2Passes(t *testing.T) {
	st := newTestStream(array2PassesPart1)
	ok, complete := st.parser.parse(&st.Buf)
	assert.True(t, ok)
	assert.False(t, complete)

	st.Stream.Append(array2PassesPart2)
	ok, complete = st.parser.parse(&st.Buf)
	msg := st.parser.message

	assert.True(t, ok)
	assert.True(t, complete)
	assert.True(t, msg.isRequest)
	assert.Equal(t, "SET key1 Hello", string(msg.message))
	assert.Equal(t, len(array2PassesPart1)+len(array2PassesPart2), msg.size)
}

func BenchmarkParserNoArgsResult(b *testing.B) {
	for i := 0; i < b.N; i++ {
		parse(noArgsRequest)
	}
}

func BenchmarkParserArrayRequest(b *testing.B) {
	for i := 0; i < b.N; i++ {
		parse(arrayRequest)
	}
}

func BenchmarkParserArrayResponse(b *testing.B) {
	for i := 0; i < b.N; i++ {
		parse(arrayResponse)
	}
}

func BenchmarkParserArrayNested(b *testing.B) {
	for i := 0; i < b.N; i++ {
		parse(arrayNestedMessage)
	}
}

func BenchmarkParserArray2Passes(b *testing.B) {
	for i := 0; i < b.N; i++ {
		st := newTestStream(array2PassesPart1)
		st.parser.parse(&st.Buf)

		st.Stream.Append(array2PassesPart2)
		st.parser.parse(&st.Buf)
	}
}
