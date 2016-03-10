// +build !integration

package thrift

import (
	"encoding/hex"
	"net"
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/packetbeat/protos"
)

func thriftForTests() *Thrift {
	t := &Thrift{}
	config := defaultConfig
	t.init(true, nil, &config)
	return t
}

func TestThrift_thriftReadString(t *testing.T) {

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"thrift", "thriftdetailed"})
	}

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

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"thrift", "thriftdetailed"})
	}

	var data []byte
	var ok, complete bool
	var stream ThriftStream
	var m *ThriftMessage

	var thrift Thrift
	thrift.InitDefaults()

	data, _ = hex.DecodeString("800100010000000470696e670000000000")
	stream = ThriftStream{data: data, message: new(ThriftMessage)}
	m = stream.message
	ok, complete = thrift.readMessageBegin(&stream)
	if !ok || !complete {
		t.Errorf("Bad result: %v %v", ok, complete)
	}
	if m.Method != "ping" || m.Type != ThriftMsgTypeCall ||
		m.SeqId != 0 || m.Version != ThriftVersion1 {
		t.Errorf("Bad values: %v %v %v %v", m.Method, m.Type, m.SeqId, m.Version)
	}

	data, _ = hex.DecodeString("800100010000000470696e6700000000")
	stream = ThriftStream{data: data, message: new(ThriftMessage)}
	m = stream.message
	ok, complete = thrift.readMessageBegin(&stream)
	if !ok || !complete {
		t.Errorf("Bad result: %v %v", ok, complete)
	}
	if m.Method != "ping" || m.Type != ThriftMsgTypeCall ||
		m.SeqId != 0 || m.Version != ThriftVersion1 {
		t.Errorf("Bad values: %v %v %v %v", m.Method, m.Type, m.SeqId, m.Version)
	}

	data, _ = hex.DecodeString("800100010000000470696e6700000001")
	stream = ThriftStream{data: data, message: new(ThriftMessage)}
	m = stream.message
	ok, complete = thrift.readMessageBegin(&stream)
	if !ok || !complete {
		t.Errorf("Bad result: %v %v", ok, complete)
	}
	if m.Method != "ping" || m.Type != ThriftMsgTypeCall ||
		m.SeqId != 1 || m.Version != ThriftVersion1 {
		t.Errorf("Bad values: %v %v %v %v", m.Method, m.Type, m.SeqId, m.Version)
	}

	data, _ = hex.DecodeString("800100010000000570696e6700000001")
	stream = ThriftStream{data: data, message: new(ThriftMessage)}
	m = stream.message
	ok, complete = thrift.readMessageBegin(&stream)
	if !ok || complete {
		t.Errorf("Bad result: %v %v", ok, complete)
	}

	data, _ = hex.DecodeString("800100010000000570696e6700000001")
	stream = ThriftStream{data: data, message: new(ThriftMessage)}
	m = stream.message
	ok, complete = thrift.readMessageBegin(&stream)
	if !ok || complete {
		t.Errorf("Bad result: %v %v", ok, complete)
	}

	data, _ = hex.DecodeString("0000000470696e670100000000")
	stream = ThriftStream{data: data, message: new(ThriftMessage)}
	m = stream.message
	ok, complete = thrift.readMessageBegin(&stream)
	if !ok || !complete {
		t.Errorf("Bad result: %v %v", ok, complete)
	}
	if m.Method != "ping" || m.Type != ThriftMsgTypeCall ||
		m.SeqId != 0 || m.Version != 0 {
		t.Errorf("Bad values: %v %v %v %v", m.Method, m.Type, m.SeqId, m.Version)
	}

	data, _ = hex.DecodeString("0000000570696e670100000000")
	stream = ThriftStream{data: data, message: new(ThriftMessage)}
	m = stream.message
	ok, complete = thrift.readMessageBegin(&stream)
	if !ok || complete {
		t.Error("Bad result:", ok, complete)
	}

}

