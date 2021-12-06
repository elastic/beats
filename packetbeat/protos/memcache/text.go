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

// Memcache text protocol command definitions with parsers and serializers to
// create events from parsed messages.
//
// All defined messages implement the textCommandType.
//
// Request message definitions are held in requestCommands and response message
// definitions in responseCommands

import (
	"bytes"
	"fmt"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/streambuf"
)

type textCommandType struct {
	name []byte
	commandType
}

// entry point for text based protocol messages
var parseTextCommand parserStateFn

func init() {
	parseTextCommand = doParseTextCommand
}

var argKey = argDef{
	parse: func(parser *parser, hdr, buf *streambuf.Buffer) error {
		keys, err := parseKeyArg(buf)
		parser.message.keys = keys
		return err
	},
	serialize: serializeKeys,
}

var argMultiKeys = argDef{
	parse: func(parser *parser, hdr, buf *streambuf.Buffer) error {
		msg := parser.message
		rest := buf.Bytes()
		buf.Advance(len(rest))
		rawKeys := bytes.FieldsFunc(rest, func(b rune) bool {
			return b == ' '
		})
		if len(rawKeys) == 0 {
			return errExpectedKeys
		}
		msg.keys = make([]memcacheString, len(rawKeys))
		for i, rawKey := range rawKeys {
			msg.keys[i] = memcacheString{rawKey}
		}
		return nil
	},
	serialize: serializeKeys,
}

var argFlags = argDef{
	parse:     textUintArg(setFlags),
	serialize: serializeFlags,
}

var argExpTime = argDef{
	parse:     textUintArg(setExpTime),
	serialize: serializeExpTime,
}

var argBytes = argDef{
	parse:     textUintArg(setByteCount),
	serialize: serializeByteCount,
}

var argCasUnique = argDef{
	parse:     textUint64Arg(setCasUnique),
	serialize: serializeCas,
}

var argAutomove = argDef{
	parse:     textUint64Arg(setValue),
	serialize: serializeAutomove,
}

var argsRaw = argDef{
	parse:     argparseNoop,
	serialize: serializeRawArgs,
}

var argStat = argDef{
	parse:     parseStatLine,
	serialize: serializeStats,
}

var (
	argDelta       = makeValueArg("delta")
	argSleepUs     = makeValueArg("sleep_us")
	argValue       = makeValueArg("value")
	argVerbosity   = makeValueArg("verbosity")
	argSourceClass = makeIValueArg("source_class")
	argDestClass   = makeIValue2Arg("dest_class")
)

var argNoReply = argDef{
	parse: func(parser *parser, hdr, buf *streambuf.Buffer) error {
		b, err := parseNoReplyArg(buf)
		parser.message.noreply = b
		return err
	},
	serialize: func(msg *message, event common.MapStr) error {
		event["noreply"] = msg.noreply
		return nil
	},
}

var argErrorMessage = argDef{
	parse: func(parser *parser, hdr, buf *streambuf.Buffer) error {
		parser.message.errorMsg = memcacheString{buf.Bytes()}
		return nil
	},
	serialize: func(msg *message, event common.MapStr) error {
		event["error_msg"] = msg.errorMsg
		return nil
	},
}

var requestCommands = []textCommandType{
	// retrieval request types
	loadCommand("get", memcacheCmdGet),
	loadCommand("gets", memcacheCmdGets),

	// store request types
	storeCommand("set", memcacheCmdSet),
	storeCommand("add", memcacheCmdAdd),
	storeCommand("replace", memcacheCmdReplace),
	storeCommand("append", memcacheCmdAppend),
	storeCommand("prepend", memcacheCmdPrepend),
	casStoreCommand("cas", memcacheCmdCas),

	// counter commands
	counterCommand("incr", memcacheCmdIncr),
	counterCommand("decr", memcacheCmdDecr),

	// touch
	defTextMessage("touch", memcacheStoreMsg, memcacheCmdTouch,
		argKey, argExpTime, argOptional(argNoReply)),

	// delete command
	deleteCommand("delete", memcacheCmdDelete, argKey, argOptional(argNoReply)),
	deleteCommand("flush_all", memcacheCmdFlushAll, argOptional(argExpTime)),

	// slabs command
	defSubCommand("slabs", memcacheSlabCtrlMsg, memcacheCmdUNKNOWN, slabsCommands),

	// lru_crawler command
	defSubCommand("lru_crawler", memcacheLruCrawlerMsg, memcacheCmdUNKNOWN,
		lruCrawlerCommands),

	// stats command (pretty diverse, just store raw argument list in string)
	defTextMessage("stats", memcacheStatsMsg, memcacheCmdStats, argsRaw),

	// others
	infoCommand("verbosity", memcacheCmdVerbosity, argVerbosity),
	infoCommand("version", memcacheCmdVersion),
	infoCommand("quit", memcacheCmdQuit, argOptional(argNoReply)),
}

