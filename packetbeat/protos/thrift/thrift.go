package thrift

import (
	"encoding/binary"
	"encoding/hex"
	"expvar"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/elastic/beats/packetbeat/procs"
	"github.com/elastic/beats/packetbeat/protos"
	"github.com/elastic/beats/packetbeat/protos/tcp"
	"github.com/elastic/beats/packetbeat/publish"
)

type ThriftMessage struct {
	Ts time.Time

	TcpTuple     common.TcpTuple
	CmdlineTuple *common.CmdlineTuple
	Direction    uint8

	start int

	fields []ThriftField

	IsRequest    bool
	HasException bool
	Version      uint32
	Type         uint32
	Method       string
	SeqId        uint32
	Params       string
	ReturnValue  string
	Exceptions   string
	FrameSize    uint32
	Service      string
	Notes        []string
}

type ThriftField struct {
	Type byte
	Id   uint16

	Value string
}

type ThriftStream struct {
	tcptuple *common.TcpTuple

	data []byte

	parseOffset int
	parseState  int

	// when this is set, don't care about the
	// traffic in this direction. Used to skip large responses.
	skipInput bool

	message *ThriftMessage
}

type ThriftTransaction struct {
	Type         string
	tuple        common.TcpTuple
	Src          common.Endpoint
	Dst          common.Endpoint
	ResponseTime int32
	Ts           int64
	JsTs         time.Time
	ts           time.Time
	cmdline      *common.CmdlineTuple
	BytesIn      uint64
	BytesOut     uint64

	Request *ThriftMessage
	Reply   *ThriftMessage
}

const (
	ThriftStartState = iota
	ThriftFieldState
)

const (
	ThriftVersionMask = 0xffff0000
	ThriftVersion1    = 0x80010000
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
	ThriftTBinary  = 1
	ThriftTCompact = 2
)

// Thrift transport types
const (
	ThriftTSocket = 1
	ThriftTFramed = 2
)

type Thrift struct {

	// config
	Ports                  []int
	StringMaxSize          int
	CollectionMaxSize      int
	DropAfterNStructFields int
	CaptureReply           bool
	ObfuscateStrings       bool
	Send_request           bool
	Send_response          bool

	TransportType byte
	ProtocolType  byte

	transactions       *common.Cache
	transactionTimeout time.Duration

	PublishQueue chan *ThriftTransaction
	results      publish.Transactions
	Idl          *ThriftIdl
}

var (
	unmatchedRequests  = expvar.NewInt("thrift.unmatched_requests")
	unmatchedResponses = expvar.NewInt("thrift.unmatched_responses")
)

func init() {
	protos.Register("thrift", New)
}

func New(
	testMode bool,
	results publish.Transactions,
	cfg *common.Config,
) (protos.Plugin, error) {
	p := &Thrift{}
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

func (thrift *Thrift) init(
	testMode bool,
	results publish.Transactions,
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
		thrift.PublishQueue = make(chan *ThriftTransaction, 1000)
		thrift.results = results
		go thrift.publishTransactions()
	}

	return nil
}

func (thrift *Thrift) getTransaction(k common.HashableTcpTuple) *ThriftTransaction {
	v := thrift.transactions.Get(k)
	if v != nil {
		return v.(*ThriftTransaction)
	}
	return nil
}

func (thrift *Thrift) InitDefaults() {
	// defaults
	thrift.StringMaxSize = 200
	thrift.CollectionMaxSize = 15
	thrift.DropAfterNStructFields = 500
	thrift.TransportType = ThriftTSocket
	thrift.ProtocolType = ThriftTBinary
	thrift.CaptureReply = true
	thrift.ObfuscateStrings = false
	thrift.Send_request = false
	thrift.Send_response = false
	thrift.transactionTimeout = protos.DefaultTransactionExpiration
}

func (thrift *Thrift) readConfig(config *thriftConfig) error {
	var err error

	thrift.Ports = config.Ports
	thrift.Send_request = config.SendRequest
	thrift.Send_response = config.SendResponse

	thrift.StringMaxSize = config.StringMaxSize
	thrift.CollectionMaxSize = config.CollectionMaxSize
	thrift.DropAfterNStructFields = config.DropAfterNStructFields
	thrift.CaptureReply = config.CaptureReply
	thrift.ObfuscateStrings = config.ObfuscateStrings

	switch config.TransportType {
	case "socket":
		thrift.TransportType = ThriftTSocket
	case "framed":
		thrift.TransportType = ThriftTFramed
	default:
		return fmt.Errorf("Transport type `%s` not known", config.TransportType)
	}

	switch config.ProtocolType {
	case "binary":
		thrift.ProtocolType = ThriftTBinary
	default:
		return fmt.Errorf("Protocol type `%s` not known", config.ProtocolType)
	}

	if len(config.IdlFiles) > 0 {
		thrift.Idl, err = NewThriftIdl(config.IdlFiles)
		if err != nil {
			return err
		}
	}

	return nil
}

