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

package thrift

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/packetbeat/procs"
	"github.com/elastic/beats/v7/packetbeat/protos"
	"github.com/elastic/beats/v7/packetbeat/protos/tcp"
	"github.com/elastic/beats/v7/pkg/logp"
)

func thriftForTests() *thriftPlugin {
	logger := logp.NewNopLogger()
	t := &thriftPlugin{
		logger:               logger.Named("thrift"),
		thriftDetailedLogger: logger.Named("thriftdetailed"),
	}
	config := defaultConfig
	t.init(true, nil, &procs.ProcessesWatcher{}, &config)
	return t
}

func TestThrift_thriftReadString(t *testing.T) {

	var data []byte
	var ok, complete bool
	var off int
	var str string

	thrift := thriftForTests()

	data, _ = hex.DecodeString("0000000470696e67")
	str, ok, complete, off = thrift.readString(data)
	if str != "ping" || !ok || !complete || off != 8 {
		t.Errorf("Bad result: %v %v %v %v", str, ok, complete, off)
	}

	data, _ = hex.DecodeString("0000000470696e670000")
	str, ok, complete, off = thrift.readString(data)
	if str != "ping" || !ok || !complete || off != 8 {
		t.Errorf("Bad result: %v %v %v %v", str, ok, complete, off)
	}

	data, _ = hex.DecodeString("0000000470696e")
	str, ok, complete, off = thrift.readString(data)
	if str != "" || !ok || complete || off != 0 {
		t.Errorf("Bad result: %v %v %v %v", str, ok, complete, off)
	}

	data, _ = hex.DecodeString("000000")
	str, ok, complete, off = thrift.readString(data)
	if str != "" || !ok || complete || off != 0 {
		t.Errorf("Bad result: %v %v %v %v", str, ok, complete, off)
	}
}

func TestThrift_readMessageBegin(t *testing.T) {

	var data []byte
	var ok, complete bool
	var stream thriftStream
	var m *thriftMessage

	var thrift thriftPlugin
	logger := logp.NewNopLogger()
	thrift.logger = logger.Named("thrift")
	thrift.thriftDetailedLogger = logger.Named("thriftdetailed")
	thrift.InitDefaults()

	data, _ = hex.DecodeString("800100010000000470696e670000000000")
	stream = thriftStream{data: data, message: new(thriftMessage)}
	m = stream.message
	ok, complete = thrift.readMessageBegin(&stream)
	if !ok || !complete {
		t.Errorf("Bad result: %v %v", ok, complete)
	}
	if m.method != "ping" || m.Type != ThriftMsgTypeCall ||
		m.seqID != 0 || m.version != thriftVersion1 {
		t.Errorf("Bad values: %v %v %v %v", m.method, m.Type, m.seqID, m.version)
	}

	data, _ = hex.DecodeString("800100010000000470696e6700000000")
	stream = thriftStream{data: data, message: new(thriftMessage)}
	m = stream.message
	ok, complete = thrift.readMessageBegin(&stream)
	if !ok || !complete {
		t.Errorf("Bad result: %v %v", ok, complete)
	}
	if m.method != "ping" || m.Type != ThriftMsgTypeCall ||
		m.seqID != 0 || m.version != thriftVersion1 {
		t.Errorf("Bad values: %v %v %v %v", m.method, m.Type, m.seqID, m.version)
	}

	data, _ = hex.DecodeString("800100010000000470696e6700000001")
	stream = thriftStream{data: data, message: new(thriftMessage)}
	m = stream.message
	ok, complete = thrift.readMessageBegin(&stream)
	if !ok || !complete {
		t.Errorf("Bad result: %v %v", ok, complete)
	}
	if m.method != "ping" || m.Type != ThriftMsgTypeCall ||
		m.seqID != 1 || m.version != thriftVersion1 {
		t.Errorf("Bad values: %v %v %v %v", m.method, m.Type, m.seqID, m.version)
	}

	data, _ = hex.DecodeString("800100010000000570696e6700000001")
	stream = thriftStream{data: data, message: new(thriftMessage)}
	ok, complete = thrift.readMessageBegin(&stream)
	if !ok || complete {
		t.Errorf("Bad result: %v %v", ok, complete)
	}

	data, _ = hex.DecodeString("800100010000000570696e6700000001")
	stream = thriftStream{data: data, message: new(thriftMessage)}
	ok, complete = thrift.readMessageBegin(&stream)
	if !ok || complete {
		t.Errorf("Bad result: %v %v", ok, complete)
	}

	data, _ = hex.DecodeString("0000000470696e670100000000")
	stream = thriftStream{data: data, message: new(thriftMessage)}
	m = stream.message
	ok, complete = thrift.readMessageBegin(&stream)
	if !ok || !complete {
		t.Errorf("Bad result: %v %v", ok, complete)
	}
	if m.method != "ping" || m.Type != ThriftMsgTypeCall ||
		m.seqID != 0 || m.version != 0 {
		t.Errorf("Bad values: %v %v %v %v", m.method, m.Type, m.seqID, m.version)
	}

	data, _ = hex.DecodeString("0000000570696e670100000000")
	stream = thriftStream{data: data, message: new(thriftMessage)}
	ok, complete = thrift.readMessageBegin(&stream)
	if !ok || complete {
		t.Error("Bad result:", ok, complete)
	}
}

