package tds

import (
	"errors"
	"expvar"
	"strings"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/elastic/beats/packetbeat/procs"
	"github.com/elastic/beats/packetbeat/protos"
	"github.com/elastic/beats/packetbeat/protos/tcp"
	"github.com/elastic/beats/packetbeat/publish"
)

// Packet types
const (
	TDS_7_Query         = 0x01
	TDS_RPC             = 0x03
	TDS_Server_Response = 0x04
	TDS_Cancels         = 0x06
	TDS_BulkCopy        = 0x07
	TDS_7_Login         = 0x10
	TDS_7_Auth          = 0x11
	TDS_8_Prelogin      = 0x12
)

// Types
const (
	SYBVOID       = 0x1F
	SYBIMAGE      = 0x22
	SYBTEXT       = 0x23
	SYBUNIQUE     = 0x24
	SYBVARBINARY  = 0x25
	SYBINTN       = 0x26
	SYBVARCHAR    = 0x27
	SYBBINARY     = 0x2D
	SYBCHAR       = 0x2F
	SYBINT1       = 0x30
	SYBBIT        = 0x32
	SYBINT2       = 0x34
	SYBINT4       = 0x38
	SYBDATETIME4  = 0x3A
	SYBREAL       = 0x3B
	SYBMONEY      = 0x3C
	SYBDATETIME   = 0x3D
	SYBFLT8       = 0x3E
	SYBSINT1      = 0x40
	SYBUINT2      = 0x41
	SYBUINT4      = 0x42
	SYBUINT8      = 0x43
	SYBVARIANT    = 0x62
	SYBNTEXT      = 0x63
	SYBNVARCHAR   = 0x67
	SYBBITN       = 0x68
	SYBDECIMAL    = 0x6A
	SYBNUMERIC    = 0x6C
	SYBFLTN       = 0x6D
	SYBMONEYN     = 0x6E
	SYBDATETIMN   = 0x6F
	SYBMONEY4     = 0x7A
	SYBINT8       = 0x7F
	XSYBVARBINARY = 0xA5
	XSYBVARCHAR   = 0xA7
	XSYBBINARY    = 0xAD
	XSYBCHAR      = 0xAF
	SYBLONGBINARY = 0xE1
	XSYBNVARCHAR  = 0xE7
	XSYBNCHAR     = 0xEF
)

const maxPayloadSize = 100 * 1024

var (
	unmatchedRequests  = expvar.NewInt("tds.unmatched_requests")
	unmatchedResponses = expvar.NewInt("tds.unmatched_responses")
)

type tdsMessage struct {
	start int
	end   int

	ts             time.Time
	isRequest      bool
	packetLength   uint32
	seq            uint8
	typ            uint8
	numberOfRows   int
	numberOfFields int
	size           uint64
	fields         []string
	rows           [][]string
	tables         string
	isOK           bool
	affectedRows   uint64
	insertID       uint64
	isError        bool
	errorCode      uint16
	errorInfo      string
	query          string
	ignoreMessage  bool
	isLastPacket   bool
	version        string
	client_version string
	clientPID      uint32
	flags_1        uint8
	flags_2        uint8
	flags_3        uint8

	hostname_length      uint16
	username_length      uint16
	password_length      uint16
	app_name_length      uint16
	server_name_length   uint16
	library_length       uint16
	language_length      uint16
	database_name_length uint16
	auth_length          uint16
	filename_length      uint16
	new_password_length  uint16

	hostname_pos      uint16
	username_pos      uint16
	password_pos      uint16
	app_name_pos      uint16
	server_name_pos   uint16
	library_pos       uint16
	language_pos      uint16
	database_name_pos uint16
	auth_pos          uint16
	filename_pos      uint16
	new_password_pos  uint16

	MAC_address []uint8

	hostname      string
	username      string
	password      string
	app_name      string
	server_name   string
	library_name  string
	language_name string
	database_name string
	filename      string
	new_password  string

	direction    uint8
	isTruncated  bool
	tcpTuple     common.TCPTuple
	cmdlineTuple *common.CmdlineTuple
	raw          []byte
	notes        []string
}