func (thrift *Thrift) GetPorts() []int {
	return thrift.Ports
}

func (m *ThriftMessage) String() string {
	return fmt.Sprintf("IsRequest: %t Type: %d Method: %s SeqId: %d Params: %s ReturnValue: %s Exceptions: %s",
		m.IsRequest, m.Type, m.Method, m.SeqId, m.Params, m.ReturnValue, m.Exceptions)
}

func (thrift *Thrift) readMessageBegin(s *ThriftStream) (bool, bool) {
	var ok, complete bool
	var offset, off int

	m := s.message

	if len(s.data[s.parseOffset:]) < 9 {
		return true, false // ok, not complete
	}

	sz := common.Bytes_Ntohl(s.data[s.parseOffset : s.parseOffset+4])
	if int32(sz) < 0 {
		m.Version = sz & ThriftVersionMask
		if m.Version != ThriftVersion1 {
			logp.Debug("thrift", "Unexpected version: %d", m.Version)
		}

		logp.Debug("thriftdetailed", "version = %d", m.Version)

		offset = s.parseOffset + 4

		logp.Debug("thriftdetailed", "offset = %d", offset)

		m.Type = sz & ThriftTypeMask
		m.Method, ok, complete, off = thrift.readString(s.data[offset:])
		if !ok {
			return false, false // not ok, not complete
		}
		if !complete {
			logp.Debug("thriftdetailed", "Method name not complete")
			return true, false // ok, not complete
		}
		offset += off

		logp.Debug("thriftdetailed", "method = %s", m.Method)
		logp.Debug("thriftdetailed", "offset = %d", offset)

		if len(s.data[offset:]) < 4 {
			logp.Debug("thriftdetailed", "Less then 4 bytes remaining")
			return true, false // ok, not complete
		}
		m.SeqId = common.Bytes_Ntohl(s.data[offset : offset+4])
		s.parseOffset = offset + 4
	} else {
		// no version mode
		offset = s.parseOffset

		m.Method, ok, complete, off = thrift.readString(s.data[offset:])
		if !ok {
			return false, false // not ok, not complete
		}
		if !complete {
			logp.Debug("thriftdetailed", "Method name not complete")
			return true, false // ok, not complete
		}
		offset += off

		logp.Debug("thriftdetailed", "method = %s", m.Method)
		logp.Debug("thriftdetailed", "offset = %d", offset)

		if len(s.data[offset:]) < 5 {
			return true, false // ok, not complete
		}

		m.Type = uint32(s.data[offset])
		offset += 1
		m.SeqId = common.Bytes_Ntohl(s.data[offset : offset+4])
		s.parseOffset = offset + 4
	}

	if m.Type == ThriftMsgTypeCall || m.Type == ThriftMsgTypeOneway {
		m.IsRequest = true
	} else {
		m.IsRequest = false
	}

	return true, true
}

// Functions to decode simple types
// They all have the same signature, returning the string value and the
// number of bytes consumed (off).
type ThriftFieldReader func(data []byte) (value string, ok bool, complete bool, off int)

// thriftReadString caps the returned value to ThriftStringMaxSize but returns the
// off to the end of it.
func (thrift *Thrift) readString(data []byte) (value string, ok bool, complete bool, off int) {
	if len(data) < 4 {
		return "", true, false, 0 // ok, not complete
	}
	sz := int(common.Bytes_Ntohl(data[:4]))
	if int32(sz) < 0 {
		return "", false, false, 0 // not ok
	}
	if len(data[4:]) < sz {
		return "", true, false, 0 // ok, not complete
	}

	if sz > thrift.StringMaxSize {
		value = string(data[4 : 4+thrift.StringMaxSize])
		value += "..."
	} else {
		value = string(data[4 : 4+sz])
	}
	off = 4 + sz

	return value, true, true, off // all good
}

