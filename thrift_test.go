package main

import (
	"encoding/hex"
	"testing"
)

func TestThrift_thriftReadString(t *testing.T) {

	if testing.Verbose() {
		LogInit(LOG_DEBUG, "", false, []string{"thrift", "thriftdetailed"})
	}

	var data []byte
	var ok, complete bool
	var off int
	var str string

	var thrift Thrift
	thrift.InitDefaults()

	data, _ = hex.DecodeString("0000000470696e67")
	str, ok, complete, off = thrift.readString(data)
	if str != "ping" || !ok || !complete || off != 8 {
		t.Error("Bad result: %s %s %s %s", str, ok, complete, off)
	}

	data, _ = hex.DecodeString("0000000470696e670000")
	str, ok, complete, off = thrift.readString(data)
	if str != "ping" || !ok || !complete || off != 8 {
		t.Error("Bad result: %s %s %s %s", str, ok, complete, off)
	}

	data, _ = hex.DecodeString("0000000470696e")
	str, ok, complete, off = thrift.readString(data)
	if str != "" || !ok || complete || off != 0 {
		t.Error("Bad result: %s %s %s %s", str, ok, complete, off)
	}

	data, _ = hex.DecodeString("000000")
	str, ok, complete, off = thrift.readString(data)
	if str != "" || !ok || complete || off != 0 {
		t.Error("Bad result: %s %s %s %s", str, ok, complete, off)
	}
}

func TestThrift_readMessageBegin(t *testing.T) {

	if testing.Verbose() {
		LogInit(LOG_DEBUG, "", false, []string{"thrift", "thriftdetailed"})
	}

	var data []byte
	var ok, complete bool
	var stream ThriftStream
	var m *ThriftMessage

	var thrift Thrift
	thrift.InitDefaults()

	data, _ = hex.DecodeString("800100010000000470696e670000000000")
	stream = ThriftStream{tcpStream: nil, data: data, message: new(ThriftMessage)}
	m = stream.message
	ok, complete = thrift.readMessageBegin(&stream)
	if !ok || !complete {
		t.Error("Bad result: %s %s", ok, complete)
	}
	if m.Method != "ping" || m.Type != ThriftMsgTypeCall ||
		m.SeqId != 0 || m.Version != ThriftVersion1 {
		t.Error("Bad values: %s %s %s %s", m.Method, m.Type, m.SeqId, m.Version)
	}

	data, _ = hex.DecodeString("800100010000000470696e6700000000")
	stream = ThriftStream{tcpStream: nil, data: data, message: new(ThriftMessage)}
	m = stream.message
	ok, complete = thrift.readMessageBegin(&stream)
	if !ok || !complete {
		t.Error("Bad result: %s %s", ok, complete)
	}
	if m.Method != "ping" || m.Type != ThriftMsgTypeCall ||
		m.SeqId != 0 || m.Version != ThriftVersion1 {
		t.Error("Bad values: %s %s %s %s", m.Method, m.Type, m.SeqId, m.Version)
	}

	data, _ = hex.DecodeString("800100010000000470696e6700000001")
	stream = ThriftStream{tcpStream: nil, data: data, message: new(ThriftMessage)}
	m = stream.message
	ok, complete = thrift.readMessageBegin(&stream)
	if !ok || !complete {
		t.Error("Bad result: %s %s", ok, complete)
	}
	if m.Method != "ping" || m.Type != ThriftMsgTypeCall ||
		m.SeqId != 1 || m.Version != ThriftVersion1 {
		t.Error("Bad values: %s %s %s %s", m.Method, m.Type, m.SeqId, m.Version)
	}

	data, _ = hex.DecodeString("800100010000000570696e6700000001")
	stream = ThriftStream{tcpStream: nil, data: data, message: new(ThriftMessage)}
	m = stream.message
	ok, complete = thrift.readMessageBegin(&stream)
	if !ok || complete {
		t.Error("Bad result: %s %s", ok, complete)
	}

	data, _ = hex.DecodeString("800100010000000570696e6700000001")
	stream = ThriftStream{tcpStream: nil, data: data, message: new(ThriftMessage)}
	m = stream.message
	ok, complete = thrift.readMessageBegin(&stream)
	if !ok || complete {
		t.Error("Bad result: %s %s", ok, complete)
	}

	data, _ = hex.DecodeString("0000000470696e670100000000")
	stream = ThriftStream{tcpStream: nil, data: data, message: new(ThriftMessage)}
	m = stream.message
	ok, complete = thrift.readMessageBegin(&stream)
	if !ok || !complete {
		t.Error("Bad result: %s %s", ok, complete)
	}
	if m.Method != "ping" || m.Type != ThriftMsgTypeCall ||
		m.SeqId != 0 || m.Version != 0 {
		t.Error("Bad values: %s %s %s %s", m.Method, m.Type, m.SeqId, m.Version)
	}

	data, _ = hex.DecodeString("0000000570696e670100000000")
	stream = ThriftStream{tcpStream: nil, data: data, message: new(ThriftMessage)}
	m = stream.message
	ok, complete = thrift.readMessageBegin(&stream)
	if !ok || complete {
		t.Error("Bad result:", ok, complete)
	}

}