type tdsTransaction struct {
	tuple        common.TCPTuple
	src          common.Endpoint
	dst          common.Endpoint
	responseTime int32
	ts           time.Time
	query        string
	method       string
	path         string // for tds, Path refers to the tds table queried
	bytesOut     uint64
	bytesIn      uint64
	notes        []string

	tds common.MapStr

	requestRaw  string
	responseRaw string

	username string
	hostname string
	server   string
	version  string
}

type tdsStream struct {
	data []byte

	parseOffset int
	isClient    bool

	message *tdsMessage
}

type tdsPlugin struct {

	// config
	ports        []int
	maxStoreRows int
	maxRowLength int
	sendRequest  bool
	sendResponse bool

	transactions       *common.Cache
	transactionTimeout time.Duration

	results publish.Transactions

	// function pointer for mocking
	handleTds func(tds *tdsPlugin, m *tdsMessage, tcp *common.TCPTuple,
		dir uint8, raw_msg []byte)
}

func init() {
	protos.Register("tds", New)
}

func New(
	testMode bool,
	results publish.Transactions,
	cfg *common.Config,
) (protos.Plugin, error) {
	p := &tdsPlugin{}
	config := defaultConfig
	if !testMode {
		if err := cfg.Unpack(&config); err != nil {
			return nil, err
		}
	}

	if err := p.init(results, &config); err != nil {
		return nil, err
	}
	return p, nil
}

func (tds *tdsPlugin) init(results publish.Transactions, config *tdsConfig) error {
	tds.setFromConfig(config)

	tds.transactions = common.NewCache(
		tds.transactionTimeout,
		protos.DefaultTransactionHashSize)
	tds.transactions.StartJanitor(tds.transactionTimeout)
	tds.handleTds = handleTds
	tds.results = results

	return nil
}

func (tds *tdsPlugin) setFromConfig(config *tdsConfig) {
	tds.ports = config.Ports
	tds.maxRowLength = config.MaxRowLength
	tds.maxStoreRows = config.MaxRows
	tds.sendRequest = config.SendRequest
	tds.sendResponse = config.SendResponse
	tds.transactionTimeout = config.TransactionTimeout
}

func (tds *tdsPlugin) getTransaction(k common.HashableTCPTuple) *tdsTransaction {
	v := tds.transactions.Get(k)
	if v != nil {
		return v.(*tdsTransaction)
	}
	return nil
}

func (tds *tdsPlugin) GetPorts() []int {
	return tds.ports
}

func (stream *tdsStream) prepareForNewMessage() {
	stream.data = stream.data[stream.parseOffset:]
	stream.parseOffset = 0
	stream.isClient = false
	stream.message = nil
}