func TestThrift_thriftReadField(t *testing.T) {
	var data []byte
	var ok, complete bool
	var stream thriftStream
	var field *thriftField
	var _old int

	var thrift thriftPlugin
	logger := logp.NewNopLogger()
	thrift.logger = logger.Named("thrift")
	thrift.thriftDetailedLogger = logger.Named("thriftdetailed")
	thrift.InitDefaults()

	data, _ = hex.DecodeString("08000100000001")
	stream = thriftStream{data: data, message: new(thriftMessage)}
	ok, complete, field = thrift.readField(&stream)
	if !ok || complete || field == nil {
		t.Error("Bad result:", ok, complete, field)
	} else {
		if field.id != 1 || field.Type != ThriftTypeI32 || field.value != "1" ||
			stream.parseOffset != len(stream.data) {
			t.Error("Bad values:", field.id, field.Type, field.value)
		}
	}

	data, _ = hex.DecodeString("0600010001")
	stream = thriftStream{data: data, message: new(thriftMessage)}
	ok, complete, field = thrift.readField(&stream)
	if !ok || complete || field == nil {
		t.Error("Bad result:", ok, complete, field)
	} else {
		if field.id != 1 || field.Type != ThriftTypeI16 || field.value != "1" ||
			stream.parseOffset != len(stream.data) {
			t.Error("Bad values:", field.id, field.Type, field.value)
		}
	}

	data, _ = hex.DecodeString("0a00010000000000000001")
	stream = thriftStream{data: data, message: new(thriftMessage)}
	ok, complete, field = thrift.readField(&stream)
	if !ok || complete || field == nil {
		t.Error("Bad result:", ok, complete, field)
	} else {
		if field.id != 1 || field.Type != ThriftTypeI64 || field.value != "1" ||
			stream.parseOffset != len(stream.data) {
			t.Error("Bad values:", field.id, field.Type, field.value)
		}
	}

	data, _ = hex.DecodeString("0400013ff3333333333333")
	stream = thriftStream{data: data, message: new(thriftMessage)}
	ok, complete, field = thrift.readField(&stream)
	if !ok || complete || field == nil {
		t.Error("Bad result:", ok, complete, field)
	} else {
		if field.id != 1 || field.Type != ThriftTypeDouble || field.value != "1.2" ||
			stream.parseOffset != len(stream.data) {
			t.Error("Bad values:", field.id, field.Type, field.value, stream.parseOffset)
		}
	}

	data, _ = hex.DecodeString("02000101")
	stream = thriftStream{data: data, message: new(thriftMessage)}
	ok, complete, field = thrift.readField(&stream)
	if !ok || complete || field == nil {
		t.Error("Bad result:", ok, complete, field)
	} else {
		if field.id != 1 || field.Type != ThriftTypeBool || field.value != "true" ||
			stream.parseOffset != len(stream.data) {
			t.Error("Bad values:", field.id, field.Type, field.value)
		}
	}

	data, _ = hex.DecodeString("0b00010000000568656c6c") // incomplete string
	stream = thriftStream{data: data, message: new(thriftMessage)}
	ok, complete, field = thrift.readField(&stream)
	if !ok || complete || field != nil {
		t.Error("Bad result:", ok, complete, field)
	}

	data, _ = hex.DecodeString("0b00010000000568656c6c6f")
	stream = thriftStream{data: data, message: new(thriftMessage)}
	ok, complete, field = thrift.readField(&stream)
	if !ok || complete || field == nil {
		t.Error("Bad result:", ok, complete, field)
	} else {
		if field.id != 1 || field.Type != ThriftTypeString || field.value != `"hello"` {
			t.Error("Bad values:", field.id, field.Type, field.value)
		}
	}

	_old, thrift.stringMaxSize = thrift.stringMaxSize, 3
	data, _ = hex.DecodeString("0b00010000000568656c6c6f")
	stream = thriftStream{data: data, message: new(thriftMessage)}
	ok, complete, field = thrift.readField(&stream)
	if !ok || complete || field == nil {
		t.Error("Bad result:", ok, complete, field)
	} else {
		if field.id != 1 || field.Type != ThriftTypeString || field.value != `"hel..."` ||
			stream.parseOffset != len(stream.data) {
			t.Error("Bad values:", field.id, field.Type, field.value)
		}
	}
	thrift.stringMaxSize = _old

	data, _ = hex.DecodeString("0f00010600000003000100020003")
	stream = thriftStream{data: data, message: new(thriftMessage)}
	ok, complete, field = thrift.readField(&stream)
	if !ok || complete || field == nil {
		t.Error("Bad result:", ok, complete, field)
	} else {
		if field.id != 1 || field.Type != ThriftTypeList ||
			field.value != "[1, 2, 3]" {
			t.Error("Bad values:", field.id, field.Type, field.value)
		}
	}

	_old, thrift.collectionMaxSize = thrift.collectionMaxSize, 1
	data, _ = hex.DecodeString("0f00010600000003000100020003")
	stream = thriftStream{data: data, message: new(thriftMessage)}
	ok, complete, field = thrift.readField(&stream)
	if !ok || complete || field == nil {
		t.Error("Bad result:", ok, complete, field)
	} else {
		if field.id != 1 || field.Type != ThriftTypeList ||
			stream.parseOffset != len(stream.data) ||
			field.value != "[1, ...]" {
			t.Error("Bad values:", field.id, field.Type, field.value)
		}
	}
	thrift.collectionMaxSize = _old

	data, _ = hex.DecodeString("0e0001060000000300010002000300")
	stream = thriftStream{data: data, message: new(thriftMessage)}
	ok, complete, field = thrift.readField(&stream)
	if !ok || complete || field == nil {
		t.Error("Bad result:", ok, complete, field)
	} else {
		if field.id != 1 || field.Type != ThriftTypeSet ||
			field.value != "{1, 2, 3}" {
			t.Error("Bad values:", field.id, field.Type, field.value)
		}
	}

	_old, thrift.collectionMaxSize = thrift.collectionMaxSize, 2
	data, _ = hex.DecodeString("0e00010600000003000100020003")
	stream = thriftStream{data: data, message: new(thriftMessage)}
	ok, complete, field = thrift.readField(&stream)
	if !ok || complete || field == nil {
		t.Error("Bad result:", ok, complete, field)
	} else {
		if field.id != 1 || field.Type != ThriftTypeSet ||
			stream.parseOffset != len(stream.data) ||
			field.value != "{1, 2, ...}" {
			t.Error("Bad values:", field.id, field.Type, field.value)
		}
	}
	thrift.collectionMaxSize = _old

	data, _ = hex.DecodeString("0d00010b0600000003000000016100010000000163000300000001620002")
	stream = thriftStream{data: data, message: new(thriftMessage)}
	ok, complete, field = thrift.readField(&stream)
	if !ok || complete || field == nil {
		t.Error("Bad result:", ok, complete, field)
	} else {
		if field.id != 1 || field.Type != ThriftTypeMap ||
			field.value != `{"a": 1, "c": 3, "b": 2}` ||
			stream.parseOffset != len(stream.data) {
			t.Error("Bad values:", field.id, field.Type, field.value)
		}
	}

	_old, thrift.collectionMaxSize = thrift.collectionMaxSize, 2
	data, _ = hex.DecodeString("0d00010b060000000300000001610001000000016300030000000162000200")
	stream = thriftStream{data: data, message: new(thriftMessage)}
	ok, complete, field = thrift.readField(&stream)
	if !ok || complete || field == nil {
		t.Error("Bad result:", ok, complete, field)
	} else {
		if field.id != 1 || field.Type != ThriftTypeMap ||
			field.value != `{"a": 1, "c": 3, ...}` {
			t.Error("Bad values:", field.id, field.Type, field.value)
		}
	}
	thrift.collectionMaxSize = _old

	data, _ = hex.DecodeString("0b00010000000568106c6c6f")
	stream = thriftStream{data: data, message: new(thriftMessage)}
	ok, complete, field = thrift.readField(&stream)
	if !ok || complete || field == nil {
		t.Error("Bad result:", ok, complete, field)
	} else {
		if field.id != 1 || field.Type != ThriftTypeString || field.value != `"h\x10llo"` {
			t.Error("Bad values:", field.id, field.Type, field.value)
		}
	}

	data, _ = hex.DecodeString("0c000108000100000001080002000000000800030000000400")
	stream = thriftStream{data: data, message: new(thriftMessage)}
	ok, complete, field = thrift.readField(&stream)
	if !ok || complete || field == nil {
		t.Error("Bad result:", ok, complete, field)
	} else {
		if field.id != 1 || field.Type != ThriftTypeStruct ||
			field.value != `(1: 1, 2: 0, 3: 4)` {
			t.Error("Bad values:", field.id, field.Type, field.value)
		}
	}

	_old, thrift.collectionMaxSize = thrift.collectionMaxSize, 2
	data, _ = hex.DecodeString("0c000108000100000001080002000000000800030000000400")
	stream = thriftStream{data: data, message: new(thriftMessage)}
	ok, complete, field = thrift.readField(&stream)
	if !ok || complete || field == nil {
		t.Error("Bad result:", ok, complete, field)
	} else {
		if field.id != 1 || field.Type != ThriftTypeStruct ||
			field.value != `(1: 1, 2: 0, ...)` {
			t.Error("Bad values:", field.id, field.Type, field.value)
		}
	}
	thrift.collectionMaxSize = _old

	data, _ = hex.DecodeString("0c0001080001000000010b00020000000568656c6c6f00")
	stream = thriftStream{data: data, message: new(thriftMessage)}
	ok, complete, field = thrift.readField(&stream)
	if !ok || complete || field == nil {
		t.Error("Bad result:", ok, complete, field)
	} else {
		if field.id != 1 || field.Type != ThriftTypeStruct ||
			field.value != `(1: 1, 2: "hello")` {
			t.Error("Bad values:", field.id, field.Type, field.value)
		}
	}

	data, _ = hex.DecodeString("0c0001080001000000010c0002080001000000010b00020000000568656c6c6f0000")
	stream = thriftStream{data: data, message: new(thriftMessage)}
	ok, complete, field = thrift.readField(&stream)
	if !ok || complete || field == nil {
		t.Error("Bad result:", ok, complete, field)
	} else {
		if field.id != 1 || field.Type != ThriftTypeStruct ||
			field.value != `(1: 1, 2: (1: 1, 2: "hello"))` {
			t.Error("Bad values:", field.id, field.Type, field.value)
		}
	}

	data, _ = hex.DecodeString("0c000008000000000000")
	stream = thriftStream{data: data, message: new(thriftMessage)}
	ok, complete, field = thrift.readField(&stream)
	require.False(t, complete, "bad packed marked as complete")

	data, _ = hex.DecodeString("0c0001080001000000010e0002060000000300010002000300")
	stream = thriftStream{data: data, message: new(thriftMessage)}
	ok, complete, field = thrift.readField(&stream)
	if !ok || complete || field == nil {
		t.Error("Bad result:", ok, complete, field)
	} else {
		if field.id != 1 || field.Type != ThriftTypeStruct ||
			field.value != `(1: 1, 2: {1, 2, 3})` {
			t.Error("Bad values:", field.id, field.Type, field.value)
		}
	}
}

