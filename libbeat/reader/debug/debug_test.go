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

package debug

import (
	"bytes"
	"io"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/logp"
)

func TestMakeNullCheck(t *testing.T) {
	t.Run("return true if null byte is received", func(t *testing.T) {
		check := makeNullCheck(logp.NewLogger("detect-null"), 1)
		assert.True(t, check(100, []byte{'a', 'b', 'c', 0x0, 'd'}))
	})

	t.Run("return false on anything other bytes", func(t *testing.T) {
		check := makeNullCheck(logp.NewLogger("detect-null"), 1)
		assert.False(t, check(100, []byte{'a', 'b', 'c', 'd'}))
	})

	t.Run("return true when a slice of bytes is present", func(t *testing.T) {
		check := makeNullCheck(logp.NewLogger("detect-null"), 3)
		assert.True(t, check(100, []byte{'a', 'b', 'c', 0x0, 0x0, 0x0, 'd'}))
	})
}

func TestSummarizeBufferInfo(t *testing.T) {
	t.Run("when position is the start of the buffer", func(t *testing.T) {
		relativePos, surround := summarizeBufferInfo(0, []byte("hello world"))
		assert.Equal(t, 0, relativePos)
		assert.Equal(t, []byte("hello world"), surround)
	})

	t.Run("when position is not the start of the buffer", func(t *testing.T) {
		c, _ := common.RandomBytes(10000)
		relativePos, surround := summarizeBufferInfo(200, c)
		assert.Equal(t, 100, relativePos)
		assert.Equal(t, 200, len(surround))
	})
}

func TestReader(t *testing.T) {
	t.Run("check that we check the content of byte", testCheckContent)
	t.Run("consume all bytes", testConsumeAll)
	t.Run("empty buffer", testEmptyBuffer)
	t.Run("should become silent after hitting max failures", testSilent)
}

func testCheckContent(t *testing.T) {
	var c int
	check := func(_ int64, _ []byte) bool {
		c++
		return true
	}

	var s bytes.Buffer
	s.WriteString("hello world")
	s.WriteByte(0x00)
	s.WriteString("hello world")
	r := ioutil.NopCloser(&s)

	reader, _ := NewReader(logp.L(), r, 5, 3, check)

	_, err := reader.Read(make([]byte, 20))
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, 1, c)
}

func testConsumeAll(t *testing.T) {
	c, _ := common.RandomBytes(2000)
	reader := ioutil.NopCloser(bytes.NewReader(c))
	var buf bytes.Buffer
	consumed := 0
	debug, _ := NewReader(logp.L(), reader, 8, 20, makeNullCheck(logp.L(), 1))
	for consumed < 2000 {
		data := make([]byte, 33)
		n, _ := debug.Read(data)
		buf.Write(data[:n])
		consumed += n
	}
	assert.Equal(t, len(c), consumed)
	assert.Equal(t, c, buf.Bytes())
}

func testEmptyBuffer(t *testing.T) {
	buf := ioutil.NopCloser(&bytes.Buffer{})
	debug, _ := NewReader(logp.L(), buf, 8, 20, makeNullCheck(logp.L(), 1))
	data := make([]byte, 33)
	n, err := debug.Read(data)
	assert.Equal(t, io.EOF, err)
	assert.Equal(t, 0, n)
}

func testSilent(t *testing.T) {
	var c int
	check := func(_ int64, buf []byte) bool {
		pattern := make([]byte, 1, 1)
		idx := bytes.Index(buf, pattern)
		if idx <= 0 {
			return false
		}
		c++
		return true
	}

	var b bytes.Buffer
	b.Write([]byte{'a', 'b', 'c', 'd', 0x00, 'e'})
	b.Write([]byte{'a', 'b', 'c', 'd', 0x00, 'e'})
	b.Write([]byte{'a', 'b', 'c', 'd', 0x00, 'e'})
	b.Write([]byte{'a', 'b', 'c', 'd', 0x00, 'e'})
	b.Write([]byte{'a', 'b', 'c', 'd', 0x00, 'e'})
	b.Write([]byte{'a', 'b', 'c', 'd', 0x00, 'e'})
	b.Write([]byte{'a', 'b', 'c', 'd', 0x00, 'e'})
	r := ioutil.NopCloser(&b)

	debug, _ := NewReader(logp.L(), r, 3, 2, check)
	consumed := 0
	for consumed < b.Len() {
		n, _ := debug.Read(make([]byte, 3))
		consumed += n
	}
	assert.Equal(t, 2, c)
	assert.Equal(t, consumed, b.Len())
}
