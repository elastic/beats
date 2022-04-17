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

//go:build !integration
// +build !integration

package memcache

import (
	"testing"
	"time"

	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/common/streambuf"
)

type testParser struct {
	testing *testing.T
	parser  *parser
	buf     *streambuf.Buffer
}

type binHeader struct {
	request   bool
	opcode    memcacheOpcode
	keyLen    uint16
	extrasLen uint8
	status    uint16
	valueLen  uint32
	opaque    uint32
	cas       uint64
}

type binValueWriter interface {
	WriteNetUint8(uint8) error
	WriteNetUint16(uint16) error
	WriteNetUint32(uint32) error
	WriteNetUint64(uint64) error
	WriteNetUint8At(uint8, int) error
	WriteNetUint16At(uint16, int) error
	WriteNetUint32At(uint32, int) error
	WriteNetUint64At(uint64, int) error
}

type (
	extraFn func(binValueWriter) int
	valueFn func(*streambuf.Buffer, int) int
)

type offsetBinWriter struct {
	w      binValueWriter
	offset int
}

var defaultTestParserConfig = parserConfig{
	maxValues:        -1,
	maxBytesPerValue: 2e6,
	parseUnknown:     true,
}

func newTestParser(tst *testing.T, state parserState) *testParser {
	t := &testParser{
		testing: tst,
		parser:  newParser(&defaultTestParserConfig),
		buf:     streambuf.New(nil),
	}
	return t
}

func newTextTestParser(t *testing.T) *testParser {
	return newTestParser(t, parseStateTextCommand)
}

func newBinTestParser(t *testing.T) *testParser {
	return newTestParser(t, parseStateBinaryCommand)
}

func (tp *testParser) parse(d []byte) (*message, error) {
	if err := tp.buf.Append(d); err != nil {
		tp.testing.Fatalf("parser buffer append error: %v", err)
		return nil, nil
	}
	return tp.parser.parse(tp.buf, time.Now())
}

func (tp *testParser) text(d string) (*message, error) {
	return tp.parse([]byte(d))
}

func (tp *testParser) parseNoFail(d []byte) *message {
	if err := tp.buf.Append(d); err != nil {
		tp.testing.Fatalf("parser buffer append error: %v", err)
		return nil
	}
	msg, err := tp.parser.parse(tp.buf, time.Now())
	if err != nil {
		tp.testing.Fatalf("parser unexpectedly failed with: %v", err)
		return nil
	}
	return msg
}

func (tp *testParser) textNoFail(d string) *message {
	return tp.parseNoFail([]byte(d))
}

func textTryParse(t *testing.T, d string) (*message, error) {
	return newTextTestParser(t).text(d)
}

func textParseNoFail(t *testing.T, d string) *message {
	return newTextTestParser(t).textNoFail(d)
}

func binTryParse(t *testing.T, buf []byte) (*message, error) {
	return newBinTestParser(t).parse(buf)
}

func binParseNoFail(t *testing.T, buf []byte) *message {
	return newBinTestParser(t).parseNoFail(buf)
}

func (h *binHeader) write(buf binValueWriter) {
	if h.request {
		buf.WriteNetUint8At(memcacheMagicRequest, 0)
	} else {
		buf.WriteNetUint8At(memcacheMagicResponse, 0)
		buf.WriteNetUint16At(h.status, 6)
	}
	total := uint32(h.extrasLen) + uint32(h.keyLen) + h.valueLen
	buf.WriteNetUint8At(uint8(h.opcode), 1)
	buf.WriteNetUint16At(h.keyLen, 2)
	buf.WriteNetUint8At(h.extrasLen, 4)
	buf.WriteNetUint32At(total, 8)
	buf.WriteNetUint32At(h.opaque, 12)
	buf.WriteNetUint64At(h.cas, 16)
}

func extra32Bit(x uint32) extraFn {
	return func(buf binValueWriter) int {
		buf.WriteNetUint32(x)
		return 4
	}
}

func extra64Bit(x uint64) extraFn {
	return func(buf binValueWriter) int {
		buf.WriteNetUint64(x)
		return 8
	}
}

var noKey valueFn = func(buf *streambuf.Buffer, off int) int {
	return 0
}

var noValue = noKey

var key = value

func value(k string) valueFn {
	tmp := []byte(k)
	return func(buf *streambuf.Buffer, off int) int {
		if len(tmp) == 0 {
			return 0
		}
		buf.WriteAt(tmp, int64(off))
		return len(tmp)
	}
}

func binValue(b []byte) valueFn {
	return func(buf *streambuf.Buffer, off int) int {
		if len(b) == 0 {
			return 0
		}
		buf.WriteAt(b, int64(off))
		return len(b)
	}
}

func extras(es ...extraFn) []extraFn {
	return es
}

func (b *offsetBinWriter) WriteNetUint8(u uint8) error {
	err := b.WriteNetUint8At(u, 0)
	b.offset++
	return err
}

func (b *offsetBinWriter) WriteNetUint16(u uint16) error {
	err := b.WriteNetUint16At(u, 0)
	b.offset += 2
	return err
}

func (b *offsetBinWriter) WriteNetUint32(u uint32) error {
	err := b.WriteNetUint32At(u, 0)
	b.offset += 4
	return err
}

func (b *offsetBinWriter) WriteNetUint64(u uint64) error {
	err := b.WriteNetUint64At(u, 0)
	b.offset += 8
	return err
}

func (b *offsetBinWriter) WriteNetUint8At(u uint8, i int) error {
	return b.w.WriteNetUint8At(u, i+b.offset)
}

func (b *offsetBinWriter) WriteNetUint16At(u uint16, i int) error {
	return b.w.WriteNetUint16At(u, i+b.offset)
}

func (b *offsetBinWriter) WriteNetUint32At(u uint32, i int) error {
	return b.w.WriteNetUint32At(u, i+b.offset)
}

func (b *offsetBinWriter) WriteNetUint64At(u uint64, i int) error {
	return b.w.WriteNetUint64At(u, i+b.offset)
}

func genBinMessage(
	hdr *binHeader,
	extras []extraFn,
	key valueFn,
	value valueFn,
) func(*streambuf.Buffer) error {
	return func(buf *streambuf.Buffer) error {
		offset := memcacheHeaderSize

		var extraLen int
		extraWriter := &offsetBinWriter{buf, memcacheHeaderSize}
		for _, e := range extras {
			extraLen += e(extraWriter)
		}
		hdr.extrasLen = uint8(extraLen)
		offset += extraLen

		keyLen := key(buf, offset)
		offset += keyLen
		hdr.keyLen = uint16(keyLen)

		valLen := value(buf, offset)
		hdr.valueLen = uint32(valLen)

		hdr.write(buf)
		return buf.Err()
	}
}

func prepareBinMessage(
	hdr *binHeader,
	extras []extraFn,
	key valueFn,
	value valueFn,
) (*streambuf.Buffer, error) {
	buf := streambuf.New(nil)
	gen := genBinMessage(hdr, extras, key, value)
	err := gen(buf)
	return buf, err
}

func makeMessageEvent(t *testing.T, msg *message) common.MapStr {
	event := common.MapStr{}
	err := msg.Event(event)
	if err != nil {
		t.Fatalf("generating message event structure failed with: %v", err)
	}
	return event
}
