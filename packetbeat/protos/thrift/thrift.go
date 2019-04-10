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

package thrift

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/monitoring"

	"github.com/elastic/beats/packetbeat/pb"
	"github.com/elastic/beats/packetbeat/procs"
	"github.com/elastic/beats/packetbeat/protos"
	"github.com/elastic/beats/packetbeat/protos/tcp"
)

type thriftPlugin struct {

	// config
	ports                  []int
	stringMaxSize          int
	collectionMaxSize      int
	dropAfterNStructFields int
	captureReply           bool
	obfuscateStrings       bool
	sendRequest            bool
	sendResponse           bool

	TransportType byte
	ProtocolType  byte

	transactions       *common.Cache
	transactionTimeout time.Duration

	publishQueue chan *thriftTransaction
	results      protos.Reporter
	idl          *thriftIdl
}

type thriftMessage struct {
	ts time.Time

	tcpTuple     common.TCPTuple
	cmdlineTuple *common.ProcessTuple
	direction    uint8

	start int

	fields []thriftField

	isRequest    bool
	hasException bool
	version      uint32
	Type         uint32
	method       string
	seqID        uint32
	params       string
	returnValue  string
	exceptions   string
	frameSize    uint32
	service      string
	notes        []string
}

type thriftField struct {
	Type byte
	id   uint16

	value string
}

type thriftStream struct {
	tcptuple *common.TCPTuple

	data []byte

	parseOffset int
	parseState  int

	// when this is set, don't care about the
	// traffic in this direction. Used to skip large responses.
	skipInput bool

	message *thriftMessage
}

type thriftTransaction struct {
	tuple    common.TCPTuple
	src      common.Endpoint
	dst      common.Endpoint
	ts       time.Time
	bytesIn  uint64
	bytesOut uint64

	request *thriftMessage
	reply   *thriftMessage
}

const (
	thriftStartState = iota
	thriftFieldState
)

const (
	thriftVersionMask = 0xffff0000
	thriftVersion1    = 0x80010000
	ThriftTypeMask    = 0x000000ff
)

// Thrift types
const (
	ThriftTypeStop   = 0
	ThriftTypeVoid   = 1
	ThriftTypeBool   = 2
	ThriftTypeByte   = 3
	ThriftTypeDouble = 4
	ThriftTypeI16    = 6
	ThriftTypeI32    = 8
	ThriftTypeI64    = 10
	ThriftTypeString = 11
	ThriftTypeStruct = 12
	ThriftTypeMap    = 13
	ThriftTypeSet    = 14
	ThriftTypeList   = 15
	ThriftTypeUtf8   = 16
	ThriftTypeUtf16  = 17
)

// Thrift message types
const (
	_ = iota
	ThriftMsgTypeCall
	ThriftMsgTypeReply
	ThriftMsgTypeException
	ThriftMsgTypeOneway
)

// Thrift protocol types
const (
	thriftTBinary  = 1
	thriftTCompact = 2
)

// Thrift transport types
const (
	thriftTSocket = 1
	thriftTFramed = 2
)

var (
	unmatchedRequests  = monitoring.NewInt(nil, "thrift.unmatched_requests")
	unmatchedResponses = monitoring.NewInt(nil, "thrift.unmatched_responses")
)

func init() {
	protos.Register("thrift", New)
}

func New(
	testMode bool,
	results protos.Reporter,
	cfg *common.Config,
) (protos.Plugin, error) {
	p := &thriftPlugin{}
	config := defaultConfig
	if !testMode {
		if err := cfg.Unpack(&config); err != nil {
			return nil, err
		}
	}

	if err := p.init(testMode, results, &config); err != nil {
		return nil, err
	}
	return p, nil
}

