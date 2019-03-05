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

package memcache

// Memcache binary protocol command definitions with parsers and serializers
// to create events from parsed messages.
//
// All commands implement the commandType and are create and registered in the
// init function.

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/streambuf"
)

type memcacheMagic uint8

const (
	memcacheMagicRequest  = 0x80
	memcacheMagicResponse = 0x81
)

const memcacheHeaderSize = 24

var memcacheBinaryCommandTable = make(map[memcacheOpcode]*commandType)

var binaryUnknownCommand *commandType

var extraFlags = argDef{
	parse:     binaryUint32Extra(setFlags),
	serialize: serializeFlags,
}

var extraExpTime = argDef{
	parse:     binaryUint32Extra(setExpTime),
	serialize: serializeExpTime,
}

var binStatsValue = argDef{
	parse:     argparseNoop,
	serialize: serializeStats,
}

var extraValue = makeValueExtra("value")
var extraDelta = makeValueExtra("delta")
var extraInitial = makeValue2Extra("initial")
var extraVerbosity = make32ValueExtra("verbosity")

func init() {
	// define all memcache opcode commands:
	binaryUnknownCommand = defBinaryCommand(
		nil,
		memcacheUnknownType,
		memcacheCmdUNKNOWN,
		requestArgs(),
		responseArgs(),
		nil)

	defBinaryCommand(
		opcodes(opcodeGet, opcodeGetK, opcodeGetKQ, opcodeGetQ),
		memcacheLoadMsg,
		memcacheCmdGet,
		requestArgs(),
		responseArgs(extraFlags),
		nil)

	defBinaryEmptyCommand(
		opcodes(opcodeDelete, opcodeDeleteQ),
		memcacheDeleteMsg, memcacheCmdDelete)

	defBinaryCommand(
		opcodes(opcodeFlush, opcodeFlushQ),
		memcacheDeleteMsg,
		memcacheCmdDelete,
		requestArgs(extraExpTime),
		responseArgs(), nil)

	defBinaryStoreCommand(opcodes(opcodeSet, opcodeSetQ), memcacheCmdSet)
	defBinaryStoreCommand(opcodes(opcodeAdd, opcodeAddQ), memcacheCmdAdd)
	defBinaryStoreCommand(
		opcodes(opcodeReplace, opcodeReplaceQ),
		memcacheCmdReplace)
	defBinaryEmptyCommand(
		opcodes(opcodeAppend, opcodeAppendQ),
		memcacheStoreMsg, memcacheCmdAppend)
	defBinaryEmptyCommand(
		opcodes(opcodePrepend, opcodePrependQ),
		memcacheStoreMsg, memcacheCmdPrepend)

	defBinaryEmptyCommand(opcodes(opcodeNoOp), memcacheInfoMsg, memcacheCmdNoOp)

	defBinaryCounterCommand(
		opcodes(opcodeIncrement, opcodeIncrementQ),
		memcacheCmdIncr)
	defBinaryCounterCommand(
		opcodes(opcodeDecrement, opcodeDecrementQ),
		memcacheCmdDecr)

	defBinaryEmptyCommand(
		opcodes(opcodeQuit, opcodeQuitQ),
		memcacheInfoMsg,
		memcacheCmdQuit)

	defBinaryCommand(
		opcodes(opcodeVersion),
		memcacheInfoMsg,
		memcacheCmdVersion,
		requestArgs(), responseArgs(),
		parseVersionNumber)

	defBinaryCommand(
		opcodes(opcodeTouch),
		memcacheStoreMsg, memcacheCmdTouch,
		requestArgs(extraExpTime),
		responseArgs(), nil)

	defBinaryCommand(
		opcodes(opcodeVerbosity),
		memcacheInfoMsg,
		memcacheCmdVerbosity,
		requestArgs(extraVerbosity),
		responseArgs(),
		nil)

	defBinarySaslCommand(opcodeSaslListMechs, memcacheCmdSaslList)
	defBinarySaslCommand(opcodeSaslAuth, memcacheCmdSaslAuth)
	defBinarySaslCommand(opcodeSaslStep, memcacheCmdSaslStep)

	defBinaryCommand(
		opcodes(opcodeStat),
		memcacheStatsMsg,
		memcacheCmdStats,
		requestArgs(),
		responseArgs(binStatsValue),
		parseStatResponse)
}

