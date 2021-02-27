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

package memcache

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_BinParseQuitCommand(t *testing.T) {
	buf, _ := prepareBinMessage(
		&binHeader{opcode: opcodeQuit, request: true},
		extras(),
		noKey, noValue)

	msg := binParseNoFail(t, buf.Bytes())
	assert.NotNil(t, msg)
	assert.Equal(t, memcacheCmdQuit, msg.command.code)
	assert.False(t, msg.isQuiet)

	buf, _ = prepareBinMessage(
		&binHeader{opcode: opcodeQuitQ, request: true},
		extras(),
		noKey, noValue)
	msg = binParseNoFail(t, buf.Bytes())
	assert.NotNil(t, msg)
	assert.Equal(t, memcacheCmdQuit, msg.command.code)
	assert.True(t, msg.isQuiet)
}

func Test_BinParseUnknownCommand(t *testing.T) {
	buf, _ := prepareBinMessage(&binHeader{opcode: 0xff}, extras(), noKey, noValue)
	msg := binParseNoFail(t, buf.Bytes())
	assert.NotNil(t, msg)
}

func Test_BinSimpleGetCommand(t *testing.T) {
	// parse
	buf, _ := prepareBinMessage(
		&binHeader{opcode: opcodeGet, request: true},
		extras(), key("key"), noValue)
	msg := binParseNoFail(t, buf.Bytes())
	assert.NotNil(t, msg)
	assert.Equal(t, memcacheCmdGet, msg.command.code)
	assert.Equal(t, 1, len(msg.keys))
	assert.Equal(t, "key", msg.keys[0].String())

	// create event
	event := makeMessageEvent(t, msg)
	assert.Equal(t, "get", event["command"])
	assert.Equal(t, "Get", event["opcode"])
	assert.Equal(t, false, event["quiet"])

	// parse another message
	buf, _ = prepareBinMessage(
		&binHeader{opcode: opcodeGet, request: true, cas: 1234},
		extras(extra32Bit(1)), noKey, value("value"))
	msg = binParseNoFail(t, buf.Bytes())
	assert.NotNil(t, msg)
	assert.Equal(t, memcacheCmdGet, msg.command.code)
	assert.Equal(t, uint16(statusCodeNoError), msg.status)
	assert.True(t, msg.isCas)
	assert.Equal(t, uint64(1234), msg.casUnique)

	// create event
	event = makeMessageEvent(t, msg)
	assert.Equal(t, "get", event["command"])
	assert.Equal(t, "Get", event["opcode"])
	assert.Equal(t, false, event["quiet"])
	assert.Equal(t, uint64(1234), event["cas_unique"])
}

func Test_BinParseSet(t *testing.T) {
	// request
	buf, _ := prepareBinMessage(
		&binHeader{opcode: opcodeSet, request: true},
		extras(extra32Bit(0x1f2f), extra32Bit(0x11223344)),
		key("key"),
		value("value"))
	msg := binParseNoFail(t, buf.Bytes())
	assert.NotNil(t, msg)
	assert.Equal(t, memcacheCmdSet, msg.command.code)
	assert.Equal(t, "key", msg.keys[0].String())
	assert.Equal(t, uint32(0x1f2f), msg.flags)
	assert.Equal(t, uint32(0x11223344), msg.exptime)
	assert.Equal(t, uint(5), msg.bytes)
	assert.Equal(t, "value", msg.values[0].String())

	// event
	event := makeMessageEvent(t, msg)
	assert.Equal(t, "set", event["command"])
	assert.Equal(t, "Set", event["opcode"])
	assert.Equal(t, uint32(0x1f2f), event["flags"])
	assert.Equal(t, uint32(0x11223344), event["exptime"])
}

func Test_BinParsetSetCont(t *testing.T) {
	buf, _ := prepareBinMessage(
		&binHeader{opcode: opcodeSet, request: true},
		extras(extra32Bit(0x1f2f), extra32Bit(0x11223344)),
		key("key"),
		value("value"))

	p := newBinTestParser(t)
	msg := p.parseNoFail(buf.Bytes()[0:16])
	assert.Nil(t, msg)

	msg = p.parseNoFail(buf.Bytes()[16:28])
	assert.Nil(t, msg)

	msg = p.parseNoFail(buf.Bytes()[28:37])
	assert.Nil(t, msg)

	msg = p.parseNoFail(buf.Bytes()[37:])
	assert.NotNil(t, msg)

	assert.Equal(t, memcacheCmdSet, msg.command.code)
	assert.Equal(t, "key", msg.keys[0].String())
	assert.Equal(t, uint32(0x1f2f), msg.flags)
	assert.Equal(t, uint32(0x11223344), msg.exptime)
	assert.Equal(t, uint(5), msg.bytes)
	assert.Equal(t, "value", msg.values[0].String())
}

func Test_BinParseCounterMessages(t *testing.T) {
	// request
	buf, _ := prepareBinMessage(
		&binHeader{opcode: opcodeIncrement, request: true},
		extras(extra64Bit(5), extra64Bit(1), extra32Bit(0x11223344)),
		key("key"), noValue)
	msg := binParseNoFail(t, buf.Bytes())
	event := makeMessageEvent(t, msg)

	assert.NotNil(t, msg)
	assert.Equal(t, uint64(5), msg.value)
	assert.Equal(t, uint64(1), msg.value2)
	assert.Equal(t, uint32(0x11223344), msg.exptime)

	assert.Equal(t, "incr", event["command"])
	assert.Equal(t, "Increment", event["opcode"])
	assert.Equal(t, uint64(5), event["delta"])
	assert.Equal(t, uint64(1), event["initial"])
	assert.Equal(t, uint32(0x11223344), event["exptime"])

	// response
	buf, _ = prepareBinMessage(
		&binHeader{opcode: opcodeIncrement, request: false},
		extras(), noKey,
		binValue([]byte{1, 2, 3, 4, 5, 6, 7, 8}))
	msg = binParseNoFail(t, buf.Bytes())
	event = makeMessageEvent(t, msg)

	assert.NotNil(t, msg)
	assert.Equal(t, uint64(0x0102030405060708), msg.value)

	assert.Equal(t, "Success", event["status"])
	assert.Equal(t, uint64(0x0102030405060708), event["value"])
}

func Test_BinParseVersionResponse(t *testing.T) {
	buf, _ := prepareBinMessage(
		&binHeader{opcode: opcodeVersion, request: false},
		extras(), noKey, value("1.2.3"))
	msg := binParseNoFail(t, buf.Bytes())
	assert.NotNil(t, msg)
	assert.Equal(t, 0, len(msg.values))
	assert.Equal(t, "1.2.3", msg.str.String())
}

func Test_BinParseStatResponse(t *testing.T) {
	buf, _ := prepareBinMessage(
		&binHeader{opcode: opcodeStat, request: false},
		extras(), key("statKey"), value("1000"))
	msg := binParseNoFail(t, buf.Bytes())
	assert.NotNil(t, msg)
	assert.Equal(t, 1, len(msg.stats))
	assert.Equal(t, "statKey", msg.stats[0].Name.String())
	assert.Equal(t, "1000", msg.stats[0].Value.String())
}

func Test_BinParseStatInvalidResponse(t *testing.T) {
	buf, _ := prepareBinMessage(
		&binHeader{opcode: opcodeStat, request: false},
		extras(), noKey, value("abc"))
	msg, err := binTryParse(t, buf.Bytes())
	assert.Nil(t, msg)
	assert.Equal(t, errExpectedKeys, err)
}