func (thrift *thriftPlugin) init(
	testMode bool,
	results protos.Reporter,
	config *thriftConfig,
) error {
	thrift.InitDefaults()

	err := thrift.readConfig(config)
	if err != nil {
		return err
	}

	thrift.transactions = common.NewCache(
		thrift.transactionTimeout,
		protos.DefaultTransactionHashSize)
	thrift.transactions.StartJanitor(thrift.transactionTimeout)

	if !testMode {
		thrift.publishQueue = make(chan *thriftTransaction, 1000)
		thrift.results = results
		go thrift.publishTransactions()
	}

	return nil
}

func (thrift *thriftPlugin) getTransaction(k common.HashableTCPTuple) *thriftTransaction {
	v := thrift.transactions.Get(k)
	if v != nil {
		return v.(*thriftTransaction)
	}
	return nil
}

func (thrift *thriftPlugin) InitDefaults() {
	// defaults
	thrift.stringMaxSize = 200
	thrift.collectionMaxSize = 15
	thrift.dropAfterNStructFields = 500
	thrift.TransportType = thriftTSocket
	thrift.ProtocolType = thriftTBinary
	thrift.captureReply = true
	thrift.obfuscateStrings = false
	thrift.sendRequest = false
	thrift.sendResponse = false
	thrift.transactionTimeout = protos.DefaultTransactionExpiration
}

func (thrift *thriftPlugin) readConfig(config *thriftConfig) error {
	var err error

	thrift.ports = config.Ports
	thrift.sendRequest = config.SendRequest
	thrift.sendResponse = config.SendResponse

	thrift.stringMaxSize = config.StringMaxSize
	thrift.collectionMaxSize = config.CollectionMaxSize
	thrift.dropAfterNStructFields = config.DropAfterNStructFields
	thrift.captureReply = config.CaptureReply
	thrift.obfuscateStrings = config.ObfuscateStrings

	switch config.TransportType {
	case "socket":
		thrift.TransportType = thriftTSocket
	case "framed":
		thrift.TransportType = thriftTFramed
	default:
		return fmt.Errorf("Transport type `%s` not known", config.TransportType)
	}

	switch config.ProtocolType {
	case "binary":
		thrift.ProtocolType = thriftTBinary
	default:
		return fmt.Errorf("Protocol type `%s` not known", config.ProtocolType)
	}

	if len(config.IdlFiles) > 0 {
		thrift.idl, err = newThriftIdl(config.IdlFiles)
		if err != nil {
			return err
		}
	}

	return nil
}

func (thrift *thriftPlugin) GetPorts() []int {
	return thrift.ports
}

func (m *thriftMessage) String() string {
	return fmt.Sprintf("IsRequest: %t Type: %d Method: %s SeqId: %d Params: %s ReturnValue: %s Exceptions: %s",
		m.isRequest, m.Type, m.method, m.seqID, m.params, m.returnValue, m.exceptions)
}

func (thrift *thriftPlugin) readMessageBegin(s *thriftStream) (bool, bool) {
	var ok, complete bool
	var offset, off int

	m := s.message

	if len(s.data[s.parseOffset:]) < 9 {
		return true, false // ok, not complete
	}

	sz := common.BytesNtohl(s.data[s.parseOffset : s.parseOffset+4])
	if int32(sz) < 0 {
		m.version = sz & thriftVersionMask
		if m.version != thriftVersion1 {
			logp.Debug("thrift", "Unexpected version: %d", m.version)
		}

		logp.Debug("thriftdetailed", "version = %d", m.version)

		offset = s.parseOffset + 4

		logp.Debug("thriftdetailed", "offset = %d", offset)

		m.Type = sz & ThriftTypeMask
		m.method, ok, complete, off = thrift.readString(s.data[offset:])
		if !ok {
			return false, false // not ok, not complete
		}
		if !complete {
			logp.Debug("thriftdetailed", "Method name not complete")
			return true, false // ok, not complete
		}
		offset += off

		logp.Debug("thriftdetailed", "method = %s", m.method)
		logp.Debug("thriftdetailed", "offset = %d", offset)

		if len(s.data[offset:]) < 4 {
			logp.Debug("thriftdetailed", "Less then 4 bytes remaining")
			return true, false // ok, not complete
		}
		m.seqID = common.BytesNtohl(s.data[offset : offset+4])
		s.parseOffset = offset + 4
	} else {
		// no version mode
		offset = s.parseOffset

		m.method, ok, complete, off = thrift.readString(s.data[offset:])
		if !ok {
			return false, false // not ok, not complete
		}
		if !complete {
			logp.Debug("thriftdetailed", "Method name not complete")
			return true, false // ok, not complete
		}
		offset += off

		logp.Debug("thriftdetailed", "method = %s", m.method)
		logp.Debug("thriftdetailed", "offset = %d", offset)

		if len(s.data[offset:]) < 5 {
			return true, false // ok, not complete
		}

		m.Type = uint32(s.data[offset])
		offset++
		m.seqID = common.BytesNtohl(s.data[offset : offset+4])
		s.parseOffset = offset + 4
	}

	if m.Type == ThriftMsgTypeCall || m.Type == ThriftMsgTypeOneway {
		m.isRequest = true
	} else {
		m.isRequest = false
	}

	return true, true
}