func tdsMessageParser(s *tdsStream) (bool, bool) {

	m := s.message
	for s.parseOffset < len(s.data) {
		m.start = s.parseOffset
		if len(s.data[s.parseOffset:]) < 8 {
			logp.Warn("TDS Message too short. Ignore it.")
			return false, false
		}
		hdr := s.data[s.parseOffset : s.parseOffset+8]
		m.typ = hdr[0]
		if hdr[1] == 0x00 {
			m.isLastPacket = false
		} else {
			m.isLastPacket = true
		}
		m.size = uint64(hdr[2]) + uint64(hdr[3])
		m.start = s.parseOffset + 8
		if m.typ == TDS_7_Query {
			// TDS Query
			m.isRequest = true
			q_len := len(s.data) - 8
			i := 0
			str := ""
			for i < q_len {
				str = str + string(s.data[s.parseOffset+8+i])
				i = i + 1
			}
			m.query = str
			m.end = int(len(m.query)) + 8
			m.size = uint64(m.end - m.start)
			s.parseOffset += int(m.end)

			return true, true

		} else if m.typ == TDS_RPC {
			// TDS RPC
			m.ignoreMessage = true
			s.parseOffset += int(len(s.data))
			m.end = s.parseOffset

			return true, true

		} else if m.typ == TDS_7_Login {
			// TDS Login
			m.size = uint64(s.data[m.start]) + uint64(s.data[m.start+1]) + uint64(s.data[m.start+2]) + uint64(s.data[m.start+4])
			m.isRequest = true
			m.query = "Login Packet"
			m.end = int(m.size)
			m.raw = s.data[s.parseOffset+8 : m.end]
			s.parseOffset += int(m.end)

			return true, true

		} else if m.typ == TDS_7_Auth {
			// TDS Authentication Packet
			q_len := len(s.data)
			i := m.start
			str := ""
			for i < q_len {
				str = str + string(s.data[i])
				i = i + 1
			}
			m.query = str
			s.parseOffset += q_len
			m.isRequest = false
			m.end = int(len(m.query)) + 8

			return true, true

		} else {
			// TDS Response
			m.isRequest = false
			m.raw = s.data[s.parseOffset+8:]
			if len(s.data) > 8 {
				if s.data[s.parseOffset+8] == 170 {
					m.isError = true
				} else {
					m.isError = false
				}

				if s.data[s.parseOffset+8] == 167 {
					m.ignoreMessage = true
				}
			}

			s.parseOffset += int(len(s.data))
			m.end = s.parseOffset

			return true, true
		}
	}
	return true, false
}

type tdsPrivateData struct {
	data [2]*tdsStream
}

// Called when the parser has identified a full message.
func (tds *tdsPlugin) messageComplete(tcptuple *common.TCPTuple, dir uint8, stream *tdsStream) {
	// all ok, ship it
	if stream == nil {
		return
	}
	msg := stream.data[stream.message.start:stream.message.end]

	if !stream.message.ignoreMessage {
		tds.handleTds(tds, stream.message, tcptuple, dir, msg)
	}

	// and reset message
	stream.prepareForNewMessage()
}

func (tds *tdsPlugin) ConnectionTimeout() time.Duration {
	return tds.transactionTimeout
}

func (tds *tdsPlugin) Parse(pkt *protos.Packet, tcptuple *common.TCPTuple,
	dir uint8, private protos.ProtocolData) protos.ProtocolData {
	defer logp.Recover("ParseTDS exception")

	priv := tdsPrivateData{}
	if private != nil {
		var ok bool
		priv, ok = private.(tdsPrivateData)
		if !ok {
			priv = tdsPrivateData{}
		}
	} else {
		logp.Debug("tds", "Private is nil")
	}

	if priv.data[dir] == nil {
		priv.data[dir] = &tdsStream{
			data:    pkt.Payload,
			message: &tdsMessage{ts: pkt.Ts},
		}
	} else {
		// concatenate bytes
		priv.data[dir].data = append(priv.data[dir].data, pkt.Payload...)
		if len(priv.data[dir].data) > tcp.TCPMaxDataInStream {
			logp.Debug("tds", "Stream data too large, dropping TCP stream")
			priv.data[dir] = nil
			return priv
		}
	}

	stream := priv.data[dir]
	for len(stream.data) > 0 {
		if stream.message == nil {
			stream.message = &tdsMessage{ts: pkt.Ts}
		}

		ok, complete := tdsMessageParser(priv.data[dir])
		logp.Debug("tdsdetailed", "tdsMessageParser returned ok=%b complete=%b", ok, complete)
		if !ok {
			// drop this tcp stream. Will retry parsing with the next
			// segment in it
			priv.data[dir] = nil
			logp.Debug("tds", "Ignore TDS message. Drop tcp stream. Try parsing with the next segment")
			return priv
		}

		if complete {
			tds.messageComplete(tcptuple, dir, stream)
		} else {
			// wait for more data
			break
		}
	}
	return priv
}