func TestThrift_thriftReadField(t *testing.T) {

	if testing.Verbose() {
		LogInit(LOG_DEBUG, "", false, []string{"thrift", "thriftdetailed"})
	}

	var data []byte
	var ok, complete bool
	var stream ThriftStream
	var field *ThriftField
	var _old int

	var thrift Thrift
	thrift.InitDefaults()

	data, _ = hex.DecodeString("08000100000001")
	stream = ThriftStream{tcpStream: nil, data: data, message: new(ThriftMessage)}
	ok, complete, field = thrift.readField(&stream)
	if !ok || complete || field == nil {
		t.Error("Bad result:", ok, complete, field)
	} else {
		if field.Id != 1 || field.Type != ThriftTypeI32 || field.Value != "1" ||
			stream.parseOffset != len(stream.data) {
			t.Error("Bad values:", field.Id, field.Type, field.Value)
		}
	}

	data, _ = hex.DecodeString("0600010001")
	stream = ThriftStream{tcpStream: nil, data: data, message: new(ThriftMessage)}
	ok, complete, field = thrift.readField(&stream)
	if !ok || complete || field == nil {
		t.Error("Bad result:", ok, complete, field)
	} else {
		if field.Id != 1 || field.Type != ThriftTypeI16 || field.Value != "1" ||
			stream.parseOffset != len(stream.data) {
			t.Error("Bad values:", field.Id, field.Type, field.Value)
		}
	}

	data, _ = hex.DecodeString("0a00010000000000000001")
	stream = ThriftStream{tcpStream: nil, data: data, message: new(ThriftMessage)}
	ok, complete, field = thrift.readField(&stream)
	if !ok || complete || field == nil {
		t.Error("Bad result:", ok, complete, field)
	} else {
		if field.Id != 1 || field.Type != ThriftTypeI64 || field.Value != "1" ||
			stream.parseOffset != len(stream.data) {
			t.Error("Bad values:", field.Id, field.Type, field.Value)
		}
	}

	data, _ = hex.DecodeString("0400013ff3333333333333")
	stream = ThriftStream{tcpStream: nil, data: data, message: new(ThriftMessage)}
	ok, complete, field = thrift.readField(&stream)
	if !ok || complete || field == nil {
		t.Error("Bad result:", ok, complete, field)
	} else {
		if field.Id != 1 || field.Type != ThriftTypeDouble || field.Value != "1.2" ||
			stream.parseOffset != len(stream.data) {
			t.Error("Bad values:", field.Id, field.Type, field.Value, stream.parseOffset)
		}
	}

	data, _ = hex.DecodeString("02000101")
	stream = ThriftStream{tcpStream: nil, data: data, message: new(ThriftMessage)}
	ok, complete, field = thrift.readField(&stream)
	if !ok || complete || field == nil {
		t.Error("Bad result:", ok, complete, field)
	} else {
		if field.Id != 1 || field.Type != ThriftTypeBool || field.Value != "true" ||
			stream.parseOffset != len(stream.data) {
			t.Error("Bad values:", field.Id, field.Type, field.Value)
		}
	}

	data, _ = hex.DecodeString("0b00010000000568656c6c") // incomplete string
	stream = ThriftStream{tcpStream: nil, data: data, message: new(ThriftMessage)}
	ok, complete, field = thrift.readField(&stream)
	if !ok || complete || field != nil {
		t.Error("Bad result:", ok, complete, field)
	}

	data, _ = hex.DecodeString("0b00010000000568656c6c6f")
	stream = ThriftStream{tcpStream: nil, data: data, message: new(ThriftMessage)}
	ok, complete, field = thrift.readField(&stream)
	if !ok || complete || field == nil {
		t.Error("Bad result:", ok, complete, field)
	} else {
		if field.Id != 1 || field.Type != ThriftTypeString || field.Value != `"hello"` {
			t.Error("Bad values:", field.Id, field.Type, field.Value)
		}
	}

	_old, thrift.StringMaxSize = thrift.StringMaxSize, 3
	data, _ = hex.DecodeString("0b00010000000568656c6c6f")
	stream = ThriftStream{tcpStream: nil, data: data, message: new(ThriftMessage)}
	ok, complete, field = thrift.readField(&stream)
	if !ok || complete || field == nil {
		t.Error("Bad result:", ok, complete, field)
	} else {
		if field.Id != 1 || field.Type != ThriftTypeString || field.Value != `"hel..."` ||
			stream.parseOffset != len(stream.data) {
			t.Error("Bad values:", field.Id, field.Type, field.Value)
		}
	}
	thrift.StringMaxSize = _old

	data, _ = hex.DecodeString("0f00010600000003000100020003")
	stream = ThriftStream{tcpStream: nil, data: data, message: new(ThriftMessage)}
	ok, complete, field = thrift.readField(&stream)
	if !ok || complete || field == nil {
		t.Error("Bad result:", ok, complete, field)
	} else {
		if field.Id != 1 || field.Type != ThriftTypeList ||
			field.Value != "[1, 2, 3]" {
			t.Error("Bad values:", field.Id, field.Type, field.Value)
		}
	}

	_old, thrift.CollectionMaxSize = thrift.CollectionMaxSize, 1
	data, _ = hex.DecodeString("0f00010600000003000100020003")
	stream = ThriftStream{tcpStream: nil, data: data, message: new(ThriftMessage)}
	ok, complete, field = thrift.readField(&stream)
	if !ok || complete || field == nil {
		t.Error("Bad result:", ok, complete, field)
	} else {
		if field.Id != 1 || field.Type != ThriftTypeList ||
			stream.parseOffset != len(stream.data) ||
			field.Value != "[1, ...]" {
			t.Error("Bad values:", field.Id, field.Type, field.Value)
		}
	}
	thrift.CollectionMaxSize = _old

	data, _ = hex.DecodeString("0e0001060000000300010002000300")
	stream = ThriftStream{tcpStream: nil, data: data, message: new(ThriftMessage)}
	ok, complete, field = thrift.readField(&stream)
	if !ok || complete || field == nil {
		t.Error("Bad result:", ok, complete, field)
	} else {
		if field.Id != 1 || field.Type != ThriftTypeSet ||
			field.Value != "{1, 2, 3}" {
			t.Error("Bad values:", field.Id, field.Type, field.Value)
		}
	}

	_old, thrift.CollectionMaxSize = thrift.CollectionMaxSize, 2
	data, _ = hex.DecodeString("0e00010600000003000100020003")
	stream = ThriftStream{tcpStream: nil, data: data, message: new(ThriftMessage)}
	ok, complete, field = thrift.readField(&stream)
	if !ok || complete || field == nil {
		t.Error("Bad result:", ok, complete, field)
	} else {
		if field.Id != 1 || field.Type != ThriftTypeSet ||
			stream.parseOffset != len(stream.data) ||
			field.Value != "{1, 2, ...}" {
			t.Error("Bad values:", field.Id, field.Type, field.Value)
		}
	}
	thrift.CollectionMaxSize = _old

	data, _ = hex.DecodeString("0d00010b0600000003000000016100010000000163000300000001620002")
	stream = ThriftStream{tcpStream: nil, data: data, message: new(ThriftMessage)}
	ok, complete, field = thrift.readField(&stream)
	if !ok || complete || field == nil {
		t.Error("Bad result:", ok, complete, field)
	} else {
		if field.Id != 1 || field.Type != ThriftTypeMap ||
			field.Value != `{"a": 1, "c": 3, "b": 2}` ||
			stream.parseOffset != len(stream.data) {
			t.Error("Bad values:", field.Id, field.Type, field.Value)
		}
	}

	_old, thrift.CollectionMaxSize = thrift.CollectionMaxSize, 2
	data, _ = hex.DecodeString("0d00010b060000000300000001610001000000016300030000000162000200")
	stream = ThriftStream{tcpStream: nil, data: data, message: new(ThriftMessage)}
	ok, complete, field = thrift.readField(&stream)
	if !ok || complete || field == nil {
		t.Error("Bad result:", ok, complete, field)
	} else {
		if field.Id != 1 || field.Type != ThriftTypeMap ||
			field.Value != `{"a": 1, "c": 3, ...}` {
			t.Error("Bad values:", field.Id, field.Type, field.Value)
		}
	}
	thrift.CollectionMaxSize = _old

	data, _ = hex.DecodeString("0b00010000000568106c6c6f")
	stream = ThriftStream{tcpStream: nil, data: data, message: new(ThriftMessage)}
	ok, complete, field = thrift.readField(&stream)
	if !ok || complete || field == nil {
		t.Error("Bad result:", ok, complete, field)
	} else {
		if field.Id != 1 || field.Type != ThriftTypeString || field.Value != `"h\x10llo"` {
			t.Error("Bad values:", field.Id, field.Type, field.Value)
		}
	}

	data, _ = hex.DecodeString("0c000108000100000001080002000000000800030000000400")
	stream = ThriftStream{tcpStream: nil, data: data, message: new(ThriftMessage)}
	ok, complete, field = thrift.readField(&stream)
	if !ok || complete || field == nil {
		t.Error("Bad result:", ok, complete, field)
	} else {
		if field.Id != 1 || field.Type != ThriftTypeStruct ||
			field.Value != `(1: 1, 2: 0, 3: 4)` {
			t.Error("Bad values:", field.Id, field.Type, field.Value)
		}
	}

	_old, thrift.CollectionMaxSize = thrift.CollectionMaxSize, 2
	data, _ = hex.DecodeString("0c000108000100000001080002000000000800030000000400")
	stream = ThriftStream{tcpStream: nil, data: data, message: new(ThriftMessage)}
	ok, complete, field = thrift.readField(&stream)
	if !ok || complete || field == nil {
		t.Error("Bad result:", ok, complete, field)
	} else {
		if field.Id != 1 || field.Type != ThriftTypeStruct ||
			field.Value != `(1: 1, 2: 0, ...)` {
			t.Error("Bad values:", field.Id, field.Type, field.Value)
		}
	}
	thrift.CollectionMaxSize = _old

	data, _ = hex.DecodeString("0c0001080001000000010b00020000000568656c6c6f00")
	stream = ThriftStream{tcpStream: nil, data: data, message: new(ThriftMessage)}
	ok, complete, field = thrift.readField(&stream)
	if !ok || complete || field == nil {
		t.Error("Bad result:", ok, complete, field)
	} else {
		if field.Id != 1 || field.Type != ThriftTypeStruct ||
			field.Value != `(1: 1, 2: "hello")` {
			t.Error("Bad values:", field.Id, field.Type, field.Value)
		}
	}

	data, _ = hex.DecodeString("0c0001080001000000010c0002080001000000010b00020000000568656c6c6f0000")
	stream = ThriftStream{tcpStream: nil, data: data, message: new(ThriftMessage)}
	ok, complete, field = thrift.readField(&stream)
	if !ok || complete || field == nil {
		t.Error("Bad result:", ok, complete, field)
	} else {
		if field.Id != 1 || field.Type != ThriftTypeStruct ||
			field.Value != `(1: 1, 2: (1: 1, 2: "hello"))` {
			t.Error("Bad values:", field.Id, field.Type, field.Value)
		}
	}

	data, _ = hex.DecodeString("0c0001080001000000010e0002060000000300010002000300")
	stream = ThriftStream{tcpStream: nil, data: data, message: new(ThriftMessage)}
	ok, complete, field = thrift.readField(&stream)
	if !ok || complete || field == nil {
		t.Error("Bad result:", ok, complete, field)
	} else {
		if field.Id != 1 || field.Type != ThriftTypeStruct ||
			field.Value != `(1: 1, 2: {1, 2, 3})` {
			t.Error("Bad values:", field.Id, field.Type, field.Value)
		}
	}

}