func TestThrift_thriftReadField(t *testing.T) {

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"thrift", "thriftdetailed"})
	}

	var data []byte
	var ok, complete bool
	var stream ThriftStream
	var field *ThriftField
	var _old int

	var thrift Thrift
	thrift.InitDefaults()

	data, _ = hex.DecodeString("08000100000001")
	stream = ThriftStream{data: data, message: new(ThriftMessage)}
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
	stream = ThriftStream{data: data, message: new(ThriftMessage)}
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
	stream = ThriftStream{data: data, message: new(ThriftMessage)}
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
	stream = ThriftStream{data: data, message: new(ThriftMessage)}
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
	stream = ThriftStream{data: data, message: new(ThriftMessage)}
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
	stream = ThriftStream{data: data, message: new(ThriftMessage)}
	ok, complete, field = thrift.readField(&stream)
	if !ok || complete || field != nil {
		t.Error("Bad result:", ok, complete, field)
	}

	data, _ = hex.DecodeString("0b00010000000568656c6c6f")
	stream = ThriftStream{data: data, message: new(ThriftMessage)}
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
	stream = ThriftStream{data: data, message: new(ThriftMessage)}
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
	stream = ThriftStream{data: data, message: new(ThriftMessage)}
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
	stream = ThriftStream{data: data, message: new(ThriftMessage)}
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
	stream = ThriftStream{data: data, message: new(ThriftMessage)}
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
	stream = ThriftStream{data: data, message: new(ThriftMessage)}
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
	stream = ThriftStream{data: data, message: new(ThriftMessage)}
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
	stream = ThriftStream{data: data, message: new(ThriftMessage)}
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
	stream = ThriftStream{data: data, message: new(ThriftMessage)}
	ok, complete, field = thrift.readField(&stream)
	if !ok || complete || field == nil {
		t.Error("Bad result:", ok, complete, field)
	} else {
		if field.Id != 1 || field.Type != ThriftTypeString || field.Value != `"h\x10llo"` {
			t.Error("Bad values:", field.Id, field.Type, field.Value)
		}
	}

	data, _ = hex.DecodeString("0c000108000100000001080002000000000800030000000400")
	stream = ThriftStream{data: data, message: new(ThriftMessage)}
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
	stream = ThriftStream{data: data, message: new(ThriftMessage)}
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
	stream = ThriftStream{data: data, message: new(ThriftMessage)}
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
	stream = ThriftStream{data: data, message: new(ThriftMessage)}
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
	stream = ThriftStream{data: data, message: new(ThriftMessage)}
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
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"thrift", "thriftdetailed"})
	}

	var data []byte
	var stream ThriftStream
	var ok, complete bool
	var m *ThriftMessage

	var thrift Thrift
	thrift.InitDefaults()

	data, _ = hex.DecodeString("800100010000000470696e670000000000")
	stream = ThriftStream{data: data, message: new(ThriftMessage)}
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
	stream = ThriftStream{data: data, message: new(ThriftMessage)}
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
	stream = ThriftStream{data: data, message: new(ThriftMessage)}
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
	stream = ThriftStream{data: data, message: new(ThriftMessage)}
	ok, complete = thrift.messageParser(&stream)
	m = stream.message
	if !ok || !complete {
		t.Error("Bad result:", ok, complete)
	}
	if m.IsRequest || m.Method != "add16" ||
		m.SeqId != 0 || m.Type != ThriftMsgTypeReply ||
		m.ReturnValue != "2" {
		t.Error("Bad result:", stream.message)
	}

	data, _ = hex.DecodeString("800100020000000b6563686f5f737472696e67000000000b00" +
		"000000000568656c6c6f00")
	stream = ThriftStream{data: data, message: new(ThriftMessage)}
	ok, complete = thrift.messageParser(&stream)
	m = stream.message
	if !ok || !complete {
		t.Error("Bad result:", ok, complete)
	}
	if m.IsRequest || m.Method != "echo_string" ||
		m.SeqId != 0 || m.Type != ThriftMsgTypeReply ||
		m.ReturnValue != `"hello"` {
		t.Error("Bad result:", stream.message)
	}

	data, _ = hex.DecodeString("80010002000000096563686f5f6c697374000000000f0000060" +
		"000000300010002000300")
	stream = ThriftStream{data: data, message: new(ThriftMessage)}
	ok, complete = thrift.messageParser(&stream)
	m = stream.message
	if !ok || !complete {
		t.Error("Bad result:", ok, complete)
	}
	if m.IsRequest || m.Method != "echo_list" ||
		m.SeqId != 0 || m.Type != ThriftMsgTypeReply ||
		m.ReturnValue != `[1, 2, 3]` {
		t.Error("Bad result:", stream.message)
	}

	data, _ = hex.DecodeString("80010002000000086563686f5f6d6170000000000d00000b06000" +
		"0000300000001610001000000016300030000000162000200")
	stream = ThriftStream{data: data, message: new(ThriftMessage)}
	ok, complete = thrift.messageParser(&stream)
	m = stream.message
	if !ok || !complete {
		t.Error("Bad result:", ok, complete)
	}
	if m.IsRequest || m.Method != "echo_map" ||
		m.SeqId != 0 || m.Type != ThriftMsgTypeReply ||
		m.ReturnValue != `{"a": 1, "c": 3, "b": 2}` {
		t.Error("Bad result:", stream.message)
	}

	data, _ = hex.DecodeString("800100020000000963616c63756c617465000000000800000000000500")
	stream = ThriftStream{data: data, message: new(ThriftMessage)}
	ok, complete = thrift.messageParser(&stream)
	m = stream.message
	if !ok || !complete {
		t.Error("Bad result:", ok, complete)
	}
	if m.IsRequest || m.Method != "calculate" ||
		m.SeqId != 0 || m.Type != ThriftMsgTypeReply || m.HasException ||
		m.ReturnValue != `5` {
		t.Error("Bad result:", stream.message)
	}

	data, _ = hex.DecodeString("800100020000000963616c63756c617465000000000c000108000100" +
		"0000040b00020000001243616e6e6f742064697669646520627920300000")
	stream = ThriftStream{data: data, message: new(ThriftMessage)}
	ok, complete = thrift.messageParser(&stream)
	m = stream.message
	if !ok || !complete {
		t.Error("Bad result:", ok, complete)
	}
	if m.IsRequest || m.Method != "calculate" ||
		m.SeqId != 0 || m.Type != ThriftMsgTypeReply || !m.HasException ||
		m.Exceptions != `(1: (1: 4, 2: "Cannot divide by 0"))` {
		t.Error("Bad result:", stream.message)
	}

	data, _ = hex.DecodeString("800100020000000b6563686f5f62696e61727900000000" +
		"0b000000000008ab0c1d281a00000000")
	stream = ThriftStream{data: data, message: new(ThriftMessage)}
	ok, complete = thrift.messageParser(&stream)
	m = stream.message
	if !ok || !complete {
		t.Error("Bad result:", ok, complete)
	}
	if m.IsRequest || m.Method != "echo_binary" ||
		m.SeqId != 0 || m.Type != ThriftMsgTypeReply || m.HasException ||
		m.ReturnValue != `ab0c1d281a000000` {
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
func expectThriftTransaction(t *testing.T, thrift *Thrift) *ThriftTransaction {
	select {
	case trans := <-thrift.PublishQueue:
		return trans
	default:
		t.Error("No transaction")
	}
	return nil
}

func testTcpTuple() *common.TcpTuple {
	t := &common.TcpTuple{
		Ip_length: 4,
		Src_ip:    net.IPv4(192, 168, 0, 1), Dst_ip: net.IPv4(192, 168, 0, 2),
		Src_port: 9200, Dst_port: 9201,
	}
	t.ComputeHashebles()
	return t
}

func TestThrift_ParseSimpleTBinary(t *testing.T) {

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"thrift", "thriftdetailed"})
	}

	thrift := thriftForTests()
	thrift.PublishQueue = make(chan *ThriftTransaction, 10)

	tcptuple := testTcpTuple()
	req := createTestPacket(t, "800100010000000470696e670000000000")
	repl := createTestPacket(t, "800100020000000470696e670000000000")

	var private thriftPrivateData
	thrift.Parse(req, tcptuple, 0, private)
	thrift.Parse(repl, tcptuple, 1, private)

	trans := expectThriftTransaction(t, thrift)
	if trans.Request.Method != "ping" ||
		trans.Request.Params != "()" ||
		trans.Reply.ReturnValue != "" ||
		trans.Request.FrameSize == 0 ||
		trans.Reply.FrameSize == 0 {

		t.Error("Bad result:", trans)
	}
}