// TestThrift_nestingDepth checks that deeply nested structs, lists, sets and
// maps are rejected as parse errors rather than exhausting the stack, which
// is a fatal error that cannot be recovered.
func TestThrift_nestingDepth(t *testing.T) {
	tests := []struct {
		name string
		// build returns an encoded value nested n levels deep.
		build func(n int) []byte
		read  func(*thriftPlugin, []byte) (string, bool, bool, int)
	}{
		{
			name: "struct",
			build: func(n int) []byte {
				return append(bytes.Repeat([]byte("\x0c\x00\x01"), n), make([]byte, n+1)...)
			},
			read: func(p *thriftPlugin, b []byte) (string, bool, bool, int) { return p.readStruct(b, 0) },
		},
		{
			name: "list",
			build: func(n int) []byte {
				b := bytes.Repeat([]byte("\x0f\x00\x00\x00\x01"), n)
				return append(b, "\x03\x00\x00\x00\x00"...)
			},
			read: func(p *thriftPlugin, b []byte) (string, bool, bool, int) { return p.readList(b, 0) },
		},
		{
			name: "set",
			build: func(n int) []byte {
				b := bytes.Repeat([]byte("\x0e\x00\x00\x00\x01"), n)
				return append(b, "\x03\x00\x00\x00\x00"...)
			},
			read: func(p *thriftPlugin, b []byte) (string, bool, bool, int) { return p.readSet(b, 0) },
		},
		{
			name: "map",
			build: func(n int) []byte {
				// Each level is a map with one entry: a byte key and a map value.
				b := bytes.Repeat([]byte("\x03\x0d\x00\x00\x00\x01\x00"), n)
				return append(b, "\x03\x03\x00\x00\x00\x00"...)
			},
			read: func(p *thriftPlugin, b []byte) (string, bool, bool, int) { return p.readMap(b, 0) },
		},
		{
			name: "map_key",
			build: func(n int) []byte {
				// Each level is a map with one entry: a map key and a byte value.
				b := bytes.Repeat([]byte("\x0d\x03\x00\x00\x00\x01"), n)
				b = append(b, "\x03\x03\x00\x00\x00\x00"...)
				return append(b, bytes.Repeat([]byte("\x00"), n)...)
			},
			read: func(p *thriftPlugin, b []byte) (string, bool, bool, int) { return p.readMap(b, 0) },
		},
	}

	for _, test := range tests {
		t.Run(test.name+"_within_limit", func(t *testing.T) {
			thrift := thriftForTests()
			_, ok, complete, _ := test.read(thrift, test.build(maxNestingDepth))
			assert.True(t, ok, "value nested %d levels should parse", maxNestingDepth)
			assert.True(t, complete, "value nested %d levels should be complete", maxNestingDepth)
		})
		t.Run(test.name+"_beyond_limit", func(t *testing.T) {
			thrift := thriftForTests()
			_, ok, _, _ := test.read(thrift, test.build(maxNestingDepth+1))
			assert.False(t, ok, "value nested %d levels should be rejected", maxNestingDepth+1)
		})
		t.Run(test.name+"_stack_exhaustion", func(t *testing.T) {
			// Without a depth limit this overflows the stack and kills the
			// test binary.
			thrift := thriftForTests()
			_, ok, _, _ := test.read(thrift, test.build(tcp.TCPMaxDataInStream/3))
			assert.False(t, ok, "extremely deep value should be rejected")
		})
	}
}