// Functions to decode simple types
// They all have the same signature, returning the string value and the
// number of bytes consumed (off).
type thriftFieldReader func(data []byte) (value string, ok bool, complete bool, off int)

// thriftReadString caps the returned value to ThriftStringMaxSize but returns the
// off to the end of it.
func (thrift *thriftPlugin) readString(data []byte) (value string, ok bool, complete bool, off int) {
	if len(data) < 4 {
		return "", true, false, 0 // ok, not complete
	}
	sz := int(common.BytesNtohl(data[:4]))
	if int32(sz) < 0 {
		return "", false, false, 0 // not ok
	}
	if len(data[4:]) < sz {
		return "", true, false, 0 // ok, not complete
	}

	if sz > thrift.stringMaxSize {
		value = string(data[4 : 4+thrift.stringMaxSize])
		value += "..."
	} else {
		value = string(data[4 : 4+sz])
	}
	off = 4 + sz

	return value, true, true, off // all good
}

func (thrift *thriftPlugin) readAndQuoteString(data []byte) (value string, ok bool, complete bool, off int) {
	value, ok, complete, off = thrift.readString(data)
	if value == "" {
		value = `""`
	} else if thrift.obfuscateStrings {
		value = `"*"`
	} else {
		if utf8.ValidString(value) {
			value = strconv.Quote(value)
		} else {
			value = hex.EncodeToString([]byte(value))
		}
	}

	return value, ok, complete, off
}

func (thrift *thriftPlugin) readBool(data []byte) (value string, ok bool, complete bool, off int) {
	if len(data) < 1 {
		return "", true, false, 0
	}
	if data[0] == byte(0) {
		value = "false"
	} else {
		value = "true"
	}

	return value, true, true, 1
}

func (thrift *thriftPlugin) readByte(data []byte) (value string, ok bool, complete bool, off int) {
	if len(data) < 1 {
		return "", true, false, 0
	}
	value = strconv.Itoa(int(data[0]))

	return value, true, true, 1
}

func (thrift *thriftPlugin) readDouble(data []byte) (value string, ok bool, complete bool, off int) {
	if len(data) < 8 {
		return "", true, false, 0
	}

	bits := binary.BigEndian.Uint64(data[:8])
	double := math.Float64frombits(bits)
	value = strconv.FormatFloat(double, 'f', -1, 64)

	return value, true, true, 8
}

func (thrift *thriftPlugin) readI16(data []byte) (value string, ok bool, complete bool, off int) {
	if len(data) < 2 {
		return "", true, false, 0
	}
	i16 := common.BytesNtohs(data[:2])
	value = strconv.Itoa(int(i16))

	return value, true, true, 2
}

func (thrift *thriftPlugin) readI32(data []byte) (value string, ok bool, complete bool, off int) {
	if len(data) < 4 {
		return "", true, false, 0
	}
	i32 := common.BytesNtohl(data[:4])
	value = strconv.Itoa(int(i32))

	return value, true, true, 4
}