func opcodes(codes ...memcacheOpcode) []memcacheOpcode {
	return codes
}

func requestArgs(args ...argDef) []argDef {
	if len(args) == 0 {
		return nil
	}
	return args
}

var responseArgs = requestArgs

func isQuietOpcode(o memcacheOpcode) bool {
	switch o {
	case opcodeGetQ,
		opcodeGetKQ,
		opcodeSetQ,
		opcodeAddQ,
		opcodeReplaceQ,
		opcodeDeleteQ,
		opcodeIncrementQ,
		opcodeDecrementQ,
		opcodeQuitQ,
		opcodeFlushQ,
		opcodeAppendQ,
		opcodePrependQ,
		opcodeGatQ,
		opcodeGatKQ,
		opcodeRSetQ,
		opcodeRAppendQ,
		opcodeRPrependQ,
		opcodeRDeleteQ,
		opcodeRIncrQ,
		opcodeRDecrQ:
		return true
	}
	return false
}

func defBinaryCommand(codes []memcacheOpcode,
	typ commandTypeCode,
	code commandCode,
	requestArgs, responseArgs []argDef,
	valueParser parserStateFn,
) *commandType {
	command := &commandType{
		typ:   typ,
		code:  code,
		parse: makeParseBinary(requestArgs, responseArgs, valueParser),
		event: makeSerializeBinary(typ, code, requestArgs, responseArgs),
	}
	for _, c := range codes {
		memcacheBinaryCommandTable[c] = command
	}
	return command
}

func defBinaryEmptyCommand(
	opcodes []memcacheOpcode,
	typ commandTypeCode,
	code commandCode,
) {
	defBinaryCommand(opcodes, typ, code, nil, nil, nil)
}

func defBinarySaslCommand(opcode memcacheOpcode, code commandCode) {
	defBinaryEmptyCommand(opcodes(opcode), memcacheAuthMsg, code)
}

func defBinaryStoreCommand(opcodes []memcacheOpcode, code commandCode) {
	defBinaryCommand(opcodes, memcacheStoreMsg, code,
		requestArgs(extraFlags, extraExpTime),
		responseArgs(), nil)
}

func defBinaryCounterCommand(opcodes []memcacheOpcode, code commandCode) {
	defBinaryCommand(
		opcodes,
		memcacheCounterMsg,
		code,
		requestArgs(extraDelta, extraInitial, extraExpTime),
		responseArgs(),
		parseBinaryCounterResponse,
	)
}

func parseBinaryCommand(parser *parser, buf *streambuf.Buffer) parseResult {
	debug("on binary message")

	if !buf.Avail(memcacheHeaderSize) {
		return parser.needMore()
	}

	msg := parser.message
	msg.isBinary = true
	magic, _ := buf.ReadNetUint8At(0)
	switch magic {
	case memcacheMagicRequest:
		msg.IsRequest = true
	case memcacheMagicResponse:
		msg.IsRequest = false
	default:
		return parser.failing(errInvalidMemcacheMagic)
	}

	opcode, _ := buf.ReadNetUint8At(1)
	keyLen, err := buf.ReadNetUint16At(2)
	extraLen, _ := buf.ReadNetUint8At(4)
	if err != nil {
		return parser.failing(err)
	}

	debug("magic: %v", magic)
	debug("opcode: %v", opcode)
	debug("extra len: %v", extraLen)
	debug("key len: %v", keyLen)

	totalHeaderLen := memcacheHeaderSize + int(extraLen) + int(keyLen)
	debug("total header len: %v", totalHeaderLen)
	if !buf.Avail(totalHeaderLen) {
		return parser.needMore()
	}

	command := memcacheBinaryCommandTable[memcacheOpcode(opcode)]
	if command == nil {
		debug("unknown command")
		command = binaryUnknownCommand
	}

	msg.opcode = memcacheOpcode(opcode)
	msg.command = command
	msg.isQuiet = isQuietOpcode(msg.opcode)
	return parser.contWithShallow(buf, command.parse)
}