// TestThrift_nestingDepthParse checks the nesting limit through Parse, the
// path used in production, where an argument value is read at depth zero.
func TestThrift_nestingDepthParse(t *testing.T) {
	// nested returns a struct containing n levels of structs.
	var nested func(n int) string
	nested = func(n int) string {
		if n == 0 {
			return "00"
		}
		return "0c0001" + nested(n-1) + "00"
	}

	tests := []struct {
		name  string
		depth int
		want  bool
	}{
		{name: "at_limit", depth: maxNestingDepth - 1, want: true},
		{name: "beyond_limit", depth: maxNestingDepth, want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			thrift := thriftForTests()
			thrift.publishQueue = make(chan *thriftTransaction, 10)
			tcptuple := testTCPTuple()

			// The request has one argument holding the nested structs.
			req := createTestPacket(t, "800100010000000470696e67"+"00000000"+"0c0001"+nested(test.depth)+"00")
			repl := createTestPacket(t, "800100020000000470696e670000000000")

			priv := thrift.Parse(req, tcptuple, 0, nil)
			thrift.Parse(repl, tcptuple, 1, nil)

			got := false
			select {
			case trans := <-thrift.publishQueue:
				got = trans.request.method == "ping" && trans.request.params != ""
			default:
			}
			assert.Equal(t, test.want, got, "transaction published for %d nested structs", test.depth)

			if !test.want {
				data, ok := priv.(thriftPrivateData)
				require.True(t, ok, "unexpected private data type %T", priv)
				assert.Nil(t, data.data[0], "stream should be dropped")
			}
		})
	}
}

func TestParserBug1(t *testing.T) {
	thrift := thriftForTests()
	thrift.publishQueue = make(chan *thriftTransaction, 10)

	tcptuple := testTCPTuple()
	var private thriftPrivateData

	pkt := &protos.Packet{
		Payload: []byte("\x80000\x00\x00\x00\t0000000000000\f00\b000000"),
	}
	fmt.Printf("raw packet: %#v\n", pkt.Payload)
	thrift.Parse(pkt, tcptuple, 0, private)
}