func (thrift *thriftPlugin) readI64(data []byte) (value string, ok bool, complete bool, off int) {
	if len(data) < 8 {
		return "", true, false, 0
	}
	i64 := common.BytesNtohll(data[:8])
	value = strconv.FormatInt(int64(i64), 10)

	return value, true, true, 8
}

// Common implementation for lists and sets (they share the same binary repr).
func (thrift *thriftPlugin) readListOrSet(data []byte) (value string, ok bool, complete bool, off int) {
	if len(data) < 5 {
		return "", true, false, 0
	}
	typ := data[0]

	funcReader, typeFound := thrift.funcReadersByType(typ)
	if !typeFound {
		logp.Debug("thrift", "Field type %d not known", typ)
		return "", false, false, 0
	}

	sz := int(common.BytesNtohl(data[1:5]))
	if sz < 0 {
		logp.Debug("thrift", "List/Set too big: %d", sz)
		return "", false, false, 0
	}

	fields := []string{}
	offset := 5

	for i := 0; i < sz; i++ {
		value, ok, complete, bytesRead := funcReader(data[offset:])
		if !ok {
			return "", false, false, 0
		}
		if !complete {
			return "", true, false, 0
		}

		if i < thrift.collectionMaxSize {
			fields = append(fields, value)
		} else if i == thrift.collectionMaxSize {
			fields = append(fields, "...")
		}
		offset += bytesRead
	}

	return strings.Join(fields, ", "), true, true, offset
}

func (thrift *thriftPlugin) readSet(data []byte) (value string, ok bool, complete bool, off int) {
	value, ok, complete, off = thrift.readListOrSet(data)
	if value != "" {
		value = "{" + value + "}"
	}
	return value, ok, complete, off
}

func (thrift *thriftPlugin) readList(data []byte) (value string, ok bool, complete bool, off int) {
	value, ok, complete, off = thrift.readListOrSet(data)
	if value != "" {
		value = "[" + value + "]"
	}
	return value, ok, complete, off
}

func (thrift *thriftPlugin) readMap(data []byte) (value string, ok bool, complete bool, off int) {
	if len(data) < 6 {
		return "", true, false, 0
	}
	typeKey := data[0]
	typeValue := data[1]

	funcReaderKey, typeFound := thrift.funcReadersByType(typeKey)
	if !typeFound {
		logp.Debug("thrift", "Field type %d not known", typeKey)
		return "", false, false, 0
	}

	funcReaderValue, typeFound := thrift.funcReadersByType(typeValue)
	if !typeFound {
		logp.Debug("thrift", "Field type %d not known", typeValue)
		return "", false, false, 0
	}

	sz := int(common.BytesNtohl(data[2:6]))
	if sz < 0 {
		logp.Debug("thrift", "Map too big: %d", sz)
		return "", false, false, 0
	}

	fields := []string{}
	offset := 6

	for i := 0; i < sz; i++ {
		key, ok, complete, bytesRead := funcReaderKey(data[offset:])
		if !ok {
			return "", false, false, 0
		}
		if !complete {
			return "", true, false, 0
		}
		offset += bytesRead

		value, ok, complete, bytesRead := funcReaderValue(data[offset:])
		if !ok {
			return "", false, false, 0
		}
		if !complete {
			return "", true, false, 0
		}
		offset += bytesRead

		if i < thrift.collectionMaxSize {
			fields = append(fields, key+": "+value)
		} else if i == thrift.collectionMaxSize {
			fields = append(fields, "...")
		}
	}

	return "{" + strings.Join(fields, ", ") + "}", true, true, offset
}