func TestThrift_ParseSimpleTFramed(t *testing.T) {

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"thrift", "thriftdetailed"})
	}

	thrift := thriftForTests()
	thrift.TransportType = ThriftTFramed
	thrift.PublishQueue = make(chan *ThriftTransaction, 10)

	tcptuple := testTcpTuple()

	req := createTestPacket(t, "0000001e8001000100000003616464000000000800010000000108"+
		"00020000000100")
	repl := createTestPacket(t, "000000178001000200000003616464000000000800000000000200")

	var private thriftPrivateData

	thrift.Parse(req, tcptuple, 0, private)
	thrift.Parse(repl, tcptuple, 1, private)

	trans := expectThriftTransaction(t, thrift)
	if trans.Request.Method != "add" ||
		trans.Request.Params != "(1: 1, 2: 1)" ||
		trans.Reply.ReturnValue != "2" {

		t.Error("Bad result:", trans)
	}

}

func TestThrift_ParseSimpleTFramedSplit(t *testing.T) {

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"thrift", "thriftdetailed"})
	}

	thrift := thriftForTests()
	thrift.TransportType = ThriftTFramed
	thrift.PublishQueue = make(chan *ThriftTransaction, 10)

	tcptuple := testTcpTuple()

	req_half1 := createTestPacket(t, "0000001e8001000100")
	req_half2 := createTestPacket(t, "000003616464000000000800010000000108"+
		"00020000000100")
	repl_half1 := createTestPacket(t, "000000178001000200000003")
	repl_half2 := createTestPacket(t, "616464000000000800000000000200")

	var private thriftPrivateData
	private = thrift.Parse(req_half1, tcptuple, 0, private).(thriftPrivateData)
	private = thrift.Parse(req_half2, tcptuple, 0, private).(thriftPrivateData)
	private = thrift.Parse(repl_half1, tcptuple, 1, private).(thriftPrivateData)
	thrift.Parse(repl_half2, tcptuple, 1, private)

	trans := expectThriftTransaction(t, thrift)
	if trans.Request.Method != "add" ||
		trans.Request.Params != "(1: 1, 2: 1)" ||
		trans.Reply.ReturnValue != "2" {

		t.Error("Bad result:", trans)
	}

}