func (tds *tdsPlugin) GapInStream(tcptuple *common.TCPTuple, dir uint8,
	nbytes int, private protos.ProtocolData) (priv protos.ProtocolData, drop bool) {

	defer logp.Recover("GapInStream(tds) exception")

	if private == nil {
		return private, false
	}
	tdsData, ok := private.(tdsPrivateData)
	if !ok {
		return private, false
	}
	stream := tdsData.data[dir]
	if stream == nil {
		return private, false
	}

	// we need to publish from here
	tds.messageComplete(tcptuple, dir, stream)
	// we always drop the TCP stream. Because it's binary and len based,
	// there are too few cases in which we could recover the stream(maybe
	// for very large blobs, leaving that as TODO)
	return private, true
}

func (tds *tdsPlugin) ReceivedFin(tcptuple *common.TCPTuple, dir uint8,
	private protos.ProtocolData) protos.ProtocolData {

	// TODO: check if we have data pending and either drop it to free
	// memory or send it up the stack.
	return private
}

func handleTds(tds *tdsPlugin, m *tdsMessage, tcptuple *common.TCPTuple,
	dir uint8, rawMsg []byte) {

	m.tcpTuple = *tcptuple
	m.direction = dir
	m.cmdlineTuple = procs.ProcWatcher.FindProcessesTuple(tcptuple.IPPort())
	m.raw = rawMsg

	if m.isRequest {
		tds.receivedTDSRequest(m)
	} else {
		tds.receivedTDSResponse(m)
	}
}

func (tds *tdsPlugin) receivedTDSRequest(msg *tdsMessage) {
	tuple := msg.tcpTuple
	trans := tds.getTransaction(tuple.Hashable())
	if trans != nil {
		if trans.tds != nil {
			logp.Debug("tds", "Two requests without a Response. Dropping old request: %s", trans.tds)
			unmatchedRequests.Add(1)
		}
	} else {
		trans = &tdsTransaction{tuple: tuple}
		tds.transactions.Put(tuple.Hashable(), trans)
	}

	trans.ts = msg.ts
	trans.src = common.Endpoint{
		IP:   msg.tcpTuple.SrcIP.String(),
		Port: msg.tcpTuple.SrcPort,
		Proc: string(msg.cmdlineTuple.Src),
	}
	trans.dst = common.Endpoint{
		IP:   msg.tcpTuple.DstIP.String(),
		Port: msg.tcpTuple.DstPort,
		Proc: string(msg.cmdlineTuple.Dst),
	}
	if msg.direction == tcp.TCPDirectionReverse {
		trans.src, trans.dst = trans.dst, trans.src
	}

	// Extract the method, by simply taking the first word and
	// making it upper case.
	query := strings.Trim(msg.query, " \n\t")
	index := strings.IndexAny(query, " \n\t")
	var method string
	if index > 0 {
		method = strings.ToUpper(query[:index])
	} else {
		method = strings.ToUpper(query)
	}

	trans.query = query
	trans.method = method
	trans.tds = common.MapStr{}

	trans.notes = msg.notes

	// save Raw message
	trans.requestRaw = msg.query
	trans.bytesIn = msg.size
	tds.publishTransaction(trans)
}