func (thrift *thriftPlugin) readStruct(data []byte) (value string, ok bool, complete bool, off int) {
	var bytesRead int
	offset := 0
	fields := []thriftField{}

	// Loop until hitting a STOP or reaching the maximum number of elements
	// we follow in a stream (at which point, we assume we interpreted something
	// wrong).
	for i := 0; ; i++ {
		var field thriftField

		if i >= thrift.dropAfterNStructFields {
			logp.Debug("thrift", "Too many fields in struct. Dropping as error")
			return "", false, false, 0
		}

		if len(data) < 1 {
			return "", true, false, 0
		}

		field.Type = byte(data[offset])
		offset++
		if field.Type == ThriftTypeStop {
			return thrift.formatStruct(fields, false, []*string{}), true, true, offset
		}

		if len(data[offset:]) < 2 {
			return "", true, false, 0 // not complete
		}

		field.id = common.BytesNtohs(data[offset : offset+2])
		offset += 2

		funcReader, typeFound := thrift.funcReadersByType(field.Type)
		if !typeFound {
			logp.Debug("thrift", "Field type %d not known", field.Type)
			return "", false, false, 0
		}

		field.value, ok, complete, bytesRead = funcReader(data[offset:])

		if !ok {
			return "", false, false, 0
		}
		if !complete {
			return "", true, false, 0
		}
		fields = append(fields, field)
		offset += bytesRead
	}
}

func (thrift *thriftPlugin) formatStruct(fields []thriftField, resolveNames bool,
	fieldnames []*string) string {

	toJoin := []string{}
	for i, field := range fields {
		if i == thrift.collectionMaxSize {
			toJoin = append(toJoin, "...")
			break
		}
		if resolveNames && int(field.id) < len(fieldnames) && fieldnames[field.id] != nil {
			toJoin = append(toJoin, *fieldnames[field.id]+": "+field.value)
		} else {
			toJoin = append(toJoin, strconv.Itoa(int(field.id))+": "+field.value)
		}
	}
	return "(" + strings.Join(toJoin, ", ") + ")"
}

// Dictionary wrapped in a function to avoid "initialization loop"
func (thrift *thriftPlugin) funcReadersByType(typ byte) (fn thriftFieldReader, exists bool) {
	switch typ {
	case ThriftTypeBool:
		return thrift.readBool, true
	case ThriftTypeByte:
		return thrift.readByte, true
	case ThriftTypeDouble:
		return thrift.readDouble, true
	case ThriftTypeI16:
		return thrift.readI16, true
	case ThriftTypeI32:
		return thrift.readI32, true
	case ThriftTypeI64:
		return thrift.readI64, true
	case ThriftTypeString:
		return thrift.readAndQuoteString, true
	case ThriftTypeList:
		return thrift.readList, true
	case ThriftTypeSet:
		return thrift.readSet, true
	case ThriftTypeMap:
		return thrift.readMap, true
	case ThriftTypeStruct:
		return thrift.readStruct, true
	default:
		return nil, false
	}
}

func (thrift *thriftPlugin) readField(s *thriftStream) (ok bool, complete bool, field *thriftField) {
	var off int

	field = new(thriftField)

	if len(s.data) == 0 {
		return true, false, nil // ok, not complete
	}
	field.Type = byte(s.data[s.parseOffset])
	offset := s.parseOffset + 1
	if field.Type == ThriftTypeStop {
		s.parseOffset = offset
		return true, true, nil // done
	}

	if len(s.data[offset:]) < 2 {
		return true, false, nil // ok, not complete
	}
	field.id = common.BytesNtohs(s.data[offset : offset+2])
	offset += 2

	funcReader, typeFound := thrift.funcReadersByType(field.Type)
	if !typeFound {
		logp.Debug("thrift", "Field type %d not known", field.Type)
		return false, false, nil
	}

	field.value, ok, complete, off = funcReader(s.data[offset:])

	if !ok {
		return false, false, nil
	}
	if !complete {
		return true, false, nil
	}
	offset += off

	s.parseOffset = offset
	return true, false, field
}