func makeParseBinary(
	requestArgs, responseArgs []argDef,
	valueParser parserStateFn,
) parserStateFn {
	return func(parser *parser, buf *streambuf.Buffer) parseResult {
		header := buf.Snapshot()
		buf.Advance(memcacheHeaderSize)

		msg := parser.message
		if msg.IsRequest {
			msg.vbucket, _ = header.ReadNetUint16At(6)
		} else {
			msg.status, _ = header.ReadNetUint16At(6)
		}

		cas, _ := header.ReadNetUint64At(16)
		if cas != 0 {
			setCasUnique(msg, cas)
		}
		msg.opaque, _ = header.ReadNetUint32At(12)

		// check message length

		extraLen, _ := header.ReadNetUint8At(4)
		keyLen, _ := header.ReadNetUint16At(2)
		totalLen, _ := header.ReadNetUint32At(8)
		if totalLen == 0 {
			// no extra, key or value -> publish message
			return parser.yield(buf.BufferConsumed())
		}

		valueLen := int(totalLen) - (int(extraLen) + int(keyLen))
		if valueLen < 0 {
			return parser.failing(errLen)
		}

		if valueParser != nil && valueLen > 0 {
			if !buf.Avail(int(totalLen)) {
				return parser.needMore()
			}
		}

		var extras *streambuf.Buffer
		if extraLen > 0 {
			tmp, _ := buf.Collect(int(extraLen))
			extras = streambuf.NewFixed(tmp)
			var err error
			if msg.IsRequest && requestArgs != nil {
				err = parseBinaryArgs(parser, requestArgs, header, extras)
			} else if responseArgs != nil {
				err = parseBinaryArgs(parser, responseArgs, header, extras)
			}
			if err != nil {
				msg.AddNotes(err.Error())
			}
		}

		if keyLen > 0 {
			key, _ := buf.Collect(int(keyLen))
			keys := []memcacheString{{key}}
			msg.keys = keys
		}

		if valueLen == 0 {
			return parser.yield(buf.BufferConsumed())
		}

		// call parseDataBinary
		msg.bytes = uint(valueLen)
		if valueParser == nil {
			return parser.contWith(buf, parseStateDataBinary)
		}
		return parser.contWithShallow(buf, valueParser)
	}
}

func parseBinaryArgs(
	parser *parser,
	args []argDef,
	header, buf *streambuf.Buffer,
) error {
	for _, arg := range args {
		if err := arg.parse(parser, header, buf); err != nil {
			return err
		}
	}
	return nil
}

func parseDataBinary(parser *parser, buf *streambuf.Buffer) parseResult {
	msg := parser.message
	data, err := buf.Collect(int(msg.bytes - msg.bytesLost))
	if err != nil {
		if err == streambuf.ErrNoMoreBytes {
			return parser.needMore()
		}
		return parser.failing(err)
	}

	debug("found data message")
	if msg.bytesLost > 0 {
		msg.countValues++
	} else {
		parser.appendMessageData(data)
	}
	return parser.yield(buf.BufferConsumed() + int(msg.bytesLost))
}

func parseBinaryCounterResponse(
	parser *parser,
	buf *streambuf.Buffer,
) parseResult {
	msg := parser.message
	if msg.IsRequest {
		return parser.contWith(buf, parseStateDataBinary)
	}

	// size already checked
	bytes, _ := buf.Collect(int(msg.bytes))
	tmp := streambuf.NewFixed(bytes)
	err := withBinaryUint64(parser, tmp, func(msg *message, value uint64) {
		msg.value = value
	})
	if err != nil {
		return parser.failing(err)
	}

	buf.Advance(8)
	return parser.yield(buf.BufferConsumed())
}