func (tds *tdsPlugin) receivedTDSResponse(msg *tdsMessage) {

	trans := tds.getTransaction(msg.tcpTuple.Hashable())

	if trans == nil {
		return
	}
	if trans.tds == nil {
		return
	}
	// save json details
	trans.tds.Update(common.MapStr{
		"affected_rows": msg.affectedRows,
		"num_rows":      msg.numberOfRows,
		"num_fields":    msg.numberOfFields,
		"isError":       msg.isError,
		"error_code":    msg.errorCode,
		"error_message": msg.errorInfo,
	})
	trans.bytesOut = msg.size

	trans.responseTime = int32(msg.ts.Sub(trans.ts).Nanoseconds() / 1e6) // resp_time in milliseconds

	// save Raw message
	if len(msg.raw) > 0 {
		fields, rows := tds.parseTDSResponse(msg.raw)

		trans.responseRaw = common.DumpInCSVFormat(fields, rows)
	}

	trans.notes = append(trans.notes, msg.notes...)
	tds.publishTransaction(trans)
	tds.transactions.Delete(trans.tuple.Hashable())

	logp.Debug("tds", "TDS transaction completed: %s", trans.tds)
	logp.Debug("tds", "%s", trans.responseRaw)
}

func (tds *tdsPlugin) parseTDSResponse(data []byte) ([]string, [][]string) {
	length, err := readLength(data, 0)
	if err != nil {
		logp.Warn("Invalid response: %v", err)
		return []string{}, [][]string{}
	}
	if length < 1 {
		logp.Warn("Warning: Skipping empty Response")
		return []string{}, [][]string{}
	}

	fields := []string{}
	row := []string{}
	rows := [][]string{}

	if len(data) < 9 {
		logp.Warn("Invalid response: data less than 8 bytes")
		return []string{}, [][]string{}
	}
	parseOffset := 0

	for parseOffset < length {

		switch data[parseOffset] {

		case 113:
			// Logout Acknowledgement
			parseOffset += 2

		case 121:
			// Return Status
			// return_value:= data[1]
			parseOffset += 2

		case 124:
			// Process ID
			// process_number:= data[1]
			parseOffset += 2

		case 129:
			// TDS 7.0 + Result
			start_off := parseOffset + 1
			// col_num:= data[start_off]
			start_off += 8
			prim := data[start_off]
			start_off += int(prim) + 2

			// col:= int(data[start_off])
			// start_off += 1
			// fields = append(fields, string(data[start_off:start_off + 2 * col - 1]))

			// start_off += 2 * col

			// if int(data[start_off]) == 209{
			// start_off += 13
			start := start_off
			str := ""
			flag := 0
			for flag != 1 {
				if int(data[start_off]) == 253 || int(data[start_off]) == 254 || int(data[start_off]) == 255 {
					flag = 1
				} else {
					start_off += 1
				}
			}

			str = string(data[start : start_off-1])

			row = append(row, str)
			//}
			parseOffset = start_off

		case 160:
			// Column Name
			total_length := int(data[parseOffset+1])
			parseOffset += 2
			i := 0
			for i < total_length {
				length := int(data[parseOffset])
				col_name := data[parseOffset+1 : parseOffset+length+2]
				fields = append(fields, string(col_name[:length]))
				parseOffset += length + 2
				i += 1
			}

		case 161:
			// Column Format
			total_length := int(data[parseOffset+1])
			parseOffset += total_length + 3

		case 167:
		// Compute Names
		// TODO Documentation missing

		case 169:
			// Order by
			length := int(data[parseOffset+1])
			parseOffset += length + 2

		case 170:
			// TDS 7.0 + Error Message
			length := int(data[parseOffset+1])
			// msg_len:= data[parseOffset + 2]
			// err_state:= data[parseOffset + 3]
			err_level := data[parseOffset+4]

			if err_level > 10 {
				logp.Debug("tds", "Error while executing")
			} else {
				logp.Warn("SQL Message")
			}
			parseOffset += length + 3

		case 171:
			// Info Message
			length := int(data[parseOffset+1])
			text := data[parseOffset+2 : parseOffset+length+3]
			row_val := string(text[:length])
			row = append(row, row_val)
			parseOffset += length + 3

		case 229:
			// Extended error message
			msg_len := int(data[parseOffset+1])
			start_off := parseOffset + 1
			i := 1
			error_msg := ""
			server_name := ""
			process_name := ""

			for i <= int(msg_len) {
				error_msg = error_msg + string(data[start_off+i])
				i = i + 1
			}

			start_off = start_off + int(msg_len) + 1
			server_len := int(data[start_off])

			i = 1

			for i <= int(server_len) {
				server_name = server_name + string(data[start_off+i])
				i = i + 1
			}

			start_off = start_off + int(server_len) + 1
			proc_len := int(data[start_off])

			i = 1

			for i <= int(proc_len) {
				process_name = process_name + string(data[start_off+i])
				i = i + 1
			}

			start_off = start_off + int(proc_len) + 1
			line_num_71 := int(data[start_off])
			line_num_72 := int(data[start_off+1])

			logp.Debug("Error %v occured on line number: %d %d Server name: %v Process name: %v", error_msg, line_num_71, line_num_72, server_name, process_name)

			parseOffset += start_off + 2

		case 172:
			// output Parameters..doubts in columns
			length := int(data[parseOffset+1])
			parseOffset += length + 1

		case 173:
			// Login Acknowledgement
			length := int(data[parseOffset+1])
			// ack:= data[parseOffset + 2]
			parseOffset += length + 3

		case 209:
			// Row Result
			length := int(data[parseOffset+1])
			text := data[parseOffset+2 : parseOffset+2+length]
			row_val := string(text[:length])
			logp.Info(row_val)
			row = append(row, row_val)
			parseOffset += length + 2

		case 227:
			// Environment Change
			length := int(data[parseOffset+1])
			parseOffset += 2
			text := data[parseOffset : parseOffset+length]
			row_val := string(text[:length])
			row = append(row, row_val)
			parseOffset += length + 1

		case 228:
			// Unknown ???
			length := int(data[parseOffset+1])
			parseOffset += 5
			text := data[parseOffset : parseOffset+length]
			row_val := string(text[:length])
			row = append(row, row_val)
			parseOffset += length + 5

		case 237:
			// Authentication
			length := int(data[parseOffset+1])
			text := data[parseOffset+2 : parseOffset+2+length]
			row_val := string(text[:length])
			row = append(row, row_val)
			parseOffset += length + 2

		case 253:
			// Result Set Done
			parseOffset += len(data) + 1
			rows = append(rows, row)
			return fields, rows

		case 254:
			// Process Done
			parseOffset += len(data) + 1
			rows = append(rows, row)
			return fields, rows

		case 255:
			// Done Inside Process
			parseOffset += len(data) + 1
			rows = append(rows, row)
			return fields, rows

		default:
			parseOffset += len(data) + 1
			// return fields, rows
			break
		}
	}
	rows = append(rows, row)
	return fields, rows
}

func (tds *tdsPlugin) publishTransaction(t *tdsTransaction) {
	if tds.results == nil {
		return
	}

	logp.Debug("tds", "tds.results exists")

	event := common.MapStr{}

	event["type"] = "tds"

	if t.tds["isError"] == true {
		event["status"] = "ERROR"
	} else {
		event["status"] = "OK"
	}
	event["responsetime"] = t.responseTime
	event["request"] = t.requestRaw
	event["response"] = t.responseRaw
	event["method"] = t.method
	event["query"] = t.query
	event["tds"] = t.tds
	event["path"] = t.path
	event["bytes_out"] = t.bytesOut
	event["bytes_in"] = t.bytesIn

	if len(t.notes) > 0 {
		event["notes"] = t.notes
	}

	event["@timestamp"] = common.Time(t.ts)
	event["src"] = &t.src
	event["dst"] = &t.dst
	tds.results.PublishTransaction(event)

}

func readLength(data []byte, offset int) (int, error) {
	if len(data[offset:]) < 3 {
		return 0, errors.New("Data too small to contain a valid length")
	}
	length := uint32(data[offset]) |
		uint32(data[offset+1])<<8 |
		uint32(data[offset+2])<<16
	return int(length), nil
}