func (thrift *thriftPlugin) messageParser(s *thriftStream) (bool, bool) {
	var ok, complete bool
	var m = s.message

	logp.Debug("thriftdetailed", "messageParser called parseState=%v offset=%v",
		s.parseState, s.parseOffset)

	for s.parseOffset < len(s.data) {
		switch s.parseState {
		case thriftStartState:
			m.start = s.parseOffset
			if thrift.TransportType == thriftTFramed {
				// read I32
				if len(s.data) < 4 {
					return true, false
				}
				m.frameSize = common.BytesNtohl(s.data[:4])
				s.parseOffset = 4
			}

			ok, complete = thrift.readMessageBegin(s)
			logp.Debug("thriftdetailed", "readMessageBegin returned: %v %v", ok, complete)
			if !ok {
				return false, false
			}
			if !complete {
				return true, false
			}

			if !m.isRequest && !thrift.captureReply {
				// don't actually read the result
				logp.Debug("thrift", "Don't capture reply")
				m.returnValue = ""
				m.exceptions = ""
				return true, true
			}
			s.parseState = thriftFieldState
		case thriftFieldState:
			ok, complete, field := thrift.readField(s)
			logp.Debug("thriftdetailed", "readField returned: %v %v", ok, complete)
			if !ok {
				return false, false
			}
			if complete {
				// done
				var method *thriftIdlMethod
				if thrift.idl != nil {
					method = thrift.idl.findMethod(m.method)
				}
				if m.isRequest {
					if method != nil {
						m.params = thrift.formatStruct(m.fields, true, method.params)

						m.service = method.service.Name
					} else {
						m.params = thrift.formatStruct(m.fields, false, nil)
					}
				} else {
					if len(m.fields) > 1 {
						logp.Warn("Thrift RPC response with more than field. Ignoring all but first")
					}
					if len(m.fields) > 0 {
						field := m.fields[0]
						if field.id == 0 {
							m.returnValue = field.value
							m.exceptions = ""
						} else {
							m.returnValue = ""
							if method != nil {
								m.exceptions = thrift.formatStruct(m.fields, true, method.exceptions)
							} else {
								m.exceptions = thrift.formatStruct(m.fields, false, nil)
							}
							m.hasException = true
						}
					}
				}
				return true, true
			}
			if field == nil {
				return true, false // ok, not complete
			}

			m.fields = append(m.fields, *field)
		}
	}

	return true, false
}

// messageGap is called when a gap of size `nbytes` is found in the
// tcp stream. Returns true if there is already enough data in the message
// read so far that we can use it further in the stack.
func (thrift *thriftPlugin) messageGap(s *thriftStream, nbytes int) (complete bool) {
	m := s.message
	switch s.parseState {
	case thriftStartState:
		// not enough data yet to be useful
		return false
	case thriftFieldState:
		if !m.isRequest {
			// large response case, can tolerate loss
			m.notes = append(m.notes, "Packet loss while capturing the response")
			return true
		}
	}

	return false
}

func (stream *thriftStream) prepareForNewMessage(flush bool) {
	if flush {
		stream.data = []byte{}
	} else {
		stream.data = stream.data[stream.parseOffset:]
	}
	//logp.Debug("thrift", "remaining data: [%s]", stream.data)
	stream.parseOffset = 0
	stream.message = nil
	stream.parseState = thriftStartState
}

type thriftPrivateData struct {
	data [2]*thriftStream
}

func (thrift *thriftPlugin) messageComplete(tcptuple *common.TCPTuple, dir uint8,
	stream *thriftStream, priv *thriftPrivateData) {

	flush := false

	if stream.message.isRequest {
		logp.Debug("thrift", "Thrift request message: %s", stream.message.method)
		if !thrift.captureReply {
			// enable the stream in the other direction to get the reply
			streamRev := priv.data[1-dir]
			if streamRev != nil {
				streamRev.skipInput = false
			}
		}
	} else {
		logp.Debug("thrift", "Thrift response message: %s", stream.message.method)
		if !thrift.captureReply {
			// disable stream in this direction
			stream.skipInput = true

			// and flush current data
			flush = true
		}
	}

	// all ok, go to next level
	stream.message.tcpTuple = *tcptuple
	stream.message.direction = dir
	stream.message.cmdlineTuple = procs.ProcWatcher.FindProcessesTupleTCP(tcptuple.IPPort())
	if stream.message.frameSize == 0 {
		stream.message.frameSize = uint32(stream.parseOffset - stream.message.start)
	}
	thrift.handleThrift(stream.message)

	// and reset message
	stream.prepareForNewMessage(flush)

}