func TestThrift_thriftMessageParser(t *testing.T) {

	var data []byte
	var stream thriftStream
	var ok, complete bool
	var m *thriftMessage

	var thrift thriftPlugin
	logger := logp.NewNopLogger()
	thrift.logger = logger.Named("thrift")
	thrift.thriftDetailedLogger = logger.Named("thriftdetailed")
	thrift.InitDefaults()

	data, _ = hex.DecodeString("800100010000000470696e670000000000")
	stream = thriftStream{data: data, message: new(thriftMessage)}
	ok, complete = thrift.messageParser(&stream)
	m = stream.message
	if !ok || !complete {
		t.Error("Bad result:", ok, complete)
	}
	if !m.isRequest || m.method != "ping" ||
		m.seqID != 0 || m.Type != ThriftMsgTypeCall || m.params != "()" {
		t.Error("Bad result:", stream.message)
	}

	data, _ = hex.DecodeString("800100010000000561646431360000000006000100010" +
		"60002000100")
	stream = thriftStream{data: data, message: new(thriftMessage)}
	ok, complete = thrift.messageParser(&stream)
	m = stream.message
	if !ok || !complete {
		t.Error("Bad result:", ok, complete)
	}
	if !m.isRequest || m.method != "add16" ||
		m.seqID != 0 || m.Type != ThriftMsgTypeCall ||
		m.params != "(1: 1, 2: 1)" {
		t.Error("Bad result:", stream.message)
	}

	data, _ = hex.DecodeString("800100010000000963616c63756c617465000000000" +
		"80001000000010c00020800010000000108000200000000080003000000040000")
	stream = thriftStream{data: data, message: new(thriftMessage)}
	ok, complete = thrift.messageParser(&stream)
	m = stream.message
	if !ok || !complete {
		t.Error("Bad result:", ok, complete)
	}
	if !m.isRequest || m.method != "calculate" ||
		m.seqID != 0 || m.Type != ThriftMsgTypeCall ||
		m.params != "(1: 1, 2: (1: 1, 2: 0, 3: 4))" {
		t.Error("Bad result:", stream.message)
	}

	data, _ = hex.DecodeString("8001000200000005616464313600000000060000000200")
	stream = thriftStream{data: data, message: new(thriftMessage)}
	ok, complete = thrift.messageParser(&stream)
	m = stream.message
	if !ok || !complete {
		t.Error("Bad result:", ok, complete)
	}
	if m.isRequest || m.method != "add16" ||
		m.seqID != 0 || m.Type != ThriftMsgTypeReply ||
		m.returnValue != "2" {
		t.Error("Bad result:", stream.message)
	}

	data, _ = hex.DecodeString("800100020000000b6563686f5f737472696e67000000000b00" +
		"000000000568656c6c6f00")
	stream = thriftStream{data: data, message: new(thriftMessage)}
	ok, complete = thrift.messageParser(&stream)
	m = stream.message
	if !ok || !complete {
		t.Error("Bad result:", ok, complete)
	}
	if m.isRequest || m.method != "echo_string" ||
		m.seqID != 0 || m.Type != ThriftMsgTypeReply ||
		m.returnValue != `"hello"` {
		t.Error("Bad result:", stream.message)
	}

	data, _ = hex.DecodeString("80010002000000096563686f5f6c697374000000000f0000060" +
		"000000300010002000300")
	stream = thriftStream{data: data, message: new(thriftMessage)}
	ok, complete = thrift.messageParser(&stream)
	m = stream.message
	if !ok || !complete {
		t.Error("Bad result:", ok, complete)
	}
	if m.isRequest || m.method != "echo_list" ||
		m.seqID != 0 || m.Type != ThriftMsgTypeReply ||
		m.returnValue != `[1, 2, 3]` {
		t.Error("Bad result:", stream.message)
	}

	data, _ = hex.DecodeString("80010002000000086563686f5f6d6170000000000d00000b06000" +
		"0000300000001610001000000016300030000000162000200")
	stream = thriftStream{data: data, message: new(thriftMessage)}
	ok, complete = thrift.messageParser(&stream)
	m = stream.message
	if !ok || !complete {
		t.Error("Bad result:", ok, complete)
	}
	if m.isRequest || m.method != "echo_map" ||
		m.seqID != 0 || m.Type != ThriftMsgTypeReply ||
		m.returnValue != `{"a": 1, "c": 3, "b": 2}` {
		t.Error("Bad result:", stream.message)
	}

	data, _ = hex.DecodeString("800100020000000963616c63756c617465000000000800000000000500")
	stream = thriftStream{data: data, message: new(thriftMessage)}
	ok, complete = thrift.messageParser(&stream)
	m = stream.message
	if !ok || !complete {
		t.Error("Bad result:", ok, complete)
	}
	if m.isRequest || m.method != "calculate" ||
		m.seqID != 0 || m.Type != ThriftMsgTypeReply || m.hasException ||
		m.returnValue != `5` {
		t.Error("Bad result:", stream.message)
	}

	data, _ = hex.DecodeString("800100020000000963616c63756c617465000000000c000108000100" +
		"0000040b00020000001243616e6e6f742064697669646520627920300000")
	stream = thriftStream{data: data, message: new(thriftMessage)}
	ok, complete = thrift.messageParser(&stream)
	m = stream.message
	if !ok || !complete {
		t.Error("Bad result:", ok, complete)
	}
	if m.isRequest || m.method != "calculate" ||
		m.seqID != 0 || m.Type != ThriftMsgTypeReply || !m.hasException ||
		m.exceptions != `(1: (1: 4, 2: "Cannot divide by 0"))` {
		t.Error("Bad result:", stream.message)
	}

	data, _ = hex.DecodeString("800100020000000b6563686f5f62696e61727900000000" +
		"0b000000000008ab0c1d281a00000000")
	stream = thriftStream{data: data, message: new(thriftMessage)}
	ok, complete = thrift.messageParser(&stream)
	m = stream.message
	if !ok || !complete {
		t.Error("Bad result:", ok, complete)
	}
	if m.isRequest || m.method != "echo_binary" ||
		m.seqID != 0 || m.Type != ThriftMsgTypeReply || m.hasException ||
		m.returnValue != `ab0c1d281a000000` {
		t.Error("Bad result:", stream.message)
	}
}

// Returns a minimally filled Packet struct, only the payload
// having value.
func createTestPacket(t *testing.T, hexstr string) *protos.Packet {
	data, err := hex.DecodeString(hexstr)
	if err != nil {
		t.Error("Failed to decode hex string")
		return nil
	}

	return &protos.Packet{
		Payload: data,
	}
}