func (thrift *Thrift) readAndQuoteString(data []byte) (value string, ok bool, complete bool, off int) {
	value, ok, complete, off = thrift.readString(data)
	if value == "" {
		value = `""`
	} else if thrift.ObfuscateStrings {
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

func (thrift *Thrift) readBool(data []byte) (value string, ok bool, complete bool, off int) {
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

func (thrift *Thrift) readByte(data []byte) (value string, ok bool, complete bool, off int) {
	if len(data) < 1 {
		return "", true, false, 0
	}
	value = strconv.Itoa(int(data[0]))

	return value, true, true, 1
}

func (thrift *Thrift) readDouble(data []byte) (value string, ok bool, complete bool, off int) {
	if len(data) < 8 {
		return "", true, false, 0
	}

	bits := binary.BigEndian.Uint64(data[:8])
	double := math.Float64frombits(bits)
	value = strconv.FormatFloat(double, 'f', -1, 64)

	return value, true, true, 8
}

func (thrift *Thrift) readI16(data []byte) (value string, ok bool, complete bool, off int) {
	if len(data) < 2 {
		return "", true, false, 0
	}
	i16 := common.Bytes_Ntohs(data[:2])
	value = strconv.Itoa(int(i16))

	return value, true, true, 2
}

func (thrift *Thrift) readI32(data []byte) (value string, ok bool, complete bool, off int) {
	if len(data) < 4 {
		return "", true, false, 0
	}
	i32 := common.Bytes_Ntohl(data[:4])
	value = strconv.Itoa(int(i32))

	return value, true, true, 4
}

func (thrift *Thrift) readI64(data []byte) (value string, ok bool, complete bool, off int) {
	if len(data) < 8 {
		return "", true, false, 0
	}
	i64 := common.Bytes_Ntohll(data[:8])
	value = strconv.FormatInt(int64(i64), 10)

	return value, true, true, 8
}

// Common implementation for lists and sets (they share the same binary repr).
func (thrift *Thrift) readListOrSet(data []byte) (value string, ok bool, complete bool, off int) {
	if len(data) < 5 {
		return "", true, false, 0
	}
	type_ := data[0]

	funcReader, typeFound := thrift.funcReadersByType(type_)
	if !typeFound {
		logp.Debug("thrift", "Field type %d not known", type_)
		return "", false, false, 0
	}

	sz := int(common.Bytes_Ntohl(data[1:5]))
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

		if i < thrift.CollectionMaxSize {
			fields = append(fields, value)
		} else if i == thrift.CollectionMaxSize {
			fields = append(fields, "...")
		}
		offset += bytesRead
	}

	return strings.Join(fields, ", "), true, true, offset
}

func (thrift *Thrift) readSet(data []byte) (value string, ok bool, complete bool, off int) {
	value, ok, complete, off = thrift.readListOrSet(data)
	if value != "" {
		value = "{" + value + "}"
	}
	return value, ok, complete, off
}

func (thrift *Thrift) readList(data []byte) (value string, ok bool, complete bool, off int) {
	value, ok, complete, off = thrift.readListOrSet(data)
	if value != "" {
		value = "[" + value + "]"
	}
	return value, ok, complete, off
}

func (thrift *Thrift) readMap(data []byte) (value string, ok bool, complete bool, off int) {
	if len(data) < 6 {
		return "", true, false, 0
	}
	type_key := data[0]
	type_value := data[1]

	funcReaderKey, typeFound := thrift.funcReadersByType(type_key)
	if !typeFound {
		logp.Debug("thrift", "Field type %d not known", type_key)
		return "", false, false, 0
	}

	funcReaderValue, typeFound := thrift.funcReadersByType(type_value)
	if !typeFound {
		logp.Debug("thrift", "Field type %d not known", type_value)
		return "", false, false, 0
	}

	sz := int(common.Bytes_Ntohl(data[2:6]))
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

		if i < thrift.CollectionMaxSize {
			fields = append(fields, key+": "+value)
		} else if i == thrift.CollectionMaxSize {
			fields = append(fields, "...")
		}
	}

	return "{" + strings.Join(fields, ", ") + "}", true, true, offset
}

func (thrift *Thrift) readStruct(data []byte) (value string, ok bool, complete bool, off int) {

	var bytesRead int
	offset := 0
	fields := []ThriftField{}

	// Loop until hitting a STOP or reaching the maximum number of elements
	// we follow in a stream (at which point, we assume we interpreted something
	// wrong).
	for i := 0; ; i++ {
		var field ThriftField

		if i >= thrift.DropAfterNStructFields {
			logp.Debug("thrift", "Too many fields in struct. Dropping as error")
			return "", false, false, 0
		}

		if len(data) < 1 {
			return "", true, false, 0
		}

		field.Type = byte(data[offset])
		offset += 1
		if field.Type == ThriftTypeStop {
			return thrift.formatStruct(fields, false, []*string{}), true, true, offset
		}

		if len(data[offset:]) < 2 {
			return "", true, false, 0 // not complete
		}

		field.Id = common.Bytes_Ntohs(data[offset : offset+2])
		offset += 2

		funcReader, typeFound := thrift.funcReadersByType(field.Type)
		if !typeFound {
			logp.Debug("thrift", "Field type %d not known", field.Type)
			return "", false, false, 0
		}

		field.Value, ok, complete, bytesRead = funcReader(data[offset:])

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

func (thrift *Thrift) formatStruct(fields []ThriftField, resolve_names bool,
	fieldnames []*string) string {

	toJoin := []string{}
	for i, field := range fields {
		if i == thrift.CollectionMaxSize {
			toJoin = append(toJoin, "...")
			break
		}
		if resolve_names && int(field.Id) < len(fieldnames) && fieldnames[field.Id] != nil {
			toJoin = append(toJoin, *fieldnames[field.Id]+": "+field.Value)
		} else {
			toJoin = append(toJoin, strconv.Itoa(int(field.Id))+": "+field.Value)
		}
	}
	return "(" + strings.Join(toJoin, ", ") + ")"
}

// Dictionary wrapped in a function to avoid "initialization loop"
func (thrift *Thrift) funcReadersByType(type_ byte) (func_ ThriftFieldReader, exists bool) {
	switch type_ {
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

func (thrift *Thrift) readField(s *ThriftStream) (ok bool, complete bool, field *ThriftField) {

	var off int

	field = new(ThriftField)

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
	field.Id = common.Bytes_Ntohs(s.data[offset : offset+2])
	offset += 2

	funcReader, typeFound := thrift.funcReadersByType(field.Type)
	if !typeFound {
		logp.Debug("thrift", "Field type %d not known", field.Type)
		return false, false, nil
	}

	field.Value, ok, complete, off = funcReader(s.data[offset:])

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

func (thrift *Thrift) messageParser(s *ThriftStream) (bool, bool) {
	var ok, complete bool
	var m = s.message

	logp.Debug("thriftdetailed", "messageParser called parseState=%v offset=%v",
		s.parseState, s.parseOffset)

	for s.parseOffset < len(s.data) {
		switch s.parseState {
		case ThriftStartState:
			m.start = s.parseOffset
			if thrift.TransportType == ThriftTFramed {
				// read I32
				if len(s.data) < 4 {
					return true, false
				}
				m.FrameSize = common.Bytes_Ntohl(s.data[:4])
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

			if !m.IsRequest && !thrift.CaptureReply {
				// don't actually read the result
				logp.Debug("thrift", "Don't capture reply")
				m.ReturnValue = ""
				m.Exceptions = ""
				return true, true
			}
			s.parseState = ThriftFieldState
		case ThriftFieldState:
			ok, complete, field := thrift.readField(s)
			logp.Debug("thriftdetailed", "readField returned: %v %v", ok, complete)
			if !ok {
				return false, false
			}
			if complete {
				// done
				var method *ThriftIdlMethod = nil
				if thrift.Idl != nil {
					method = thrift.Idl.FindMethod(m.Method)
				}
				if m.IsRequest {
					if method != nil {
						m.Params = thrift.formatStruct(m.fields, true, method.Params)

						m.Service = method.Service.Name
					} else {
						m.Params = thrift.formatStruct(m.fields, false, nil)
					}
				} else {
					if len(m.fields) > 1 {
						logp.Warn("Thrift RPC response with more than field. Ignoring all but first")
					}
					if len(m.fields) > 0 {
						field := m.fields[0]
						if field.Id == 0 {
							m.ReturnValue = field.Value
							m.Exceptions = ""
						} else {
							m.ReturnValue = ""
							if method != nil {
								m.Exceptions = thrift.formatStruct(m.fields, true, method.Exceptions)
							} else {
								m.Exceptions = thrift.formatStruct(m.fields, false, nil)
							}
							m.HasException = true
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
func (thrift *Thrift) messageGap(s *ThriftStream, nbytes int) (complete bool) {
	m := s.message
	switch s.parseState {
	case ThriftStartState:
		// not enough data yet to be useful
		return false
	case ThriftFieldState:
		if !m.IsRequest {
			// large response case, can tolerate loss
			m.Notes = append(m.Notes, "Packet loss while capturing the response")
			return true
		}
	}

	return false
}

func (stream *ThriftStream) PrepareForNewMessage(flush bool) {
	if flush {
		stream.data = []byte{}
	} else {
		stream.data = stream.data[stream.parseOffset:]
	}
	//logp.Debug("thrift", "remaining data: [%s]", stream.data)
	stream.parseOffset = 0
	stream.message = nil
	stream.parseState = ThriftStartState
}

type thriftPrivateData struct {
	Data [2]*ThriftStream
}

func (thrift *Thrift) messageComplete(tcptuple *common.TcpTuple, dir uint8,
	stream *ThriftStream, priv *thriftPrivateData) {

	var flush bool = false

	if stream.message.IsRequest {
		logp.Debug("thrift", "Thrift request message: %s", stream.message.Method)
		if !thrift.CaptureReply {
			// enable the stream in the other direction to get the reply
			stream_rev := priv.Data[1-dir]
			if stream_rev != nil {
				stream_rev.skipInput = false
			}
		}
	} else {
		logp.Debug("thrift", "Thrift response message: %s", stream.message.Method)
		if !thrift.CaptureReply {
			// disable stream in this direction
			stream.skipInput = true

			// and flush current data
			flush = true
		}
	}

	// all ok, go to next level
	stream.message.TcpTuple = *tcptuple
	stream.message.Direction = dir
	stream.message.CmdlineTuple = procs.ProcWatcher.FindProcessesTuple(tcptuple.IpPort())
	if stream.message.FrameSize == 0 {
		stream.message.FrameSize = uint32(stream.parseOffset - stream.message.start)
	}
	thrift.handleThrift(stream.message)

	// and reset message
	stream.PrepareForNewMessage(flush)

}

func (thrift *Thrift) ConnectionTimeout() time.Duration {
	return thrift.transactionTimeout
}

func (thrift *Thrift) Parse(pkt *protos.Packet, tcptuple *common.TcpTuple, dir uint8,
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

	stream := priv.Data[dir]

	if stream == nil {
		stream = &ThriftStream{
			tcptuple: tcptuple,
			data:     pkt.Payload,
			message:  &ThriftMessage{Ts: pkt.Ts},
		}
		priv.Data[dir] = stream
	} else {
		if stream.skipInput {
			// stream currently suspended in this direction
			return priv
		}
		// concatenate bytes
		stream.data = append(stream.data, pkt.Payload...)
		if len(stream.data) > tcp.TCP_MAX_DATA_IN_STREAM {
			logp.Debug("thrift", "Stream data too large, dropping TCP stream")
			priv.Data[dir] = nil
			return priv
		}
	}

	for len(stream.data) > 0 {
		if stream.message == nil {
			stream.message = &ThriftMessage{Ts: pkt.Ts}
		}

		ok, complete := thrift.messageParser(priv.Data[dir])
		logp.Debug("thriftdetailed", "messageParser returned %v %v", ok, complete)

		if !ok {
			// drop this tcp stream. Will retry parsing with the next
			// segment in it
			priv.Data[dir] = nil
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

func (thrift *Thrift) handleThrift(msg *ThriftMessage) {
	if msg.IsRequest {
		thrift.receivedRequest(msg)
	} else {
		thrift.receivedReply(msg)
	}
}

func (thrift *Thrift) receivedRequest(msg *ThriftMessage) {
	tuple := msg.TcpTuple

	trans := thrift.getTransaction(tuple.Hashable())
	if trans != nil {
		logp.Debug("thrift", "Two requests without reply, assuming the old one is oneway")
		unmatchedRequests.Add(1)
		thrift.PublishQueue <- trans
	}

	trans = &ThriftTransaction{
		Type:  "thrift",
		tuple: tuple,
	}
	thrift.transactions.Put(tuple.Hashable(), trans)

	trans.ts = msg.Ts
	trans.Ts = int64(trans.ts.UnixNano() / 1000)
	trans.JsTs = msg.Ts
	trans.Src = common.Endpoint{
		Ip:   msg.TcpTuple.Src_ip.String(),
		Port: msg.TcpTuple.Src_port,
		Proc: string(msg.CmdlineTuple.Src),
	}
	trans.Dst = common.Endpoint{
		Ip:   msg.TcpTuple.Dst_ip.String(),
		Port: msg.TcpTuple.Dst_port,
		Proc: string(msg.CmdlineTuple.Dst),
	}
	if msg.Direction == tcp.TcpDirectionReverse {
		trans.Src, trans.Dst = trans.Dst, trans.Src
	}

	trans.Request = msg
	trans.BytesIn = uint64(msg.FrameSize)
}

func (thrift *Thrift) receivedReply(msg *ThriftMessage) {

	// we need to search the request first.
	tuple := msg.TcpTuple

	trans := thrift.getTransaction(tuple.Hashable())
	if trans == nil {
		logp.Debug("thrift", "Response from unknown transaction. Ignoring: %v", tuple)
		unmatchedResponses.Add(1)
		return
	}

	if trans.Request.Method != msg.Method {
		logp.Debug("thrift", "Response from another request received '%s' '%s'"+
			". Ignoring.", trans.Request.Method, msg.Method)
		unmatchedResponses.Add(1)
		return
	}

	trans.Reply = msg
	trans.BytesOut = uint64(msg.FrameSize)

	trans.ResponseTime = int32(msg.Ts.Sub(trans.ts).Nanoseconds() / 1e6) // resp_time in milliseconds

	thrift.PublishQueue <- trans
	thrift.transactions.Delete(tuple.Hashable())

	logp.Debug("thrift", "Transaction queued")
}

func (thrift *Thrift) ReceivedFin(tcptuple *common.TcpTuple, dir uint8,
	private protos.ProtocolData) protos.ProtocolData {

	trans := thrift.getTransaction(tcptuple.Hashable())
	if trans != nil {
		if trans.Request != nil && trans.Reply == nil {
			logp.Debug("thrift", "FIN and had only one transaction. Assuming one way")
			thrift.PublishQueue <- trans
			thrift.transactions.Delete(trans.tuple.Hashable())
		}
	}

	return private
}

func (thrift *Thrift) GapInStream(tcptuple *common.TcpTuple, dir uint8,
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
	stream := thriftData.Data[dir]
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

func (thrift *Thrift) publishTransactions() {
	for t := range thrift.PublishQueue {
		event := common.MapStr{}

		event["type"] = "thrift"
		if t.Reply != nil && t.Reply.HasException {
			event["status"] = common.ERROR_STATUS
		} else {
			event["status"] = common.OK_STATUS
		}
		event["responsetime"] = t.ResponseTime
		thriftmap := common.MapStr{}

		if t.Request != nil {
			event["method"] = t.Request.Method
			event["path"] = t.Request.Service
			event["query"] = fmt.Sprintf("%s%s", t.Request.Method, t.Request.Params)
			event["bytes_in"] = t.BytesIn
			event["bytes_out"] = t.BytesOut
			thriftmap = common.MapStr{
				"params": t.Request.Params,
			}
			if len(t.Request.Service) > 0 {
				thriftmap["service"] = t.Request.Service
			}

			if thrift.Send_request {
				event["request"] = fmt.Sprintf("%s%s", t.Request.Method,
					t.Request.Params)
			}
		}

		if t.Reply != nil {
			thriftmap["return_value"] = t.Reply.ReturnValue
			if len(t.Reply.Exceptions) > 0 {
				thriftmap["exceptions"] = t.Reply.Exceptions
			}
			event["bytes_out"] = uint64(t.Reply.FrameSize)

			if thrift.Send_response {
				if !t.Reply.HasException {
					event["response"] = t.Reply.ReturnValue
				} else {
					event["response"] = fmt.Sprintf("Exceptions: %s",
						t.Reply.Exceptions)
				}
			}
			if len(t.Reply.Notes) > 0 {
				event["notes"] = t.Reply.Notes
			}
		} else {
			event["bytes_out"] = 0
		}
		event["thrift"] = thriftmap

		event["@timestamp"] = common.Time(t.ts)
		event["src"] = &t.Src
		event["dst"] = &t.Dst

		if thrift.results != nil {
			thrift.results.PublishTransaction(event)
		}

		logp.Debug("thrift", "Published event")
	}
}