func (thrift *thriftPlugin) ConnectionTimeout() time.Duration {
	return thrift.transactionTimeout
}

func (thrift *thriftPlugin) Parse(pkt *protos.Packet, tcptuple *common.TCPTuple, dir uint8,
	private protos.ProtocolData) protos.ProtocolData {

	defer logp.Recover("ParseThrift exception")

	priv := thriftPrivateData{}
	if private != nil {
		var ok bool
		priv, ok = private.(thriftPrivateData)
		if !ok {
			priv = thriftPrivateData{}
		}
	}

	stream := priv.data[dir]

	if stream == nil {
		stream = &thriftStream{
			tcptuple: tcptuple,
			data:     pkt.Payload,
			message:  &thriftMessage{ts: pkt.Ts},
		}
		priv.data[dir] = stream
	} else {
		if stream.skipInput {
			// stream currently suspended in this direction
			return priv
		}
		// concatenate bytes
		stream.data = append(stream.data, pkt.Payload...)
		if len(stream.data) > tcp.TCPMaxDataInStream {
			logp.Debug("thrift", "Stream data too large, dropping TCP stream")
			priv.data[dir] = nil
			return priv
		}
	}

	for len(stream.data) > 0 {
		if stream.message == nil {
			stream.message = &thriftMessage{ts: pkt.Ts}
		}

		ok, complete := thrift.messageParser(priv.data[dir])
		logp.Debug("thriftdetailed", "messageParser returned %v %v", ok, complete)

		if !ok {
			// drop this tcp stream. Will retry parsing with the next
			// segment in it
			priv.data[dir] = nil
			logp.Debug("thrift", "Ignore Thrift message. Drop tcp stream. Try parsing with the next segment")
			return priv
		}

		if complete {
			thrift.messageComplete(tcptuple, dir, stream, &priv)
		} else {
			// wait for more data
			break
		}
	}

	logp.Debug("thriftdetailed", "Out")

	return priv
}

func (thrift *thriftPlugin) handleThrift(msg *thriftMessage) {
	if msg.isRequest {
		thrift.receivedRequest(msg)
	} else {
		thrift.receivedReply(msg)
	}
}

func (thrift *thriftPlugin) receivedRequest(msg *thriftMessage) {
	tuple := msg.tcpTuple

	trans := thrift.getTransaction(tuple.Hashable())
	if trans != nil {
		logp.Debug("thrift", "Two requests without reply, assuming the old one is oneway")
		unmatchedRequests.Add(1)
		thrift.publishQueue <- trans
	}

	trans = &thriftTransaction{
		tuple: tuple,
	}
	thrift.transactions.Put(tuple.Hashable(), trans)

	trans.ts = msg.ts
	trans.src, trans.dst = common.MakeEndpointPair(msg.tcpTuple.BaseTuple, msg.cmdlineTuple)
	if msg.direction == tcp.TCPDirectionReverse {
		trans.src, trans.dst = trans.dst, trans.src
	}

	trans.request = msg
	trans.bytesIn = uint64(msg.frameSize)
}