// Helper function to read from the Publisher Queue
func expectThriftTransaction(t *testing.T, thrift *thriftPlugin) *thriftTransaction {
	select {
	case trans := <-thrift.publishQueue:
		return trans
	default:
		t.Error("No transaction")
	}
	return nil
}

func testTCPTuple() *common.TCPTuple {
	t := &common.TCPTuple{
		IPLength: 4,
		BaseTuple: common.BaseTuple{
			SrcIP: net.IPv4(192, 168, 0, 1), DstIP: net.IPv4(192, 168, 0, 2),
			SrcPort: 9200, DstPort: 9201,
		},
	}
	t.ComputeHashables()
	return t
}

func TestThrift_ParseSimpleTBinary(t *testing.T) {

	thrift := thriftForTests()
	thrift.publishQueue = make(chan *thriftTransaction, 10)

	tcptuple := testTCPTuple()
	req := createTestPacket(t, "800100010000000470696e670000000000")
	repl := createTestPacket(t, "800100020000000470696e670000000000")

	var private thriftPrivateData
	thrift.Parse(req, tcptuple, 0, private)
	thrift.Parse(repl, tcptuple, 1, private)

	trans := expectThriftTransaction(t, thrift)
	if trans.request.method != "ping" ||
		trans.request.params != "()" ||
		trans.reply.returnValue != "" ||
		trans.request.frameSize == 0 ||
		trans.reply.frameSize == 0 {

		t.Error("Bad result:", trans)
	}
}

func TestThrift_ParseSimpleTFramed(t *testing.T) {

	thrift := thriftForTests()
	thrift.TransportType = thriftTFramed
	thrift.publishQueue = make(chan *thriftTransaction, 10)

	tcptuple := testTCPTuple()

	req := createTestPacket(t, "0000001e8001000100000003616464000000000800010000000108"+
		"00020000000100")
	repl := createTestPacket(t, "000000178001000200000003616464000000000800000000000200")

	var private thriftPrivateData

	thrift.Parse(req, tcptuple, 0, private)
	thrift.Parse(repl, tcptuple, 1, private)

	trans := expectThriftTransaction(t, thrift)
	if trans.request.method != "add" ||
		trans.request.params != "(1: 1, 2: 1)" ||
		trans.reply.returnValue != "2" {

		t.Error("Bad result:", trans)
	}
}

func TestThrift_ParseSimpleTFramedSplit(t *testing.T) {

	thrift := thriftForTests()
	thrift.TransportType = thriftTFramed
	thrift.publishQueue = make(chan *thriftTransaction, 10)

	tcptuple := testTCPTuple()

	reqHalf1 := createTestPacket(t, "0000001e8001000100")
	reqHalf2 := createTestPacket(t, "000003616464000000000800010000000108"+
		"00020000000100")
	replHalf1 := createTestPacket(t, "000000178001000200000003")
	replHalf2 := createTestPacket(t, "616464000000000800000000000200")

	var private thriftPrivateData
	private = thrift.Parse(reqHalf1, tcptuple, 0, private).(thriftPrivateData)
	private = thrift.Parse(reqHalf2, tcptuple, 0, private).(thriftPrivateData)
	private = thrift.Parse(replHalf1, tcptuple, 1, private).(thriftPrivateData)
	thrift.Parse(replHalf2, tcptuple, 1, private)

	trans := expectThriftTransaction(t, thrift)
	if trans.request.method != "add" ||
		trans.request.params != "(1: 1, 2: 1)" ||
		trans.reply.returnValue != "2" {

		t.Error("Bad result:", trans)
	}
}

func TestThrift_ParseSimpleTFramedSplitInterleaved(t *testing.T) {

	thrift := thriftForTests()
	thrift.TransportType = thriftTFramed
	thrift.publishQueue = make(chan *thriftTransaction, 10)

	tcptuple := testTCPTuple()

	reqHalf1 := createTestPacket(t, "0000001e8001000100")
	replHalf1 := createTestPacket(t, "000000178001000200000003")
	reqHalf2 := createTestPacket(t, "000003616464000000000800010000000108"+
		"00020000000100")
	replHalf2 := createTestPacket(t, "616464000000000800000000000200")

	var private thriftPrivateData
	private = thrift.Parse(reqHalf1, tcptuple, 0, private).(thriftPrivateData)
	private = thrift.Parse(reqHalf2, tcptuple, 0, private).(thriftPrivateData)
	private = thrift.Parse(replHalf1, tcptuple, 1, private).(thriftPrivateData)
	thrift.Parse(replHalf2, tcptuple, 1, private)

	trans := expectThriftTransaction(t, thrift)
	if trans.request.method != "add" ||
		trans.request.params != "(1: 1, 2: 1)" ||
		trans.reply.returnValue != "2" {

		t.Error("Bad result:", trans)
	}
}

func TestThrift_Parse_OneWayCallWithFin(t *testing.T) {

	thrift := thriftForTests()
	thrift.TransportType = thriftTFramed
	thrift.publishQueue = make(chan *thriftTransaction, 10)

	tcptuple := testTCPTuple()

	req := createTestPacket(t, "0000001080010001000000037a69700000000000")

	var private thriftPrivateData
	thrift.Parse(req, tcptuple, 0, private)
	thrift.ReceivedFin(tcptuple, 0, private)

	trans := expectThriftTransaction(t, thrift)
	if trans.request.method != "zip" ||
		trans.request.params != "()" ||
		trans.reply != nil {

		t.Error("Bad result:", trans)
	}
}