func TestThrift_ParseSimpleTFramedSplitInterleaved(t *testing.T) {

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"thrift", "thriftdetailed"})
	}

	thrift := thriftForTests()
	thrift.TransportType = ThriftTFramed
	thrift.PublishQueue = make(chan *ThriftTransaction, 10)

	tcptuple := testTcpTuple()

	req_half1 := createTestPacket(t, "0000001e8001000100")
	repl_half1 := createTestPacket(t, "000000178001000200000003")
	req_half2 := createTestPacket(t, "000003616464000000000800010000000108"+
		"00020000000100")
	repl_half2 := createTestPacket(t, "616464000000000800000000000200")

	var private thriftPrivateData
	private = thrift.Parse(req_half1, tcptuple, 0, private).(thriftPrivateData)
	private = thrift.Parse(req_half2, tcptuple, 0, private).(thriftPrivateData)
	private = thrift.Parse(repl_half1, tcptuple, 1, private).(thriftPrivateData)
	thrift.Parse(repl_half2, tcptuple, 1, private)

	trans := expectThriftTransaction(t, thrift)
	if trans.Request.Method != "add" ||
		trans.Request.Params != "(1: 1, 2: 1)" ||
		trans.Reply.ReturnValue != "2" {

		t.Error("Bad result:", trans)
	}
}

func TestThrift_Parse_OneWayCallWithFin(t *testing.T) {

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"thrift", "thriftdetailed"})
	}

	thrift := thriftForTests()
	thrift.TransportType = ThriftTFramed
	thrift.PublishQueue = make(chan *ThriftTransaction, 10)

	tcptuple := testTcpTuple()

	req := createTestPacket(t, "0000001080010001000000037a69700000000000")

	var private thriftPrivateData
	thrift.Parse(req, tcptuple, 0, private)
	thrift.ReceivedFin(tcptuple, 0, private)

	trans := expectThriftTransaction(t, thrift)
	if trans.Request.Method != "zip" ||
		trans.Request.Params != "()" ||
		trans.Reply != nil || trans.ResponseTime != 0 {

		t.Error("Bad result:", trans)
	}
}