func TestThrift_thriftMessageParser(t *testing.T) {

	if testing.Verbose() {
		LogInit(LOG_DEBUG, "", false, []string{"thrift", "thriftdetailed"})
	}

	var data []byte
	var stream ThriftStream
	var ok, complete bool
	var m *ThriftMessage

	var thrift Thrift
	thrift.InitDefaults()

	data, _ = hex.DecodeString("800100010000000470696e670000000000")
	stream = ThriftStream{tcpStream: nil, data: data, message: new(ThriftMessage)}
	ok, complete = thrift.messageParser(&stream)
	m = stream.message
	if !ok || !complete {
		t.Error("Bad result:", ok, complete)
	}
	if !m.IsRequest || m.Method != "ping" ||
		m.SeqId != 0 || m.Type != ThriftMsgTypeCall || m.Params != "()" {
		t.Error("Bad result:", stream.message)
	}

	data, _ = hex.DecodeString("800100010000000561646431360000000006000100010" +
		"60002000100")
	stream = ThriftStream{tcpStream: nil, data: data, message: new(ThriftMessage)}
	ok, complete = thrift.messageParser(&stream)
	m = stream.message
	if !ok || !complete {
		t.Error("Bad result:", ok, complete)
	}
	if !m.IsRequest || m.Method != "add16" ||
		m.SeqId != 0 || m.Type != ThriftMsgTypeCall ||
		m.Params != "(1: 1, 2: 1)" {
		t.Error("Bad result:", stream.message)
	}

	data, _ = hex.DecodeString("800100010000000963616c63756c617465000000000" +
		"80001000000010c00020800010000000108000200000000080003000000040000")
	stream = ThriftStream{tcpStream: nil, data: data, message: new(ThriftMessage)}
	ok, complete = thrift.messageParser(&stream)
	m = stream.message
	if !ok || !complete {
		t.Error("Bad result:", ok, complete)
	}
	if !m.IsRequest || m.Method != "calculate" ||
		m.SeqId != 0 || m.Type != ThriftMsgTypeCall ||
		m.Params != "(1: 1, 2: (1: 1, 2: 0, 3: 4))" {
		t.Error("Bad result:", stream.message)
	}

	data, _ = hex.DecodeString("8001000200000005616464313600000000060000000200")
	stream = ThriftStream{tcpStream: nil, data: data, message: new(ThriftMessage)}
	ok, complete = thrift.messageParser(&stream)
	m = stream.message
	if !ok || !complete {
		t.Error("Bad result:", ok, complete)
	}
	if m.IsRequest || m.Method != "add16" ||
		m.SeqId != 0 || m.Type != ThriftMsgTypeReply ||
		m.Result != "(0: 2)" {
		t.Error("Bad result:", stream.message)
	}

	data, _ = hex.DecodeString("800100020000000b6563686f5f737472696e67000000000b00" +
		"000000000568656c6c6f00")
	stream = ThriftStream{tcpStream: nil, data: data, message: new(ThriftMessage)}
	ok, complete = thrift.messageParser(&stream)
	m = stream.message
	if !ok || !complete {
		t.Error("Bad result:", ok, complete)
	}
	if m.IsRequest || m.Method != "echo_string" ||
		m.SeqId != 0 || m.Type != ThriftMsgTypeReply ||
		m.Result != `(0: "hello")` {
		t.Error("Bad result:", stream.message)
	}

	data, _ = hex.DecodeString("80010002000000096563686f5f6c697374000000000f0000060" +
		"000000300010002000300")
	stream = ThriftStream{tcpStream: nil, data: data, message: new(ThriftMessage)}
	ok, complete = thrift.messageParser(&stream)
	m = stream.message
	if !ok || !complete {
		t.Error("Bad result:", ok, complete)
	}
	if m.IsRequest || m.Method != "echo_list" ||
		m.SeqId != 0 || m.Type != ThriftMsgTypeReply ||
		m.Result != `(0: [1, 2, 3])` {
		t.Error("Bad result:", stream.message)
	}

	data, _ = hex.DecodeString("80010002000000086563686f5f6d6170000000000d00000b06000" +
		"0000300000001610001000000016300030000000162000200")
	stream = ThriftStream{tcpStream: nil, data: data, message: new(ThriftMessage)}
	ok, complete = thrift.messageParser(&stream)
	m = stream.message
	if !ok || !complete {
		t.Error("Bad result:", ok, complete)
	}
	if m.IsRequest || m.Method != "echo_map" ||
		m.SeqId != 0 || m.Type != ThriftMsgTypeReply ||
		m.Result != `(0: {"a": 1, "c": 3, "b": 2})` {
		t.Error("Bad result:", stream.message)
	}

	data, _ = hex.DecodeString("800100020000000963616c63756c617465000000000800000000000500")
	stream = ThriftStream{tcpStream: nil, data: data, message: new(ThriftMessage)}
	ok, complete = thrift.messageParser(&stream)
	m = stream.message
	if !ok || !complete {
		t.Error("Bad result:", ok, complete)
	}
	if m.IsRequest || m.Method != "calculate" ||
		m.SeqId != 0 || m.Type != ThriftMsgTypeReply ||
		m.Result != `(0: 5)` {
		t.Error("Bad result:", stream.message)
	}

	data, _ = hex.DecodeString("800100020000000963616c63756c617465000000000c000108000100" +
		"0000040b00020000001243616e6e6f742064697669646520627920300000")
	stream = ThriftStream{tcpStream: nil, data: data, message: new(ThriftMessage)}
	ok, complete = thrift.messageParser(&stream)
	m = stream.message
	if !ok || !complete {
		t.Error("Bad result:", ok, complete)
	}
	if m.IsRequest || m.Method != "calculate" ||
		m.SeqId != 0 || m.Type != ThriftMsgTypeReply ||
		m.Result != `(1: (1: 4, 2: "Cannot divide by 0"))` {
		t.Error("Bad result:", stream.message)
	}
}