func TestThrift_Parse_OneWayCall2Requests(t *testing.T) {
	thrift := thriftForTests()
	thrift.TransportType = thriftTFramed
	thrift.publishQueue = make(chan *thriftTransaction, 10)

	tcptuple := testTCPTuple()

	reqzip := createTestPacket(t, "0000001080010001000000037a69700000000000")
	req := createTestPacket(t, "0000001e8001000100000003616464000000000800010000000108"+
		"00020000000100")
	repl := createTestPacket(t, "000000178001000200000003616464000000000800000000000200")

	var private thriftPrivateData
	thrift.Parse(reqzip, tcptuple, 0, private)
	thrift.Parse(req, tcptuple, 0, private)
	thrift.Parse(repl, tcptuple, 1, private)

	trans := expectThriftTransaction(t, thrift)
	if trans.request.method != "zip" ||
		trans.request.params != "()" ||
		trans.reply != nil {

		t.Error("Bad result:", trans)
	}

	trans = expectThriftTransaction(t, thrift)
	if trans.request.method != "add" ||
		trans.request.params != "(1: 1, 2: 1)" ||
		trans.reply.returnValue != "2" {

		t.Error("Bad result:", trans)
	}
}

func TestThrift_Parse_RequestReplyMismatch(t *testing.T) {

	thrift := thriftForTests()
	thrift.TransportType = thriftTFramed
	thrift.publishQueue = make(chan *thriftTransaction, 10)

	tcptuple := testTCPTuple()

	reqzip := createTestPacket(t, "0000001080010001000000037a69700000000000")
	repladd := createTestPacket(t, "000000178001000200000003616464000000000800000000000200")

	var private thriftPrivateData
	thrift.Parse(reqzip, tcptuple, 0, private)
	thrift.Parse(repladd, tcptuple, 1, private)

	// Nothing should be received at this point
	select {
	case trans := <-thrift.publishQueue:
		t.Error("Bad result:", trans)
	default:
		// ok
	}
}

func TestThrift_ParseSimpleTFramed_NoReply(t *testing.T) {

	thrift := thriftForTests()
	thrift.TransportType = thriftTFramed
	thrift.captureReply = false
	thrift.publishQueue = make(chan *thriftTransaction, 10)

	tcptuple := testTCPTuple()

	req := createTestPacket(t, "0000001e8001000100000003616464000000000800010000000108"+
		"00020000000100")
	repl := createTestPacket(t, "000000178001000200000003616464000000000800000000000200")

	var private thriftPrivateData
	thrift.Parse(req, tcptuple, 0, private)
	thrift.Parse(repl, tcptuple, 1, private)

	trans := expectThriftTransaction(t, thrift)
	if trans.request.method != "add" ||
		trans.request.params != "(1: 1, 2: 1)" ||
		trans.reply.returnValue != "" {

		t.Error("Bad result:", trans)
	}

	// play it again in the same stream
	thrift.Parse(req, tcptuple, 0, private)
	thrift.Parse(repl, tcptuple, 1, private)

	trans = expectThriftTransaction(t, thrift)
	if trans.request.method != "add" ||
		trans.request.params != "(1: 1, 2: 1)" ||
		trans.reply.returnValue != "" {

		t.Error("Bad result:", trans)
	}
}

func TestThrift_ParseObfuscateStrings(t *testing.T) {

	thrift := thriftForTests()
	thrift.TransportType = thriftTFramed
	thrift.obfuscateStrings = true
	thrift.publishQueue = make(chan *thriftTransaction, 10)

	tcptuple := testTCPTuple()

	req := createTestPacket(t, "00000024800100010000000b6563686f5f737472696e670000"+
		"00000b00010000000568656c6c6f00")
	repl := createTestPacket(t, "00000024800100020000000b6563686f5f737472696e67000"+
		"000000b00000000000568656c6c6f00")

	var private thriftPrivateData
	thrift.Parse(req, tcptuple, 0, private)
	thrift.Parse(repl, tcptuple, 1, private)

	trans := expectThriftTransaction(t, thrift)
	if trans.request.method != "echo_string" ||
		trans.request.params != `(1: "*")` ||
		trans.reply.returnValue != `"*"` {

		t.Error("Bad result:", trans)
	}
}

func BenchmarkThrift_ParseSkipReply(b *testing.B) {

	thrift := thriftForTests()
	thrift.TransportType = thriftTFramed
	thrift.publishQueue = make(chan *thriftTransaction, 10)
	thrift.captureReply = false

	tcptuple := testTCPTuple()

	dataReq, _ := hex.DecodeString("0000001e8001000100000003616464000000000800010000000108" +
		"00020000000100")
	req := &protos.Packet{Payload: dataReq}
	dataRepl, _ := hex.DecodeString("000000178001000200000003616464000000000800000000000200")
	repl := &protos.Packet{Payload: dataRepl}

	var private thriftPrivateData
	for n := 0; n < b.N; n++ {
		thrift.Parse(req, tcptuple, 0, private)
		thrift.Parse(repl, tcptuple, 1, private)

		select {
		case trans := <-thrift.publishQueue:

			if trans.request.method != "add" ||
				trans.request.params != "(1: 1, 2: 1)" {

				b.Error("Bad result:", trans)
			}
		default:
			b.Error("No transaction")
		}

		// next should be empty
		select {
		case trans := <-thrift.publishQueue:
			b.Error("Transaction still in queue: ", trans)
		default:
			// ok
		}
	}
}

func TestThrift_Parse_Exception(t *testing.T) {

	thrift := thriftForTests()
	thrift.publishQueue = make(chan *thriftTransaction, 10)

	tcptuple := testTCPTuple()

	req := createTestPacket(t, "800100010000000963616c63756c6174650000000008000"+
		"1000000010c00020800010000000108000200000000080003000000040000")
	repl := createTestPacket(t, "800100020000000963616c63756c617465000000000c00"+
		"01080001000000040b00020000001243616e6e6f742064697669646520627920300000")

	var private thriftPrivateData
	thrift.Parse(req, tcptuple, 0, private)
	thrift.Parse(repl, tcptuple, 1, private)

	trans := expectThriftTransaction(t, thrift)
	if trans.request.method != "calculate" ||
		trans.request.params != "(1: 1, 2: (1: 1, 2: 0, 3: 4))" ||
		trans.reply.exceptions != `(1: (1: 4, 2: "Cannot divide by 0"))` ||
		!trans.reply.hasException {

		t.Error("Bad result:", trans)
	}
}