func TestThrift_Parse_OneWayCall2Requests(t *testing.T) {

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"thrift", "thriftdetailed"})
	}

	thrift := thriftForTests()
	thrift.TransportType = ThriftTFramed
	thrift.PublishQueue = make(chan *ThriftTransaction, 10)

	tcptuple := testTcpTuple()

	reqzip := createTestPacket(t, "0000001080010001000000037a69700000000000")
	req := createTestPacket(t, "0000001e8001000100000003616464000000000800010000000108"+
		"00020000000100")
	repl := createTestPacket(t, "000000178001000200000003616464000000000800000000000200")

	var private thriftPrivateData
	thrift.Parse(reqzip, tcptuple, 0, private)
	thrift.Parse(req, tcptuple, 0, private)
	thrift.Parse(repl, tcptuple, 1, private)

	trans := expectThriftTransaction(t, thrift)
	if trans.Request.Method != "zip" ||
		trans.Request.Params != "()" ||
		trans.Reply != nil || trans.ResponseTime != 0 {

		t.Error("Bad result:", trans)
	}

	trans = expectThriftTransaction(t, thrift)
	if trans.Request.Method != "add" ||
		trans.Request.Params != "(1: 1, 2: 1)" ||
		trans.Reply.ReturnValue != "2" {

		t.Error("Bad result:", trans)
	}
}

func TestThrift_Parse_RequestReplyMismatch(t *testing.T) {

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"thrift", "thriftdetailed"})
	}

	thrift := thriftForTests()
	thrift.TransportType = ThriftTFramed
	thrift.PublishQueue = make(chan *ThriftTransaction, 10)

	tcptuple := testTcpTuple()

	reqzip := createTestPacket(t, "0000001080010001000000037a69700000000000")
	repladd := createTestPacket(t, "000000178001000200000003616464000000000800000000000200")

	var private thriftPrivateData
	thrift.Parse(reqzip, tcptuple, 0, private)
	thrift.Parse(repladd, tcptuple, 1, private)

	// Nothing should be received at this point
	select {
	case trans := <-thrift.PublishQueue:
		t.Error("Bad result:", trans)
	default:
		// ok
	}
}

func TestThrift_ParseSimpleTFramed_NoReply(t *testing.T) {

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"thrift", "thriftdetailed"})
	}

	thrift := thriftForTests()
	thrift.TransportType = ThriftTFramed
	thrift.CaptureReply = false
	thrift.PublishQueue = make(chan *ThriftTransaction, 10)

	tcptuple := testTcpTuple()

	req := createTestPacket(t, "0000001e8001000100000003616464000000000800010000000108"+
		"00020000000100")
	repl := createTestPacket(t, "000000178001000200000003616464000000000800000000000200")

	var private thriftPrivateData
	thrift.Parse(req, tcptuple, 0, private)
	thrift.Parse(repl, tcptuple, 1, private)

	trans := expectThriftTransaction(t, thrift)
	if trans.Request.Method != "add" ||
		trans.Request.Params != "(1: 1, 2: 1)" ||
		trans.Reply.ReturnValue != "" {

		t.Error("Bad result:", trans)
	}

	// play it again in the same stream
	thrift.Parse(req, tcptuple, 0, private)
	thrift.Parse(repl, tcptuple, 1, private)

	trans = expectThriftTransaction(t, thrift)
	if trans.Request.Method != "add" ||
		trans.Request.Params != "(1: 1, 2: 1)" ||
		trans.Reply.ReturnValue != "" {

		t.Error("Bad result:", trans)
	}
}