// Returns a minimally filled Packet struct, only the payload
// having value.
func createTestPacket(t *testing.T, hexstr string) *Packet {
	data, err := hex.DecodeString(hexstr)
	if err != nil {
		t.Error("Failed to decode hex string")
		return nil
	}

	return &Packet{
		payload: data,
	}
}

// Helper function to read from the Publisher Queue
func expectThriftTransaction(t *testing.T, thrift Thrift) *ThriftTransaction {
	select {
	case trans := <-thrift.PublishQueue:
		return trans
	default:
		t.Error("No transaction")
	}
	return nil
}

func TestThrift_ParseSimpleTBinary(t *testing.T) {

	if testing.Verbose() {
		LogInit(LOG_DEBUG, "", false, []string{"thrift", "thriftdetailed"})
	}

	var thrift Thrift
	thrift.Init()

	thrift.PublishQueue = make(chan *ThriftTransaction, 10)

	var tcp TcpStream
	tcp.tuple = &IpPortTuple{
		Src_ip: 1, Dst_ip: 1, Src_port: 9200, Dst_port: 9201,
	}

	req := createTestPacket(t, "800100010000000470696e670000000000")
	repl := createTestPacket(t, "800100020000000470696e670000000000")

	thrift.Parse(req, &tcp, 0)
	thrift.Parse(repl, &tcp, 1)

	trans := expectThriftTransaction(t, thrift)
	if trans.Request.Method != "ping" ||
		trans.Request.Params != "()" ||
		trans.Reply.Result != "()" {

		t.Error("Bad result:", trans)
	}
}