var slabsCommands = []textCommandType{
	defTextMessage("reassign", memcacheSlabCtrlMsg, memcacheCmdSlabsReassign,
		argSourceClass, argDestClass),
	defTextMessage("automove", memcacheSlabCtrlMsg, memcacheCmdSlabsAutomove,
		argAutomove),
}

var lruCrawlerCommands = []textCommandType{
	defTextMessage("enable", memcacheLruCrawlerMsg, memcacheCmdLruEnable),
	defTextMessage("disable", memcacheLruCrawlerMsg, memcacheCmdLruDisable),
	defTextMessage("sleep", memcacheLruCrawlerMsg, memcacheCmdLruSleep, argSleepUs),
	defTextMessage("tocrawl", memcacheLruCrawlerMsg, memcacheCmdLruToCrawl, argValue),
	defTextMessage("crawl", memcacheLruCrawlerMsg, memcacheCmdLruToCrawl, argsRaw),
}

var responseCommands = []textCommandType{
	// retrieval response types
	defTextDataResponse("VALUE", memcacheLoadMsg, memcacheResValue,
		argKey, argFlags, argBytes, argOptional(argCasUnique)),

	defTextMessage("END", memcacheLoadMsg, memcacheResEnd),

	// store response types
	successResp("STORED", memcacheResStored),
	failResp("NOT_STORED", memcacheResNotStored),
	successResp("EXISTS", memcacheResExists),
	failResp("NOT_FOUND", memcacheResNotFound),

	// touch response types
	successResp("TOUCHED", memcacheResTouched),

	// delete response types
	successResp("DELETED", memcacheResDeleted),

	successResp("OK", memcacheResOK),

	// response error types
	failResp("ERROR", memcacheErrError),
	failMsgResp("CLIENT_ERROR", memcacheErrClientError),
	failMsgResp("SERVER_ERROR", memcacheErrServerError),
	failMsgResp("BUSY", memcacheErrBusy),
	failMsgResp("BADCLASS", memcacheErrBadClass),
	failMsgResp("NOSPARE", memcacheErrNoSpare),
	failMsgResp("NOTFULL", memcacheErrNotFull),
	failMsgResp("UNSAFE", memcacheErrUnsafe),
	failMsgResp("SAME", memcacheErrSame),

	// stats
	defTextMessage("STAT", memcacheStatsMsg, memcacheResStat, argStat),

	// The version response type. Version string is storedin raw_args.
	defTextMessage("VERSION", memcacheInfoMsg, memcacheResVersion),
}

// non-standard message types
var counterResponse = makeTextCommand(
	"",
	memcacheCounterMsg,
	memcacheResCounterOp,
	parseCounterResponse,
	serializeCounterResponse)

var unknownCommand = makeTextCommand(
	"UNKNOWN",
	memcacheUnknownType,
	memcacheCmdUNKNOWN,
	parseUnknown,
	serializeUnknown)

func makeTextCommand(
	name string,
	typ commandTypeCode,
	code commandCode,
	parse parserStateFn,
	event eventFn,
) textCommandType {
	return textCommandType{
		[]byte(name),
		commandType{
			typ:   typ,
			code:  code,
			parse: parse,
			event: event,
		},
	}
}

func defTextMessage(
	name string,
	typ commandTypeCode,
	code commandCode,
	args ...argDef,
) textCommandType {
	return makeTextCommand(name, typ, code,
		makeMessageParser(args),
		serializeRequest(typ, code, args...))
}

func makeDefTextDataMessage(
	isRequest bool,
) func(string, commandTypeCode, commandCode, ...argDef) textCommandType {
	serialize := serializeDataResponse
	if isRequest {
		serialize = serializeDataRequest
	}
	return func(
		name string,
		typ commandTypeCode,
		code commandCode,
		args ...argDef,
	) textCommandType {
		return makeTextCommand(name, typ, code,
			makeDataMessageParser(args),
			serialize(typ, code, args...))
	}
}