func TestThrift_ParseObfuscateStrings(t *testing.T) {

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"thrift", "thriftdetailed"})
	}

	thrift := thriftForTests()
	thrift.TransportType = ThriftTFramed
	thrift.ObfuscateStrings = true
	thrift.PublishQueue = make(chan *ThriftTransaction, 10)

	tcptuple := testTcpTuple()

	req := createTestPacket(t, "00000024800100010000000b6563686f5f737472696e670000"+
		"00000b00010000000568656c6c6f00")
	repl := createTestPacket(t, "00000024800100020000000b6563686f5f737472696e67000"+
		"000000b00000000000568656c6c6f00")

	var private thriftPrivateData
	thrift.Parse(req, tcptuple, 0, private)
	thrift.Parse(repl, tcptuple, 1, private)

	trans := expectThriftTransaction(t, thrift)
	if trans.Request.Method != "echo_string" ||
		trans.Request.Params != `(1: "*")` ||
		trans.Reply.ReturnValue != `"*"` {

		t.Error("Bad result:", trans)
	}
}

func BenchmarkThrift_ParseSkipReply(b *testing.B) {

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"thrift", "thriftdetailed"})
	}

	thrift := thriftForTests()
	thrift.TransportType = ThriftTFramed
	thrift.PublishQueue = make(chan *ThriftTransaction, 10)
	thrift.CaptureReply = false

	tcptuple := testTcpTuple()

	data_req, _ := hex.DecodeString("0000001e8001000100000003616464000000000800010000000108" +
		"00020000000100")
	req := &protos.Packet{Payload: data_req}
	data_repl, _ := hex.DecodeString("000000178001000200000003616464000000000800000000000200")
	repl := &protos.Packet{Payload: data_repl}

	var private thriftPrivateData
	for n := 0; n < b.N; n++ {
		thrift.Parse(req, tcptuple, 0, private)
		thrift.Parse(repl, tcptuple, 1, private)

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

func TestThrift_Parse_Exception(t *testing.T) {

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"thrift", "thriftdetailed"})
	}

	thrift := thriftForTests()
	thrift.PublishQueue = make(chan *ThriftTransaction, 10)

	tcptuple := testTcpTuple()

	req := createTestPacket(t, "800100010000000963616c63756c6174650000000008000"+
		"1000000010c00020800010000000108000200000000080003000000040000")
	repl := createTestPacket(t, "800100020000000963616c63756c617465000000000c00"+
		"01080001000000040b00020000001243616e6e6f742064697669646520627920300000")

	var private thriftPrivateData
	thrift.Parse(req, tcptuple, 0, private)
	thrift.Parse(repl, tcptuple, 1, private)

	trans := expectThriftTransaction(t, thrift)
	if trans.Request.Method != "calculate" ||
		trans.Request.Params != "(1: 1, 2: (1: 1, 2: 0, 3: 4))" ||
		trans.Reply.Exceptions != `(1: (1: 4, 2: "Cannot divide by 0"))` ||
		!trans.Reply.HasException {

		t.Error("Bad result:", trans)
	}
}

func TestThrift_ParametersNames(t *testing.T) {

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"thrift", "thriftdetailed"})
	}

	thrift := thriftForTests()
	thrift.TransportType = ThriftTFramed
	thrift.Idl = thriftIdlForTesting(t, `
		service Test {
			   i32 add(1:i32 num1, 2: i32 num2)
		}
		`)

	thrift.PublishQueue = make(chan *ThriftTransaction, 10)

	tcptuple := testTcpTuple()

	req := createTestPacket(t, "0000001e8001000100000003616464000000000800010000000108"+
		"00020000000100")
	repl := createTestPacket(t, "000000178001000200000003616464000000000800000000000200")

	var private thriftPrivateData
	thrift.Parse(req, tcptuple, 0, private)
	thrift.Parse(repl, tcptuple, 1, private)

	trans := expectThriftTransaction(t, thrift)
	if trans.Request.Method != "add" ||
		trans.Request.Params != "(num1: 1, num2: 1)" ||
		trans.Reply.ReturnValue != "2" ||
		trans.Request.Service != "Test" {

		t.Error("Bad result:", trans)
	}

}