func TestThrift_ParseSimpleTFramed(t *testing.T) {

	if testing.Verbose() {
		LogInit(LOG_DEBUG, "", false, []string{"thrift", "thriftdetailed"})
	}

	var thrift Thrift
	thrift.Init()
	thrift.TransportType = ThriftTFramed

	thrift.PublishQueue = make(chan *ThriftTransaction, 10)

	var tcp TcpStream
	tcp.tuple = &IpPortTuple{
		Src_ip: 1, Dst_ip: 1, Src_port: 9200, Dst_port: 9201,
	}

	req := createTestPacket(t, "0000001e8001000100000003616464000000000800010000000108"+
		"00020000000100")
	repl := createTestPacket(t, "000000178001000200000003616464000000000800000000000200")

	thrift.Parse(req, &tcp, 0)
	thrift.Parse(repl, &tcp, 1)

	trans := expectThriftTransaction(t, thrift)
	if trans.Request.Method != "add" ||
		trans.Request.Params != "(1: 1, 2: 1)" ||
		trans.Reply.Result != "(0: 2)" {

		t.Error("Bad result:", trans)
	}

}

func TestThrift_ParseSimpleTFramedSplit(t *testing.T) {

	if testing.Verbose() {
		LogInit(LOG_DEBUG, "", false, []string{"thrift", "thriftdetailed"})
	}

	var thrift Thrift
	thrift.Init()
	thrift.TransportType = ThriftTFramed

	thrift.PublishQueue = make(chan *ThriftTransaction, 10)

	var tcp TcpStream
	tcp.tuple = &IpPortTuple{
		Src_ip: 1, Dst_ip: 1, Src_port: 9200, Dst_port: 9201,
	}

	req_half1 := createTestPacket(t, "0000001e8001000100")
	req_half2 := createTestPacket(t, "000003616464000000000800010000000108"+
		"00020000000100")
	repl_half1 := createTestPacket(t, "000000178001000200000003")
	repl_half2 := createTestPacket(t, "616464000000000800000000000200")

	thrift.Parse(req_half1, &tcp, 0)
	thrift.Parse(req_half2, &tcp, 0)
	thrift.Parse(repl_half1, &tcp, 1)
	thrift.Parse(repl_half2, &tcp, 1)

	trans := expectThriftTransaction(t, thrift)
	if trans.Request.Method != "add" ||
		trans.Request.Params != "(1: 1, 2: 1)" ||
		trans.Reply.Result != "(0: 2)" {

		t.Error("Bad result:", trans)
	}

}