func TestThrift_ParametersNames(t *testing.T) {

	thrift := thriftForTests()
	thrift.TransportType = thriftTFramed
	thrift.idl = thriftIdlForTesting(t, `
		service Test {
			   i32 add(1:i32 num1, 2: i32 num2)
		}
		`)

	thrift.publishQueue = make(chan *thriftTransaction, 10)

	tcptuple := testTCPTuple()

	req := createTestPacket(t, "0000001e8001000100000003616464000000000800010000000108"+
		"00020000000100")
	repl := createTestPacket(t, "000000178001000200000003616464000000000800000000000200")

	var private thriftPrivateData
	thrift.Parse(req, tcptuple, 0, private)
	thrift.Parse(repl, tcptuple, 1, private)

	trans := expectThriftTransaction(t, thrift)
	if trans.request.method != "add" ||
		trans.request.params != "(num1: 1, num2: 1)" ||
		trans.reply.returnValue != "2" ||
		trans.request.service != "Test" {

		t.Error("Bad result:", trans)
	}
}

func TestThrift_ExceptionName(t *testing.T) {

	thrift := thriftForTests()
	thrift.idl = thriftIdlForTesting(t, `
		exception InvalidOperation {
		  1: i32 what,
		  2: string why
		}
		service Test {
		   i32 calculate(1:i32 logid, 2:Work w) throws (1:InvalidOperation ouch),
		}
		`)

	thrift.publishQueue = make(chan *thriftTransaction, 10)

	tcptuple := testTCPTuple()

	req := createTestPacket(t, "800100010000000963616c63756c6174650000000008000"+
		"1000000010c00020800010000000108000200000000080003000000040000")
	repl := createTestPacket(t, "800100020000000963616c63756c617465000000000c00"+
		"01080001000000040b00020000001243616e6e6f742064697669646520627920300000")

	var private thriftPrivateData
	thrift.Parse(req, tcptuple, 0, private)
	thrift.Parse(repl, tcptuple, 1, private)

	trans := expectThriftTransaction(t, thrift)
	if trans.request.method != "calculate" ||
		trans.request.params != "(logid: 1, w: (1: 1, 2: 0, 3: 4))" ||
		trans.reply.returnValue != "" ||
		trans.reply.exceptions != `(ouch: (1: 4, 2: "Cannot divide by 0"))` ||
		!trans.reply.hasException ||
		trans.request.service != "Test" {

		t.Error("Bad result:", trans)
	}
}

func TestThrift_GapInStream_response(t *testing.T) {

	thrift := thriftForTests()
	thrift.idl = thriftIdlForTesting(t, `
		exception InvalidOperation {
		  1: i32 what,
		  2: string why
		}
		service Test {
		   i32 calculate(1:i32 logid, 2:Work w) throws (1:InvalidOperation ouch),
		}
		`)

	thrift.publishQueue = make(chan *thriftTransaction, 10)

	tcptuple := testTCPTuple()

	req := createTestPacket(t, "800100010000000963616c63756c6174650000000008000"+
		"1000000010c00020800010000000108000200000000080003000000040000")
	// missing last few bytes
	repl := createTestPacket(t, "800100020000000963616c63756c617465000000000c00"+
		"01080001000000040b00020000001243616e6e6f742064697669646520")

	private := protos.ProtocolData(new(thriftPrivateData))
	private = thrift.Parse(req, tcptuple, 0, private)
	private = thrift.Parse(repl, tcptuple, 1, private)
	_, drop := thrift.GapInStream(tcptuple, 1, 5, private)
	if drop == false {
		t.Error("GapInStream returned drop=false")
	}

	trans := expectThriftTransaction(t, thrift)
	// The exception is not captured, but otherwise the values from the request
	// are correct
	if trans.request.method != "calculate" ||
		trans.request.params != "(logid: 1, w: (1: 1, 2: 0, 3: 4))" ||
		trans.reply.returnValue != "" ||
		trans.reply.exceptions != `` ||
		trans.reply.hasException ||
		trans.request.service != "Test" ||
		trans.reply.notes[0] != "Packet loss while capturing the response" {

		t.Error("trans.Reply.Exceptions", trans.reply.exceptions)
		t.Error("trans.Reply.HasException", trans.reply.hasException)
	}
}

func TestThrift_GapInStream_request(t *testing.T) {

	thrift := thriftForTests()
	thrift.idl = thriftIdlForTesting(t, `
		exception InvalidOperation {
		  1: i32 what,
		  2: string why
		}
		service Test {
		   i32 calculate(1:i32 logid, 2:Work w) throws (1:InvalidOperation ouch),
		}
		`)

	thrift.publishQueue = make(chan *thriftTransaction, 10)

	tcptuple := testTCPTuple()

	// missing bytes from the request
	req := createTestPacket(t, "800100010000000963616c63756c6174")
	repl := createTestPacket(t, "800100020000000963616c63756c617465000000000c00"+
		"01080001000000040b00020000001243616e6e6f742064697669646520627920300000")

	private := protos.ProtocolData(new(thriftPrivateData))
	private = thrift.Parse(req, tcptuple, 0, private)
	private, drop := thrift.GapInStream(tcptuple, 0, 5, private)

	thrift.Parse(repl, tcptuple, 1, private)
	if drop == false {
		t.Error("GapInStream returned drop=false")
	}

	// packet loss in requests should result in no transaction
	select {
	case trans := <-thrift.publishQueue:
		t.Error("Expected no transaction but got one:", trans)
	default:
		// ok
	}
}