var (
	defTextDataRequest  = makeDefTextDataMessage(true)
	defTextDataResponse = makeDefTextDataMessage(false)
)

func loadCommand(name string, code commandCode) textCommandType {
	return defTextMessage(name, memcacheLoadMsg, code, argMultiKeys)
}

func storeCommand(name string, code commandCode) textCommandType {
	return defTextDataRequest(name, memcacheStoreMsg, code,
		argKey, argFlags, argExpTime, argBytes, argOptional(argNoReply),
	)
}

func deleteCommand(name string, code commandCode, args ...argDef) textCommandType {
	return defTextMessage(name, memcacheDeleteMsg, code, args...)
}

func casStoreCommand(name string, code commandCode) textCommandType {
	return defTextDataRequest(name, memcacheStoreMsg, code,
		argKey, argFlags, argExpTime, argBytes, argCasUnique, argOptional(argNoReply))
}

func infoCommand(name string, code commandCode, args ...argDef) textCommandType {
	return defTextMessage(name, memcacheInfoMsg, code, args...)
}

func counterCommand(name string, code commandCode) textCommandType {
	return defTextMessage(name, memcacheCounterMsg, code,
		argKey, argDelta, argOptional(argNoReply))
}

func defSubCommand(
	name string,
	typ commandTypeCode,
	code commandCode,
	commands []textCommandType,
) textCommandType {
	return makeTextCommand(name, typ, code,
		makeSubMessageParser(commands), serializeNop)
}

func successResp(name string, code commandCode) textCommandType {
	return defTextMessage(name, memcacheSuccessResp, code)
}

func failResp(name string, code commandCode, args ...argDef) textCommandType {
	return defTextMessage(name, memcacheFailResp, code, args...)
}

func failMsgResp(name string, code commandCode) textCommandType {
	return failResp(name, code, argErrorMessage)
}

func makeDataMessageParser(args []argDef) parserStateFn {
	return func(parser *parser, buf *streambuf.Buffer) parseResult {
		if err := parseTextArgs(parser, args); err != nil {
			return parser.failing(err)
		}
		return parser.contWith(buf, parseStateData)
	}
}

// Creates command message parser parsing the arguments defined in argDef.
// without any binary data in protocol. The parser generated works on already
// separated command.
func makeMessageParser(args []argDef) parserStateFn {
	return func(parser *parser, buf *streambuf.Buffer) parseResult {
		if err := parseTextArgs(parser, args); err != nil {
			return parser.failing(err)
		}
		return parser.yieldNoData(buf)
	}
}

func makeSubMessageParser(commands []textCommandType) parserStateFn {
	return func(parser *parser, buf *streambuf.Buffer) parseResult {
		msg := parser.message
		sub, args, err := splitCommandAndArgs(msg.rawArgs)
		if err != nil {
			return parser.failing(err)
		}

		debug("handle subcommand: %s", sub)
		cmd := findTextCommandType(commands, sub)
		if cmd == nil {
			debug("unknown sub-command: %s", sub)
			if parser.config.parseUnknown {
				cmd = &unknownCommand
			} else {
				return parser.failing(errParserUnknownCommand)
			}
		}

		msg.command = &cmd.commandType
		msg.rawArgs = args
		return parser.contWithShallow(buf, cmd.parse)
	}
}

func makeValueArg(name string) argDef {
	return argDef{
		parse:     textUint64Arg(setValue),
		serialize: serializeValue(name),
	}
}

func makeIValueArg(name string) argDef {
	return argDef{
		parse: func(parser *parser, hdr, buf *streambuf.Buffer) error {
			return withInt64Arg(parser, buf, func(msg *message, v int64) {
				msg.ivalue = v
			})
		},
		serialize: func(msg *message, event common.MapStr) error {
			event[name] = msg.ivalue
			return nil
		},
	}
}

func makeIValue2Arg(name string) argDef {
	return argDef{
		parse: func(parser *parser, hdr, buf *streambuf.Buffer) error {
			return withInt64Arg(parser, buf, func(msg *message, v int64) {
				msg.ivalue2 = v
			})
		},
		serialize: func(msg *message, event common.MapStr) error {
			event[name] = msg.ivalue2
			return nil
		},
	}
}