func TestThrift_ExceptionName(t *testing.T) {

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"thrift", "thriftdetailed"})
	}

	thrift := thriftForTests()
	thrift.Idl = thriftIdlForTesting(t, `
		exception InvalidOperation {
		  1: i32 what,
		  2: string why
		}
		service Test {
		   i32 calculate(1:i32 logid, 2:Work w) throws (1:InvalidOperation ouch),
		}
		`)

	thrift.PublishQueue = make(chan *ThriftTransaction, 10)

	tcptuple := testTcpTuple()

	req := createTestPacket(t, "800100010000000963616c63756c6174650000000008000"+
		"1000000010c00020800010000000108000200000000080003000000040000")
	repl := createTestPacket(t, "800100020000000963616c63756c617465000000000c00"+
		"01080001000000040b00020000001243616e6e6f742064697669646520627920300000")

	var private thriftPrivateData
	thrift.Parse(req, tcptuple, 0, private)
	thrift.Parse(repl, tcptuple, 1, private)

	trans := expectThriftTransaction(t, thrift)
	if trans.Request.Method != "calculate" ||
		trans.Request.Params != "(logid: 1, w: (1: 1, 2: 0, 3: 4))" ||
		trans.Reply.ReturnValue != "" ||
		trans.Reply.Exceptions != `(ouch: (1: 4, 2: "Cannot divide by 0"))` ||
		!trans.Reply.HasException ||
		trans.Request.Service != "Test" {

		t.Error("Bad result:", trans)
	}
}

func TestThrift_GapInStream_response(t *testing.T) {

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"thrift", "thriftdetailed"})
	}

	thrift := thriftForTests()
	thrift.Idl = thriftIdlForTesting(t, `
		exception InvalidOperation {
		  1: i32 what,
		  2: string why
		}
		service Test {
		   i32 calculate(1:i32 logid, 2:Work w) throws (1:InvalidOperation ouch),
		}
		`)

	thrift.PublishQueue = make(chan *ThriftTransaction, 10)

	tcptuple := testTcpTuple()

	req := createTestPacket(t, "800100010000000963616c63756c6174650000000008000"+
		"1000000010c00020800010000000108000200000000080003000000040000")
	// missing last few bytes
	repl := createTestPacket(t, "800100020000000963616c63756c617465000000000c00"+
		"01080001000000040b00020000001243616e6e6f742064697669646520")

	private := protos.ProtocolData(new(thriftPrivateData))
	private = thrift.Parse(req, tcptuple, 0, private)
	private = thrift.Parse(repl, tcptuple, 1, private)
	private, drop := thrift.GapInStream(tcptuple, 1, 5, private)

	if drop == false {
		t.Error("GapInStream returned drop=false")
	}

	trans := expectThriftTransaction(t, thrift)
	// The exception is not captured, but otherwise the values from the request
	// are correct
	if trans.Request.Method != "calculate" ||
		trans.Request.Params != "(logid: 1, w: (1: 1, 2: 0, 3: 4))" ||
		trans.Reply.ReturnValue != "" ||
		trans.Reply.Exceptions != `` ||
		trans.Reply.HasException ||
		trans.Request.Service != "Test" ||
		trans.Reply.Notes[0] != "Packet loss while capturing the response" {

		t.Error("trans.Reply.Exceptions", trans.Reply.Exceptions)
		t.Error("trans.Reply.HasException", trans.Reply.HasException)
	}
}

func TestThrift_GapInStream_request(t *testing.T) {

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"thrift", "thriftdetailed"})
	}

	thrift := thriftForTests()
	thrift.Idl = thriftIdlForTesting(t, `
		exception InvalidOperation {
		  1: i32 what,
		  2: string why
		}
		service Test {
		   i32 calculate(1:i32 logid, 2:Work w) throws (1:InvalidOperation ouch),
		}
		`)

	thrift.PublishQueue = make(chan *ThriftTransaction, 10)

	tcptuple := testTcpTuple()

	// missing bytes from the request
	req := createTestPacket(t, "800100010000000963616c63756c6174")
	repl := createTestPacket(t, "800100020000000963616c63756c617465000000000c00"+
		"01080001000000040b00020000001243616e6e6f742064697669646520627920300000")

	private := protos.ProtocolData(new(thriftPrivateData))
	private = thrift.Parse(req, tcptuple, 0, private)
	private, drop := thrift.GapInStream(tcptuple, 0, 5, private)

	private = thrift.Parse(repl, tcptuple, 1, private)

	if drop == false {
		t.Error("GapInStream returned drop=false")
	}

	// packet loss in requests should result in no transaction
	select {
	case trans := <-thrift.PublishQueue:
		t.Error("Expected no transaction but got one:", trans)
	default:
		// ok
	}
}