func parseVersionNumber(parser *parser, buf *streambuf.Buffer) parseResult {
	msg := parser.message
	if msg.IsRequest {
		return parser.contWith(buf, parseStateDataBinary)
	}

	// size already checked
	bytes, _ := buf.Collect(int(msg.bytes))
	msg.str = memcacheString{bytes}

	return parser.yield(buf.BufferConsumed())
}

func parseStatResponse(parser *parser, buf *streambuf.Buffer) parseResult {
	msg := parser.message
	if msg.IsRequest {
		return parser.contWith(buf, parseStateDataBinary)
	}

	bytes, _ := buf.Collect(int(msg.bytes))

	if len(msg.keys) == 0 {
		return parser.failing(errExpectedKeys)
	}

	msg.stats = append(msg.stats, memcacheStat{
		msg.keys[0],
		memcacheString{bytes},
	})
	return parser.yield(buf.BufferConsumed())
}

func makeSerializeBinary(
	typ commandTypeCode,
	code commandCode,
	requestArgs []argDef,
	responseArgs []argDef,
) eventFn {
	command := code.String()
	eventType := typ.String()
	return func(msg *message, event common.MapStr) error {
		event["command"] = command
		event["type"] = eventType
		event["opcode"] = msg.opcode.String()
		event["opcode_value"] = msg.opcode
		event["opaque"] = msg.opaque

		if msg.keys != nil {
			serializeKeys(msg, event)
		}
		if msg.isCas {
			serializeCas(msg, event)
		}
		if msg.countValues > 0 {
			event["count_values"] = msg.countValues
			if len(msg.values) > 0 {
				event["values"] = msg.values
			}
			event["bytes"] = msg.bytes + msg.bytesLost
		}
		if msg.str.raw != nil {
			event["version"] = &msg.str
		}

		if msg.IsRequest {
			event["quiet"] = msg.isQuiet
			event["vbucket"] = msg.vbucket
			return serializeArgs(msg, event, requestArgs)
		}

		status := memcacheStatusCode(msg.status)
		event["status"] = status.String()
		event["status_code"] = status

		if typ == memcacheCounterMsg {
			event["value"] = msg.value
		}
		return serializeArgs(msg, event, responseArgs)
	}
}

func make32ValueExtra(name string) argDef {
	return argDef{
		parse: binaryUint32Extra(func(msg *message, v uint32) {
			msg.value = uint64(v)
		}),
		serialize: serializeValue(name),
	}
}

func makeValueExtra(name string) argDef {
	return argDef{
		parse:     binaryUint64Extra(setValue),
		serialize: serializeValue(name),
	}
}

func makeValue2Extra(name string) argDef {
	return argDef{
		parse:     binaryUint64Extra(setValue2),
		serialize: serializeValue2(name),
	}
}

func binaryUint32Extra(setter func(*message, uint32)) argParser {
	return func(parser *parser, hdr, buf *streambuf.Buffer) error {
		return withBinaryUint32(parser, buf, setter)
	}
}

func binaryUint64Extra(setter func(*message, uint64)) argParser {
	return func(parser *parser, hdr, buf *streambuf.Buffer) error {
		return withBinaryUint64(parser, buf, setter)
	}
}

func withBinaryUint32(
	parser *parser,
	buf *streambuf.Buffer,
	f func(*message, uint32),
) error {
	val, err := buf.ReadNetUint32()
	if err == nil {
		f(parser.message, val)
	}
	return err
}

func withBinaryUint64(
	parser *parser,
	buf *streambuf.Buffer,
	f func(*message, uint64),
) error {
	val, err := buf.ReadNetUint64()
	if err == nil {
		f(parser.message, val)
	}
	return err
}