func TestThrift_ParseSimpleTFramedSplitInterleaved(t *testing.T) {

	if testing.Verbose() {
		LogInit(LOG_DEBUG, "", false, []string{"thrift", "thriftdetailed"})
	}

	var thrift Thrift
	thrift.Init()
	thrift.TransportType = ThriftTFramed

	thrift.PublishQueue = make(chan *ThriftTransaction, 10)

	var tcp TcpStream
	tcp.tuple = &IpPortTuple{
		Src_ip: 1, Dst_ip: 1, Src_port: 9200, Dst_port: 9201,
	}

	req_half1 := createTestPacket(t, "0000001e8001000100")
	repl_half1 := createTestPacket(t, "000000178001000200000003")
	req_half2 := createTestPacket(t, "000003616464000000000800010000000108"+
		"00020000000100")
	repl_half2 := createTestPacket(t, "616464000000000800000000000200")

	thrift.Parse(req_half1, &tcp, 0)
	thrift.Parse(req_half2, &tcp, 0)
	thrift.Parse(repl_half1, &tcp, 1)
	thrift.Parse(repl_half2, &tcp, 1)

	trans := expectThriftTransaction(t, thrift)
	if trans.Request.Method != "add" ||
		trans.Request.Params != "(1: 1, 2: 1)" ||
		trans.Reply.Result != "(0: 2)" {

		t.Error("Bad result:", trans)
	}
}