func doParseTextCommand(parser *parser, buf *streambuf.Buffer) parseResult {
	line, err := buf.UntilCRLF()
	if err != nil {
		if err == streambuf.ErrNoMoreBytes {
			return parser.needMore()
		}
		return parser.failing(err)
	}

	msg := parser.message
	command, args, err := splitCommandAndArgs(line)
	if err != nil {
		return parser.failing(err)
	}

	debug("parse command: '%s' '%s'", command, args)

	msg.IsRequest = 'a' <= command[0] && command[0] <= 'z'
	var cmd *textCommandType
	if msg.IsRequest {
		cmd = findTextCommandType(requestCommands, command)
	} else {
		cmd = findTextCommandType(responseCommands, command)
		if cmd == nil {
			b := command[0]
			if '0' <= b && b <= '9' {
				cmd = &counterResponse
			}
		}
	}
	if cmd == nil {
		debug("unknown command: %s", msg.command)
		if parser.config.parseUnknown {
			cmd = &unknownCommand
		} else {
			return parser.failing(errParserUnknownCommand)
		}
	}

	msg.command = &cmd.commandType
	msg.rawArgs = args
	msg.commandLine = memcacheString{line}
	msg.rawCommand = command

	// the command parser will work on already separated command line.
	// The parser will either yield a message directly, or switch to binary
	// data parsing mode, which is provided by explicit state
	return parser.contWithShallow(buf, cmd.parse)
}

func parseUnknown(parser *parser, buf *streambuf.Buffer) parseResult {
	return parser.yieldNoData(buf)
}

func parseCounterResponse(parser *parser, buf *streambuf.Buffer) parseResult {
	msg := parser.message
	tmp := streambuf.NewFixed(msg.rawCommand)
	msg.value, _ = tmp.UintASCII(false)
	if tmp.Failed() {
		err := tmp.Err()
		debug("counter response invalid: %v", err)
		return parser.failing(err)
	}
	debug("parsed counter response: %v", msg.value)
	return parser.yieldNoData(buf)
}

func parseData(parser *parser, buf *streambuf.Buffer) parseResult {
	msg := parser.message
	debug("parse message data (%v)", msg.bytes)
	data, err := buf.CollectWithSuffix(
		int(msg.bytes-msg.bytesLost),
		[]byte("\r\n"),
	)
	if err != nil {
		if err == streambuf.ErrNoMoreBytes {
			return parser.needMore()
		}
		return parser.failing(err)
	}

	debug("found message data")
	if msg.bytesLost > 0 {
		msg.countValues++
	} else {
		parser.appendMessageData(data)
	}
	return parser.yield(buf.BufferConsumed() + int(msg.bytesLost))
}

func parseStatLine(parser *parser, hdr, buf *streambuf.Buffer) error {
	name, _ := parseStringArg(buf)
	value, _ := parseStringArg(buf)
	if buf.Failed() {
		return buf.Err()
	}

	msg := parser.message
	msg.stats = append(msg.stats, memcacheStat{
		memcacheString{name},
		memcacheString{value},
	})
	return nil
}

func parseTextArgs(parser *parser, args []argDef) (err error) {
	buf := streambuf.NewFixed(parser.message.rawArgs)
	for _, arg := range args {
		debug("args rest: %s", buf.Bytes())
		err = arg.parse(parser, nil, buf)
		if err != nil {
			break
		}
	}
	return
}

func splitCommandAndArgs(line []byte) ([]byte, []byte, error) {
	commandLine := streambuf.NewFixed(line)
	command, err := parseStringArg(commandLine)
	if err != nil {
		return nil, nil, err
	}
	var args []byte
	if commandLine.Len() > 0 {
		commandLine.Advance(1)
		args = commandLine.Bytes()
	}
	return command, args, commandLine.Err()
}

func parseStringArg(buf *streambuf.Buffer) ([]byte, error) {
	if err := parseNextArg(buf); err != nil {
		return nil, err
	}
	return buf.UntilSymbol(' ', false)
}

func parseKeyArg(buf *streambuf.Buffer) ([]memcacheString, error) {
	str, err := parseStringArg(buf)
	if err != nil {
		return nil, err
	}
	return []memcacheString{{str}}, nil
}

func parseNoReplyArg(buf *streambuf.Buffer) (bool, error) {
	debug("parse noreply")

	err := parseNextArg(buf)
	if err != nil {
		return false, textArgError(err)
	}

	noreplyArg := []byte("noreply")
	noreply := bytes.HasPrefix(buf.Bytes(), noreplyArg)
	if !noreply {
		return false, errExpectedNoReply
	}
	return true, nil
}

