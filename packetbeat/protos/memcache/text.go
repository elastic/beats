package memcache

// Memcache text protocol command defitions with parsers and serializers to
// create events from parsed messages.
//
// All defined messages implement the textCommandType.
//
// Request message definitions are held in requestCommands and response message
// definitions in responseCommands

import (
	"bytes"
	"fmt"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/streambuf"
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
		raw_keys := bytes.FieldsFunc(rest, func(b rune) bool {
			return b == ' '
		})
		if len(raw_keys) == 0 {
			return ErrExpectedKeys
		}
		msg.keys = make([]memcacheString, len(raw_keys))
		for i, raw_key := range raw_keys {
			msg.keys[i] = memcacheString{raw_key}
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

var argDelta = makeValueArg("delta")
var argSleepUs = makeValueArg("sleep_us")
var argValue = makeValueArg("value")
var argVerbosity = makeValueArg("verbosity")
var argSourceClass = makeIValueArg("source_class")
var argDestClass = makeIValue2Arg("dest_class")

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
	loadCommand("get", MemcacheCmdGet),
	loadCommand("gets", MemcacheCmdGets),

	// store request types
	storeCommand("set", MemcacheCmdSet),
	storeCommand("add", MemcacheCmdAdd),
	storeCommand("replace", MemcacheCmdReplace),
	storeCommand("append", MemcacheCmdAppend),
	storeCommand("prepend", MemcacheCmdPrepend),
	casStoreCommand("cas", MemcacheCmdCas),

	// counter commands
	counterCommand("incr", MemcacheCmdIncr),
	counterCommand("decr", MemcacheCmdDecr),

	// touch
	defTextMessage("touch", MemcacheStoreMsg, MemcacheCmdTouch,
		argKey, argExpTime, argOptional(argNoReply)),

	// delete command
	deleteCommand("delete", MemcacheCmdDelete, argKey, argOptional(argNoReply)),
	deleteCommand("flush_all", MemcacheCmdFlushAll, argOptional(argExpTime)),

	// slabs command
	defSubCommand("slabs", MemcacheSlabCtrlMsg, MemcacheCmdUNKNOWN, slabsCommands),

	// lru_crawler command
	defSubCommand("lru_crawler", MemcacheLruCrawlerMsg, MemcacheCmdUNKNOWN,
		lruCrawlerCommands),

	// stats command (pretty diverse, just store raw argument list in string)
	defTextMessage("stats", MemcacheStatsMsg, MemcacheCmdStats, argsRaw),

	// others
	infoCommand("verbosity", MemcacheCmdVerbosity, argVerbosity),
	infoCommand("version", MemcacheCmdVersion),
	infoCommand("quit", MemcacheCmdQuit, argOptional(argNoReply)),
}

var slabsCommands = []textCommandType{
	defTextMessage("reassign", MemcacheSlabCtrlMsg, MemcacheCmdSlabsReassign,
		argSourceClass, argDestClass),
	defTextMessage("automove", MemcacheSlabCtrlMsg, MemcacheCmdSlabsAutomove,
		argAutomove),
}

var lruCrawlerCommands = []textCommandType{
	defTextMessage("enable", MemcacheLruCrawlerMsg, MemcacheCmdLruEnable),
	defTextMessage("disable", MemcacheLruCrawlerMsg, MemcacheCmdLruDisable),
	defTextMessage("sleep", MemcacheLruCrawlerMsg, MemcacheCmdLruSleep, argSleepUs),
	defTextMessage("tocrawl", MemcacheLruCrawlerMsg, MemcacheCmdLruToCrawl, argValue),
	defTextMessage("crawl", MemcacheLruCrawlerMsg, MemcacheCmdLruToCrawl, argsRaw),
}

var responseCommands = []textCommandType{
	// retrieval response types
	defTextDataResponse("VALUE", MemcacheLoadMsg, MemcacheResValue,
		argKey, argFlags, argBytes, argOptional(argCasUnique)),

	defTextMessage("END", MemcacheLoadMsg, MemcacheResEnd),

	// store response types
	successResp("STORED", MemcacheResStored),
	failResp("NOT_STORED", MemcacheResNotStored),
	successResp("EXISTS", MemcacheResExists),
	failResp("NOT_FOUND", MemcacheResNotFound),

	// touch response types
	successResp("TOUCHED", MemcacheResTouched),

	// delete response types
	successResp("DELETED", MemcacheResDeleted),

	successResp("OK", MemcacheResOK),

	// response error types
	failResp("ERROR", MemcacheErrError),
	failMsgResp("CLIENT_ERROR", MemcacheErrClientError),
	failMsgResp("SERVER_ERROR", MemcacheErrServerError),
	failMsgResp("BUSY", MemcacheErrBusy),
	failMsgResp("BADCLASS", MemcacheErrBadClass),
	failMsgResp("NOSPARE", MemcacheErrNoSpare),
	failMsgResp("NOTFULL", MemcacheErrNotFull),
	failMsgResp("UNSAFE", MemcacheErrUnsafe),
	failMsgResp("SAME", MemcacheErrSame),

	// stats
	defTextMessage("STAT", MemcacheStatsMsg, MemcacheResStat, argStat),

	// The version response type. Version string is storedin raw_args.
	defTextMessage("VERSION", MemcacheInfoMsg, MemcacheResVersion),
}

// non-standard message types
var counterResponse = makeTextCommand(
	"",
	MemcacheCounterMsg,
	MemcacheResCounterOp,
	parseCounterResponse,
	serializeCounterResponse)

var unknownCommand = makeTextCommand(
	"UNKNOWN",
	MemcacheUnknownType,
	MemcacheCmdUNKNOWN,
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
	is_request bool,
) func(string, commandTypeCode, commandCode, ...argDef) textCommandType {
	serialize := serializeDataResponse
	if is_request {
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

var defTextDataRequest = makeDefTextDataMessage(true)
var defTextDataResponse = makeDefTextDataMessage(false)

func loadCommand(name string, code commandCode) textCommandType {
	return defTextMessage(name, MemcacheLoadMsg, code, argMultiKeys)
}

func storeCommand(name string, code commandCode) textCommandType {
	return defTextDataRequest(name, MemcacheStoreMsg, code,
		argKey, argFlags, argExpTime, argBytes, argOptional(argNoReply),
	)
}

func deleteCommand(name string, code commandCode, args ...argDef) textCommandType {
	return defTextMessage(name, MemcacheDeleteMsg, code, args...)
}

func casStoreCommand(name string, code commandCode) textCommandType {
	return defTextDataRequest(name, MemcacheStoreMsg, code,
		argKey, argFlags, argExpTime, argBytes, argCasUnique, argOptional(argNoReply))
}

func infoCommand(name string, code commandCode, args ...argDef) textCommandType {
	return defTextMessage(name, MemcacheInfoMsg, code, args...)
}

func counterCommand(name string, code commandCode) textCommandType {
	return defTextMessage(name, MemcacheCounterMsg, code,
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
	return defTextMessage(name, MemcacheSuccessResp, code)
}

func failResp(name string, code commandCode, args ...argDef) textCommandType {
	return defTextMessage(name, MemcacheFailResp, code, args...)
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
			if parser.config.parseUnkown {
				cmd = &unknownCommand
			} else {
				return parser.failing(ErrParserUnknownCommand)
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

func makeValue2Arg(name string) argDef {
	return argDef{
		parse:     textUint64Arg(setValue2),
		serialize: serializeValue2(name),
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
		if parser.config.parseUnkown {
			cmd = &unknownCommand
		} else {
			return parser.failing(ErrParserUnknownCommand)
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
	msg.value, _ = tmp.AsciiUint(false)
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
		msg.count_values++
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

func parseTextArgs(parser *parser, args []argDef) error {
	var err error = nil
	buf := streambuf.NewFixed(parser.message.rawArgs)
	for _, arg := range args {
		debug("args rest: %s", buf.Bytes())
		err = arg.parse(parser, nil, buf)
		if err != nil {
			break
		}
	}
	return err
}

func splitCommandAndArgs(line []byte) ([]byte, []byte, error) {
	command_line := streambuf.NewFixed(line)
	command, err := parseStringArg(command_line)
	if err != nil {
		return nil, nil, err
	}
	var args []byte
	if command_line.Len() > 0 {
		command_line.Advance(1)
		args = command_line.Bytes()
	}
	return command, args, command_line.Err()
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

	var noreplyArg = []byte("noreply")
	noreply := bytes.HasPrefix(buf.Bytes(), noreplyArg)
	if !noreply {
		return false, ErrExpectedNoReply
	}
	return true, nil
}

func parseNextArg(buf *streambuf.Buffer) error {
	err := buf.IgnoreSymbol(' ')
	if err == streambuf.ErrUnexpectedEOB || err == streambuf.ErrNoMoreBytes {
		buf.SetError(nil)
		return ErrNoMoreArgument
	}
	if buf.Len() == 0 {
		return ErrNoMoreArgument
	}
	return nil
}

func textArgError(err error) error {
	if err == streambuf.ErrUnexpectedEOB {
		return ErrNoMoreArgument
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
	value, err := buf.AsciiUint(false)
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
	value, err := buf.AsciiUint(false)
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
	value, err := buf.AsciiInt(false)
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
	event_type := typ.String()
	return func(msg *message, event common.MapStr) error {
		event["command"] = command
		event["type"] = event_type
		event["count_values"] = msg.count_values
		if msg.count_values != 0 && msg.data.IsSet() {
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
	event_type := typ.String()
	return func(msg *message, event common.MapStr) error {
		event["command"] = response
		event["type"] = event_type
		event["count_values"] = msg.count_values
		if msg.count_values != 0 && len(msg.values) > 0 {
			event["values"] = msg.values
		}
		return serializeArgs(msg, event, args)
	}
}

func serializeUnknown(msg *message, event common.MapStr) error {
	event["line"] = msg.commandLine
	event["command"] = MemcacheCmdUNKNOWN.String()
	event["type"] = MemcacheUnknownType.String()
	return nil
}

func serializeCounterResponse(msg *message, event common.MapStr) error {
	event["command"] = MemcacheResCounterOp.String()
	event["type"] = MemcacheCounterMsg.String()
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