func TestThrift_Parse_OneWayCallWithFin(t *testing.T) {

	if testing.Verbose() {
		LogInit(LOG_DEBUG, "", false, []string{"thrift", "thriftdetailed"})
	}

	var thrift Thrift
	thrift.Init()
	thrift.TransportType = ThriftTFramed

	thrift.PublishQueue = make(chan *ThriftTransaction, 10)

	var tcp TcpStream
	tcp.tuple = &IpPortTuple{
		Src_ip: 1, Dst_ip: 1, Src_port: 9200, Dst_port: 9201,
	}

	req := createTestPacket(t, "0000001080010001000000037a69700000000000")

	thrift.Parse(req, &tcp, 0)
	thrift.ReceivedFin(&tcp, 0)

	trans := expectThriftTransaction(t, thrift)
	if trans.Request.Method != "zip" ||
		trans.Request.Params != "()" ||
		trans.Reply != nil || trans.ResponseTime != 0 {

		t.Error("Bad result:", trans)
	}
}

func TestThrift_Parse_OneWayCall2Requests(t *testing.T) {

	if testing.Verbose() {
		LogInit(LOG_DEBUG, "", false, []string{"thrift", "thriftdetailed"})
	}

	var thrift Thrift
	thrift.Init()
	thrift.TransportType = ThriftTFramed
	thrift.PublishQueue = make(chan *ThriftTransaction, 10)

	var tcp TcpStream
	tcp.tuple = &IpPortTuple{
		Src_ip: 1, Dst_ip: 1, Src_port: 9200, Dst_port: 9201,
	}

	reqzip := createTestPacket(t, "0000001080010001000000037a69700000000000")
	req := createTestPacket(t, "0000001e8001000100000003616464000000000800010000000108"+
		"00020000000100")
	repl := createTestPacket(t, "000000178001000200000003616464000000000800000000000200")

	thrift.Parse(reqzip, &tcp, 0)
	thrift.Parse(req, &tcp, 0)
	thrift.Parse(repl, &tcp, 1)

	trans := expectThriftTransaction(t, thrift)
	if trans.Request.Method != "zip" ||
		trans.Request.Params != "()" ||
		trans.Reply != nil || trans.ResponseTime != 0 {

		t.Error("Bad result:", trans)
	}

	trans = expectThriftTransaction(t, thrift)
	if trans.Request.Method != "add" ||
		trans.Request.Params != "(1: 1, 2: 1)" ||
		trans.Reply.Result != "(0: 2)" {

		t.Error("Bad result:", trans)
	}
}

func TestThrift_Parse_RequestReplyMismatch(t *testing.T) {

	if testing.Verbose() {
		LogInit(LOG_DEBUG, "", false, []string{"thrift", "thriftdetailed"})
	}

	var thrift Thrift
	thrift.Init()
	thrift.TransportType = ThriftTFramed
	thrift.PublishQueue = make(chan *ThriftTransaction, 10)

	var tcp TcpStream
	tcp.tuple = &IpPortTuple{
		Src_ip: 1, Dst_ip: 1, Src_port: 9200, Dst_port: 9201,
	}

	reqzip := createTestPacket(t, "0000001080010001000000037a69700000000000")
	repladd := createTestPacket(t, "000000178001000200000003616464000000000800000000000200")

	thrift.Parse(reqzip, &tcp, 0)
	thrift.Parse(repladd, &tcp, 1)
}