func parseNextArg(buf *streambuf.Buffer) error {
	err := buf.IgnoreSymbol(' ')
	if err == streambuf.ErrUnexpectedEOB || err == streambuf.ErrNoMoreBytes {
		buf.SetError(nil)
		return errNoMoreArgument
	}
	if buf.Len() == 0 {
		return errNoMoreArgument
	}
	return nil
}

func textArgError(err error) error {
	if err == streambuf.ErrUnexpectedEOB {
		return errNoMoreArgument
	}
	return err
}

func withUintArg(
	parser *parser,
	buf *streambuf.Buffer,
	fn func(msg *message, v uint32),
) error {
	msg := parser.message
	parseNextArg(buf)
	value, err := buf.UintASCII(false)
	if err == nil {
		fn(msg, uint32(value))
	}
	return textArgError(err)
}

func withUint64Arg(
	parser *parser,
	buf *streambuf.Buffer,
	fn func(msg *message, v uint64),
) error {
	parseNextArg(buf)
	value, err := buf.UintASCII(false)
	if err == nil {
		fn(parser.message, value)
	}
	return textArgError(err)
}

func textUintArg(setter func(*message, uint32)) argParser {
	return func(parser *parser, hdr, buf *streambuf.Buffer) error {
		return withUintArg(parser, buf, setter)
	}
}

func textUint64Arg(setter func(*message, uint64)) argParser {
	return func(parser *parser, hdr, buf *streambuf.Buffer) error {
		return withUint64Arg(parser, buf, setter)
	}
}

func withInt64Arg(
	parser *parser,
	buf *streambuf.Buffer,
	fn func(msg *message, v int64),
) error {
	parseNextArg(buf)
	value, err := buf.IntASCII(false)
	if err == nil {
		fn(parser.message, value)
	}
	return textArgError(err)
}

func findTextCommandType(commands []textCommandType, name []byte) *textCommandType {
	for _, cmd := range commands {
		if bytes.Equal(name, cmd.name) {
			return &cmd
		}
	}
	return nil
}

func serializeRequest(
	typ commandTypeCode,
	code commandCode,
	args ...argDef,
) eventFn {
	command := code.String()
	eventType := typ.String()
	return func(msg *message, event common.MapStr) error {
		event["command"] = command
		event["type"] = eventType
		return serializeArgs(msg, event, args)
	}
}

func serializeDataRequest(
	typ commandTypeCode,
	code commandCode,
	args ...argDef,
) eventFn {
	command := code.String()
	eventType := typ.String()
	return func(msg *message, event common.MapStr) error {
		event["command"] = command
		event["type"] = eventType
		event["count_values"] = msg.countValues
		if msg.countValues != 0 && msg.data.IsSet() {
			event["values"] = msg.data
		}
		return serializeArgs(msg, event, args)
	}
}

func serializeDataResponse(
	typ commandTypeCode,
	code commandCode,
	args ...argDef,
) eventFn {
	response := code.String()
	eventType := typ.String()
	return func(msg *message, event common.MapStr) error {
		event["command"] = response
		event["type"] = eventType
		event["count_values"] = msg.countValues
		if msg.countValues != 0 && len(msg.values) > 0 {
			event["values"] = msg.values
		}
		return serializeArgs(msg, event, args)
	}
}

func serializeUnknown(msg *message, event common.MapStr) error {
	event["line"] = msg.commandLine
	event["command"] = memcacheCmdUNKNOWN.String()
	event["type"] = memcacheUnknownType.String()
	return nil
}

func serializeCounterResponse(msg *message, event common.MapStr) error {
	event["command"] = memcacheResCounterOp.String()
	event["type"] = memcacheCounterMsg.String()
	event["value"] = msg.value
	return nil
}

func serializeRawArgs(msg *message, event common.MapStr) error {
	event["raw_args"] = memcacheString{msg.rawArgs}
	return nil
}

func serializeAutomove(msg *message, event common.MapStr) error {
	var s string
	switch msg.value {
	case 0:
		s = "standby"
	case 1:
		s = "slow"
	case 2:
		s = "aggressive"
	default:
		s = fmt.Sprint(msg.value)
	}
	event["automove"] = s
	return nil
}