func (thrift *thriftPlugin) receivedReply(msg *thriftMessage) {
	// we need to search the request first.
	tuple := msg.tcpTuple

	trans := thrift.getTransaction(tuple.Hashable())
	if trans == nil {
		logp.Debug("thrift", "Response from unknown transaction. Ignoring: %v", tuple)
		unmatchedResponses.Add(1)
		return
	}

	if trans.request.method != msg.method {
		logp.Debug("thrift", "Response from another request received '%s' '%s'"+
			". Ignoring.", trans.request.method, msg.method)
		unmatchedResponses.Add(1)
		return
	}

	trans.reply = msg
	trans.bytesOut = uint64(msg.frameSize)

	thrift.publishQueue <- trans
	thrift.transactions.Delete(tuple.Hashable())

	logp.Debug("thrift", "Transaction queued")
}

func (thrift *thriftPlugin) ReceivedFin(tcptuple *common.TCPTuple, dir uint8,
	private protos.ProtocolData) protos.ProtocolData {

	trans := thrift.getTransaction(tcptuple.Hashable())
	if trans != nil {
		if trans.request != nil && trans.reply == nil {
			logp.Debug("thrift", "FIN and had only one transaction. Assuming one way")
			thrift.publishQueue <- trans
			thrift.transactions.Delete(trans.tuple.Hashable())
		}
	}

	return private
}

func (thrift *thriftPlugin) GapInStream(tcptuple *common.TCPTuple, dir uint8,
	nbytes int, private protos.ProtocolData) (priv protos.ProtocolData, drop bool) {

	defer logp.Recover("GapInStream(thrift) exception")
	logp.Debug("thriftdetailed", "GapInStream called")

	if private == nil {
		return private, false
	}
	thriftData, ok := private.(thriftPrivateData)
	if !ok {
		return private, false
	}
	stream := thriftData.data[dir]
	if stream == nil || stream.message == nil {
		// nothing to do
		return private, false
	}

	if thrift.messageGap(stream, nbytes) {
		// we need to publish from here
		thrift.messageComplete(tcptuple, dir, stream, &thriftData)
	}

	// we always drop the TCP stream. Because it's binary and len based,
	// there are too few cases in which we could recover the stream (maybe
	// for very large blobs, leaving that as TODO)
	return private, true
}

func (thrift *thriftPlugin) publishTransactions() {
	for t := range thrift.publishQueue {
		evt, pbf := pb.NewBeatEvent(t.ts)
		pbf.SetSource(&t.src)
		pbf.SetDestination(&t.dst)
		pbf.Source.Bytes = int64(t.bytesIn)
		pbf.Destination.Bytes = int64(t.bytesOut)
		pbf.Event.Dataset = "thrift"
		pbf.Network.Transport = "tcp"
		pbf.Network.Protocol = pbf.Event.Dataset

		var status string
		if t.reply != nil && t.reply.hasException {
			status = common.ERROR_STATUS
		} else {
			status = common.OK_STATUS
		}

		fields := evt.Fields
		fields["type"] = pbf.Event.Dataset
		fields["status"] = status
		thriftFields := common.MapStr{}
		fields["thrift"] = thriftFields

		if t.request != nil {
			fields["method"] = t.request.method
			fields["path"] = t.request.service
			query := t.request.method + t.request.params
			fields["query"] = query
			pbf.Event.Start = t.request.ts

			thriftFields["params"] = t.request.params
			if len(t.request.service) > 0 {
				thriftFields["service"] = t.request.service
			}

			if thrift.sendRequest {
				fields["request"] = query
			}
		}

		if t.reply != nil {
			pbf.Event.End = t.reply.ts

			thriftFields["return_value"] = t.reply.returnValue
			if len(t.reply.exceptions) > 0 {
				thriftFields["exceptions"] = t.reply.exceptions
			}

			if thrift.sendResponse {
				if !t.reply.hasException {
					fields["response"] = t.reply.returnValue
				} else {
					fields["response"] = fmt.Sprintf("Exceptions: %s",
						t.reply.exceptions)
				}
			}

			pbf.Error.Message = t.reply.notes
		}

		if thrift.results != nil {
			thrift.results(evt)
		}

		logp.Debug("thrift", "Published event")
	}
}