func TestThrift_ParseSimpleTFramed_NoReply(t *testing.T) {

	if testing.Verbose() {
		LogInit(LOG_DEBUG, "", false, []string{"thrift", "thriftdetailed"})
	}

	var thrift Thrift
	thrift.Init()
	thrift.TransportType = ThriftTFramed
	thrift.CaptureReply = false

	thrift.PublishQueue = make(chan *ThriftTransaction, 10)

	var tcp TcpStream
	tcp.tuple = &IpPortTuple{
		Src_ip: 1, Dst_ip: 1, Src_port: 9200, Dst_port: 9201,
	}

	req := createTestPacket(t, "0000001e8001000100000003616464000000000800010000000108"+
		"00020000000100")
	repl := createTestPacket(t, "000000178001000200000003616464000000000800000000000200")

	thrift.Parse(req, &tcp, 0)
	thrift.Parse(repl, &tcp, 1)

	trans := expectThriftTransaction(t, thrift)
	if trans.Request.Method != "add" ||
		trans.Request.Params != "(1: 1, 2: 1)" ||
		trans.Reply.Result != "" {

		t.Error("Bad result:", trans)
	}

	// play it again in the same stream
	thrift.Parse(req, &tcp, 0)
	thrift.Parse(repl, &tcp, 1)

	trans = expectThriftTransaction(t, thrift)
	if trans.Request.Method != "add" ||
		trans.Request.Params != "(1: 1, 2: 1)" ||
		trans.Reply.Result != "" {

		t.Error("Bad result:", trans)
	}
}

func TestThrift_ParseObfuscateStrings(t *testing.T) {

	if testing.Verbose() {
		LogInit(LOG_DEBUG, "", false, []string{"thrift", "thriftdetailed"})
	}

	var thrift Thrift
	thrift.Init()
	thrift.TransportType = ThriftTFramed
	thrift.ObfuscateStrings = true

	thrift.PublishQueue = make(chan *ThriftTransaction, 10)

	var tcp TcpStream
	tcp.tuple = &IpPortTuple{
		Src_ip: 1, Dst_ip: 1, Src_port: 9200, Dst_port: 9201,
	}

	req := createTestPacket(t, "00000024800100010000000b6563686f5f737472696e670000" +
		"00000b00010000000568656c6c6f00")
	repl := createTestPacket(t, "00000024800100020000000b6563686f5f737472696e67000" +
		"000000b00000000000568656c6c6f00")

	thrift.Parse(req, &tcp, 0)
	thrift.Parse(repl, &tcp, 1)

	trans := expectThriftTransaction(t, thrift)
	if trans.Request.Method != "echo_string" ||
		trans.Request.Params != `(1: "*")` ||
		trans.Reply.Result != `(0: "*")` {

		t.Error("Bad result:", trans)
	}
}

func BenchmarkThrift_ParseSkipReply(b *testing.B) {

	if testing.Verbose() {
		LogInit(LOG_DEBUG, "", false, []string{"thrift", "thriftdetailed"})
	}

	var thrift Thrift
	thrift.Init()
	thrift.TransportType = ThriftTFramed
	thrift.PublishQueue = make(chan *ThriftTransaction, 10)
	thrift.CaptureReply = false

	var tcp TcpStream
	tcp.tuple = &IpPortTuple{
		Src_ip: 1, Dst_ip: 1, Src_port: 9200, Dst_port: 9201,
	}

	data_req, _ := hex.DecodeString("0000001e8001000100000003616464000000000800010000000108"+
		"00020000000100")
	req := &Packet{payload: data_req}
	data_repl, _ := hex.DecodeString("000000178001000200000003616464000000000800000000000200")
	repl := &Packet{payload: data_repl}

	for n:=0; n < b.N; n++ {
		thrift.Parse(req, &tcp, 0)
		thrift.Parse(repl, &tcp, 1)

		select {
		case trans := <-thrift.PublishQueue:

			if trans.Request.Method != "add" ||
				trans.Request.Params != "(1: 1, 2: 1)" {

				b.Error("Bad result:", trans)
			}
		default:
			b.Error("No transaction")
		}

		// next should be empty
		select {
		case trans := <-thrift.PublishQueue:
			b.Error("Transaction still in queue: ", trans)
		default:
			// ok
		}
	}
}
