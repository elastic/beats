package mysql

import (
	"errors"
	"expvar"
	"fmt"
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
	MYSQL_CMD_QUERY = 3
)

const MAX_PAYLOAD_SIZE = 100 * 1024

var (
	unmatchedRequests  = expvar.NewInt("mysql.unmatched_requests")
	unmatchedResponses = expvar.NewInt("mysql.unmatched_responses")
)

type MysqlMessage struct {
	start int
	end   int

	Ts             time.Time
	IsRequest      bool
	PacketLength   uint32
	Seq            uint8
	Typ            uint8
	NumberOfRows   int
	NumberOfFields int
	Size           uint64
	Fields         []string
	Rows           [][]string
	Tables         string
	IsOK           bool
	AffectedRows   uint64
	InsertId       uint64
	IsError        bool
	ErrorCode      uint16
	ErrorInfo      string
	Query          string
	IgnoreMessage  bool

	Direction    uint8
	IsTruncated  bool
	TcpTuple     common.TcpTuple
	CmdlineTuple *common.CmdlineTuple
	Raw          []byte
	Notes        []string
}

type MysqlTransaction struct {
	Type         string
	tuple        common.TcpTuple
	Src          common.Endpoint
	Dst          common.Endpoint
	ResponseTime int32
	Ts           int64
	JsTs         time.Time
	ts           time.Time
	Query        string
	Method       string
	Path         string // for mysql, Path refers to the mysql table queried
	BytesOut     uint64
	BytesIn      uint64
	Notes        []string

	Mysql common.MapStr

	Request_raw  string
	Response_raw string
}

type MysqlStream struct {
	tcptuple *common.TcpTuple

	data []byte

	parseOffset int
	parseState  parseState
	isClient    bool

	message *MysqlMessage
}

type parseState int

const (
	mysqlStateStart parseState = iota
	mysqlStateEatMessage
	mysqlStateEatFields
	mysqlStateEatRows

	MysqlStateMax
)

var stateStrings []string = []string{
	"Start",
	"EatMessage",
	"EatFields",
	"EatRows",
}

func (state parseState) String() string {
	return stateStrings[state]
}

type Mysql struct {

	// config
	Ports         []int
	maxStoreRows  int
	maxRowLength  int
	Send_request  bool
	Send_response bool

	transactions       *common.Cache
	transactionTimeout time.Duration

	results publish.Transactions

	// function pointer for mocking
	handleMysql func(mysql *Mysql, m *MysqlMessage, tcp *common.TcpTuple,
		dir uint8, raw_msg []byte)
}

func init() {
	protos.Register("mysql", New)
}

func New(
	testMode bool,
	results publish.Transactions,
	cfg *common.Config,
) (protos.Plugin, error) {
	p := &Mysql{}
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

func (mysql *Mysql) init(results publish.Transactions, config *mysqlConfig) error {
	mysql.setFromConfig(config)

	mysql.transactions = common.NewCache(
		mysql.transactionTimeout,
		protos.DefaultTransactionHashSize)
	mysql.transactions.StartJanitor(mysql.transactionTimeout)
	mysql.handleMysql = handleMysql
	mysql.results = results

	return nil
}

func (mysql *Mysql) setFromConfig(config *mysqlConfig) {
	mysql.Ports = config.Ports
	mysql.maxRowLength = config.MaxRowLength
	mysql.maxStoreRows = config.MaxRows
	mysql.Send_request = config.SendRequest
	mysql.Send_response = config.SendResponse
	mysql.transactionTimeout = config.TransactionTimeout
}

func (mysql *Mysql) getTransaction(k common.HashableTcpTuple) *MysqlTransaction {
	v := mysql.transactions.Get(k)
	if v != nil {
		return v.(*MysqlTransaction)
	}
	return nil
}

func (mysql *Mysql) GetPorts() []int {
	return mysql.Ports
}

func (stream *MysqlStream) PrepareForNewMessage() {
	stream.data = stream.data[stream.parseOffset:]
	stream.parseState = mysqlStateStart
	stream.parseOffset = 0
	stream.isClient = false
	stream.message = nil
}

func mysqlMessageParser(s *MysqlStream) (bool, bool) {

	logp.Debug("mysqldetailed", "MySQL parser called. parseState = %s", s.parseState)

	m := s.message
	for s.parseOffset < len(s.data) {
		switch s.parseState {
		case mysqlStateStart:
			m.start = s.parseOffset
			if len(s.data[s.parseOffset:]) < 5 {
				logp.Warn("MySQL Message too short. Ignore it.")
				return false, false
			}
			hdr := s.data[s.parseOffset : s.parseOffset+5]
			m.PacketLength = uint32(hdr[0]) | uint32(hdr[1])<<8 | uint32(hdr[2])<<16
			m.Seq = uint8(hdr[3])
			m.Typ = uint8(hdr[4])

			logp.Debug("mysqldetailed", "MySQL Header: Packet length %d, Seq %d, Type=%d", m.PacketLength, m.Seq, m.Typ)

			if m.Seq == 0 {
				// starts Command Phase

				if m.Typ == MYSQL_CMD_QUERY {
					// parse request
					m.IsRequest = true
					m.start = s.parseOffset
					s.parseState = mysqlStateEatMessage

				} else {
					// ignore command
					m.IgnoreMessage = true
					s.parseState = mysqlStateEatMessage
				}

				if !s.isClient {
					s.isClient = true
				}

			} else if !s.isClient {
				// parse response
				m.IsRequest = false

				if uint8(hdr[4]) == 0x00 || uint8(hdr[4]) == 0xfe {
					logp.Debug("mysqldetailed", "Received OK response")
					m.start = s.parseOffset
					s.parseState = mysqlStateEatMessage
					m.IsOK = true
				} else if uint8(hdr[4]) == 0xff {
					logp.Debug("mysqldetailed", "Received ERR response")
					m.start = s.parseOffset
					s.parseState = mysqlStateEatMessage
					m.IsError = true
				} else if m.PacketLength == 1 {
					logp.Debug("mysqldetailed", "Query response. Number of fields %d", uint8(hdr[4]))
					m.NumberOfFields = int(hdr[4])
					m.start = s.parseOffset
					s.parseOffset += 5
					s.parseState = mysqlStateEatFields
				} else {
					// something else. ignore
					m.IgnoreMessage = true
					s.parseState = mysqlStateEatMessage
				}

			} else {
				// something else, not expected
				logp.Debug("mysql", "Unexpected MySQL message of type %d received.", m.Typ)
				return false, false
			}
			break

		case mysqlStateEatMessage:
			if len(s.data[s.parseOffset:]) >= int(m.PacketLength)+4 {
				s.parseOffset += 4 //header
				s.parseOffset += int(m.PacketLength)
				m.end = s.parseOffset
				if m.IsRequest {
					m.Query = string(s.data[m.start+5 : m.end])
				} else if m.IsOK {
					// affected rows
					affectedRows, off, complete, err := read_linteger(s.data, m.start+5)
					if !complete {
						return true, false
					}
					if err != nil {
						logp.Debug("mysql", "Error on read_linteger: %s", err)
						return false, false
					}
					m.AffectedRows = affectedRows

					// last insert id
					insertId, off, complete, err := read_linteger(s.data, off)
					if !complete {
						return true, false
					}
					if err != nil {
						logp.Debug("mysql", "Error on read_linteger: %s", err)
						return false, false
					}
					m.InsertId = insertId
				} else if m.IsError {
					// int<1>header (0xff)
					// int<2>error code
					// string[1] sql state marker
					// string[5] sql state
					// string<EOF> error message
					m.ErrorCode = uint16(s.data[m.start+6])<<8 | uint16(s.data[m.start+5])

					m.ErrorInfo = string(s.data[m.start+8:m.start+13]) + ": " + string(s.data[m.start+13:])
				}
				m.Size = uint64(m.end - m.start)
				logp.Debug("mysqldetailed", "Message complete. remaining=%d", len(s.data[s.parseOffset:]))
				return true, true
			} else {
				// wait for more
				return true, false
			}

		case mysqlStateEatFields:
			if len(s.data[s.parseOffset:]) < 4 {
				// wait for more
				return true, false
			}

			hdr := s.data[s.parseOffset : s.parseOffset+4]
			m.PacketLength = uint32(hdr[0]) | uint32(hdr[1])<<8 | uint32(hdr[2])<<16
			m.Seq = uint8(hdr[3])
			logp.Debug("mysqldetailed", "Fields: packet length %d, packet number %d", m.PacketLength, m.Seq)

			if len(s.data[s.parseOffset:]) >= int(m.PacketLength)+4 {
				s.parseOffset += 4 // header

				if uint8(s.data[s.parseOffset]) == 0xfe {
					logp.Debug("mysqldetailed", "Received EOF packet")
					// EOF marker
					s.parseOffset += int(m.PacketLength)

					s.parseState = mysqlStateEatRows
				} else {
					_ /* catalog */, off, complete, err := read_lstring(s.data, s.parseOffset)
					if !complete {
						return true, false
					}
					if err != nil {
						logp.Debug("mysql", "Error on read_lstring: %s", err)
						return false, false
					}
					db /*schema */, off, complete, err := read_lstring(s.data, off)
					if !complete {
						return true, false
					}
					if err != nil {
						logp.Debug("mysql", "Error on read_lstring: %s", err)
						return false, false
					}
					table /* table */, off, complete, err := read_lstring(s.data, off)
					if !complete {
						return true, false
					}
					if err != nil {
						logp.Debug("mysql", "Error on read_lstring: %s", err)
						return false, false
					}

					db_table := string(db) + "." + string(table)

					if len(m.Tables) == 0 {
						m.Tables = db_table
					} else if !strings.Contains(m.Tables, db_table) {
						m.Tables = m.Tables + ", " + db_table
					}
					logp.Debug("mysqldetailed", "db=%s, table=%s", db, table)
					s.parseOffset += int(m.PacketLength)
					// go to next field
				}
			} else {
				// wait for more
				return true, false
			}
			break

		case mysqlStateEatRows:
			if len(s.data[s.parseOffset:]) < 4 {
				// wait for more
				return true, false
			}
			hdr := s.data[s.parseOffset : s.parseOffset+4]
			m.PacketLength = uint32(hdr[0]) | uint32(hdr[1])<<8 | uint32(hdr[2])<<16
			m.Seq = uint8(hdr[3])

			logp.Debug("mysqldetailed", "Rows: packet length %d, packet number %d", m.PacketLength, m.Seq)

			if len(s.data[s.parseOffset:]) >= int(m.PacketLength)+4 {
				s.parseOffset += 4 //header

				if uint8(s.data[s.parseOffset]) == 0xfe {
					logp.Debug("mysqldetailed", "Received EOF packet")
					// EOF marker
					s.parseOffset += int(m.PacketLength)

					if m.end == 0 {
						m.end = s.parseOffset
					} else {
						m.IsTruncated = true
					}
					if !m.IsError {
						// in case the response was sent successfully
						m.IsOK = true
					}
					m.Size = uint64(m.end - m.start)
					return true, true
				} else {
					s.parseOffset += int(m.PacketLength)
					if m.end == 0 && s.parseOffset > MAX_PAYLOAD_SIZE {
						// only send up to here, but read until the end
						m.end = s.parseOffset
					}
					m.NumberOfRows += 1
					// go to next row
				}
			} else {
				// wait for more
				return true, false
			}

			break
		}
	}

	return true, false
}

// messageGap is called when a gap of size `nbytes` is found in the
// tcp stream. Returns true if there is already enough data in the message
// read so far that we can use it further in the stack.
func (mysql *Mysql) messageGap(s *MysqlStream, nbytes int) (complete bool) {

	m := s.message
	switch s.parseState {
	case mysqlStateStart, mysqlStateEatMessage:
		// not enough data yet to be useful
		return false
	case mysqlStateEatFields, mysqlStateEatRows:
		// enough data here
		m.end = s.parseOffset
		if m.IsRequest {
			m.Notes = append(m.Notes, "Packet loss while capturing the request")
		} else {
			m.Notes = append(m.Notes, "Packet loss while capturing the response")
		}
		return true
	}

	return true
}

type mysqlPrivateData struct {
	Data [2]*MysqlStream
}

// Called when the parser has identified a full message.
func (mysql *Mysql) messageComplete(tcptuple *common.TcpTuple, dir uint8, stream *MysqlStream) {
	// all ok, ship it
	msg := stream.data[stream.message.start:stream.message.end]

	if !stream.message.IgnoreMessage {
		mysql.handleMysql(mysql, stream.message, tcptuple, dir, msg)
	}

	// and reset message
	stream.PrepareForNewMessage()
}

func (mysql *Mysql) ConnectionTimeout() time.Duration {
	return mysql.transactionTimeout
}

func (mysql *Mysql) Parse(pkt *protos.Packet, tcptuple *common.TcpTuple,
	dir uint8, private protos.ProtocolData) protos.ProtocolData {

	defer logp.Recover("ParseMysql exception")

	priv := mysqlPrivateData{}
	if private != nil {
		var ok bool
		priv, ok = private.(mysqlPrivateData)
		if !ok {
			priv = mysqlPrivateData{}
		}
	}

	if priv.Data[dir] == nil {
		priv.Data[dir] = &MysqlStream{
			tcptuple: tcptuple,
			data:     pkt.Payload,
			message:  &MysqlMessage{Ts: pkt.Ts},
		}
	} else {
		// concatenate bytes
		priv.Data[dir].data = append(priv.Data[dir].data, pkt.Payload...)
		if len(priv.Data[dir].data) > tcp.TCP_MAX_DATA_IN_STREAM {
			logp.Debug("mysql", "Stream data too large, dropping TCP stream")
			priv.Data[dir] = nil
			return priv
		}
	}

	stream := priv.Data[dir]
	for len(stream.data) > 0 {
		if stream.message == nil {
			stream.message = &MysqlMessage{Ts: pkt.Ts}
		}

		ok, complete := mysqlMessageParser(priv.Data[dir])
		//logp.Debug("mysqldetailed", "mysqlMessageParser returned ok=%b complete=%b", ok, complete)
		if !ok {
			// drop this tcp stream. Will retry parsing with the next
			// segment in it
			priv.Data[dir] = nil
			logp.Debug("mysql", "Ignore MySQL message. Drop tcp stream. Try parsing with the next segment")
			return priv
		}

		if complete {
			mysql.messageComplete(tcptuple, dir, stream)
		} else {
			// wait for more data
			break
		}
	}
	return priv
}

func (mysql *Mysql) GapInStream(tcptuple *common.TcpTuple, dir uint8,
	nbytes int, private protos.ProtocolData) (priv protos.ProtocolData, drop bool) {

	defer logp.Recover("GapInStream(mysql) exception")

	if private == nil {
		return private, false
	}
	mysqlData, ok := private.(mysqlPrivateData)
	if !ok {
		return private, false
	}
	stream := mysqlData.Data[dir]
	if stream == nil || stream.message == nil {
		// nothing to do
		return private, false
	}

	if mysql.messageGap(stream, nbytes) {
		// we need to publish from here
		mysql.messageComplete(tcptuple, dir, stream)
	}

	// we always drop the TCP stream. Because it's binary and len based,
	// there are too few cases in which we could recover the stream (maybe
	// for very large blobs, leaving that as TODO)
	return private, true
}

func (mysql *Mysql) ReceivedFin(tcptuple *common.TcpTuple, dir uint8,
	private protos.ProtocolData) protos.ProtocolData {

	// TODO: check if we have data pending and either drop it to free
	// memory or send it up the stack.
	return private
}

func handleMysql(mysql *Mysql, m *MysqlMessage, tcptuple *common.TcpTuple,
	dir uint8, raw_msg []byte) {

	m.TcpTuple = *tcptuple
	m.Direction = dir
	m.CmdlineTuple = procs.ProcWatcher.FindProcessesTuple(tcptuple.IpPort())
	m.Raw = raw_msg

	if m.IsRequest {
		mysql.receivedMysqlRequest(m)
	} else {
		mysql.receivedMysqlResponse(m)
	}
}

func (mysql *Mysql) receivedMysqlRequest(msg *MysqlMessage) {
	tuple := msg.TcpTuple
	trans := mysql.getTransaction(tuple.Hashable())
	if trans != nil {
		if trans.Mysql != nil {
			logp.Debug("mysql", "Two requests without a Response. Dropping old request: %s", trans.Mysql)
			unmatchedRequests.Add(1)
		}
	} else {
		trans = &MysqlTransaction{Type: "mysql", tuple: tuple}
		mysql.transactions.Put(tuple.Hashable(), trans)
	}

	trans.ts = msg.Ts
	trans.Ts = int64(trans.ts.UnixNano() / 1000) // transactions have microseconds resolution
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

	// Extract the method, by simply taking the first word and
	// making it upper case.
	query := strings.Trim(msg.Query, " \n\t")
	index := strings.IndexAny(query, " \n\t")
	var method string
	if index > 0 {
		method = strings.ToUpper(query[:index])
	} else {
		method = strings.ToUpper(query)
	}

	trans.Query = query
	trans.Method = method

	trans.Mysql = common.MapStr{}

	trans.Notes = msg.Notes

	// save Raw message
	trans.Request_raw = msg.Query
	trans.BytesIn = msg.Size
}

func (mysql *Mysql) receivedMysqlResponse(msg *MysqlMessage) {
	trans := mysql.getTransaction(msg.TcpTuple.Hashable())
	if trans == nil {
		logp.Debug("mysql", "Response from unknown transaction. Ignoring.")
		unmatchedResponses.Add(1)
		return
	}
	// check if the request was received
	if trans.Mysql == nil {
		logp.Debug("mysql", "Response from unknown transaction. Ignoring.")
		unmatchedResponses.Add(1)
		return

	}
	// save json details
	trans.Mysql.Update(common.MapStr{
		"affected_rows": msg.AffectedRows,
		"insert_id":     msg.InsertId,
		"num_rows":      msg.NumberOfRows,
		"num_fields":    msg.NumberOfFields,
		"iserror":       msg.IsError,
		"error_code":    msg.ErrorCode,
		"error_message": msg.ErrorInfo,
	})
	trans.BytesOut = msg.Size
	trans.Path = msg.Tables

	trans.ResponseTime = int32(msg.Ts.Sub(trans.ts).Nanoseconds() / 1e6) // resp_time in milliseconds

	// save Raw message
	if len(msg.Raw) > 0 {
		fields, rows := mysql.parseMysqlResponse(msg.Raw)

		trans.Response_raw = common.DumpInCSVFormat(fields, rows)
	}

	trans.Notes = append(trans.Notes, msg.Notes...)

	mysql.publishTransaction(trans)
	mysql.transactions.Delete(trans.tuple.Hashable())

	logp.Debug("mysql", "Mysql transaction completed: %s", trans.Mysql)
	logp.Debug("mysql", "%s", trans.Response_raw)
}

func (mysql *Mysql) parseMysqlResponse(data []byte) ([]string, [][]string) {

	length, err := read_length(data, 0)
	if err != nil {
		logp.Warn("Invalid response: %v", err)
		return []string{}, [][]string{}
	}
	if length < 1 {
		logp.Warn("Warning: Skipping empty Response")
		return []string{}, [][]string{}
	}

	fields := []string{}
	rows := [][]string{}

	if len(data) < 5 {
		logp.Warn("Invalid response: data less than 4 bytes")
		return []string{}, [][]string{}
	}

	if uint8(data[4]) == 0x00 {
		// OK response
	} else if uint8(data[4]) == 0xff {
		// Error response
	} else {
		offset := 5

		logp.Debug("mysql", "Data len: %d", len(data))

		// Read fields
		for {
			length, err = read_length(data, offset)
			if err != nil {
				logp.Warn("Invalid response: %v", err)
				return []string{}, [][]string{}
			}

			if len(data[offset:]) < 5 {
				logp.Warn("Invalid response.")
				return []string{}, [][]string{}
			}

			if uint8(data[offset+4]) == 0xfe {
				// EOF
				offset += length + 4
				break
			}

			_ /* catalog */, off, complete, err := read_lstring(data, offset+4)
			if err != nil || !complete {
				logp.Debug("mysql", "Reading field: %v %v", err, complete)
				return fields, rows
			}
			_ /*database*/, off, complete, err = read_lstring(data, off)
			if err != nil || !complete {
				logp.Debug("mysql", "Reading field: %v %v", err, complete)
				return fields, rows
			}
			_ /*table*/, off, complete, err = read_lstring(data, off)
			if err != nil || !complete {
				logp.Debug("mysql", "Reading field: %v %v", err, complete)
				return fields, rows
			}
			_ /*org table*/, off, complete, err = read_lstring(data, off)
			if err != nil || !complete {
				logp.Debug("mysql", "Reading field: %v %v", err, complete)
				return fields, rows
			}
			name, off, complete, err := read_lstring(data, off)
			if err != nil || !complete {
				logp.Debug("mysql", "Reading field: %v %v", err, complete)
				return fields, rows
			}
			_ /* org name */, off, complete, err = read_lstring(data, off)
			if err != nil || !complete {
				logp.Debug("mysql", "Reading field: %v %v", err, complete)
				return fields, rows
			}

			fields = append(fields, string(name))

			offset += length + 4
			if len(data) < offset {
				logp.Warn("Invalid response.")
				return []string{}, [][]string{}
			}
		}

		// Read rows
		for offset < len(data) {
			var row []string
			var row_len int

			if len(data[offset:]) < 5 {
				logp.Warn("Invalid response.")
				break
			}

			if uint8(data[offset+4]) == 0xfe {
				// EOF
				offset += length + 4
				break
			}

			length, err = read_length(data, offset)
			if err != nil {
				logp.Warn("Invalid response: %v", err)
				break
			}
			off := offset + 4 // skip length + packet number
			start := off
			for off < start+length {
				var text []byte

				if uint8(data[off]) == 0xfb {
					text = []byte("NULL")
					off++
				} else {
					var err error
					var complete bool
					text, off, complete, err = read_lstring(data, off)
					if err != nil || !complete {
						logp.Debug("mysql", "Error parsing rows: %s %b", err, complete)
						// nevertheless, return what we have so far
						return fields, rows
					}
				}

				if row_len < mysql.maxRowLength {
					if row_len+len(text) > mysql.maxRowLength {
						text = text[:mysql.maxRowLength-row_len]
					}
					row = append(row, string(text))
					row_len += len(text)
				}
			}

			logp.Debug("mysqldetailed", "Append row: %v", row)

			rows = append(rows, row)
			if len(rows) >= mysql.maxStoreRows {
				break
			}

			offset += length + 4
		}
	}
	return fields, rows
}

func (mysql *Mysql) publishTransaction(t *MysqlTransaction) {

	if mysql.results == nil {
		return
	}

	logp.Debug("mysql", "mysql.results exists")

	event := common.MapStr{}
	event["type"] = "mysql"

	if t.Mysql["iserror"].(bool) {
		event["status"] = common.ERROR_STATUS
	} else {
		event["status"] = common.OK_STATUS
	}

	event["responsetime"] = t.ResponseTime
	if mysql.Send_request {
		event["request"] = t.Request_raw
	}
	if mysql.Send_response {
		event["response"] = t.Response_raw
	}
	event["method"] = t.Method
	event["query"] = t.Query
	event["mysql"] = t.Mysql
	event["path"] = t.Path
	event["bytes_out"] = t.BytesOut
	event["bytes_in"] = t.BytesIn

	if len(t.Notes) > 0 {
		event["notes"] = t.Notes
	}

	event["@timestamp"] = common.Time(t.ts)
	event["src"] = &t.Src
	event["dst"] = &t.Dst

	mysql.results.PublishTransaction(event)
}

func read_lstring(data []byte, offset int) ([]byte, int, bool, error) {
	length, off, complete, err := read_linteger(data, offset)
	if err != nil {
		return nil, 0, false, err
	}
	if !complete || len(data[off:]) < int(length) {
		return nil, 0, false, nil
	}

	return data[off : off+int(length)], off + int(length), true, nil
}
func read_linteger(data []byte, offset int) (uint64, int, bool, error) {
	if len(data) < offset+1 {
		return 0, 0, false, nil
	}
	switch uint8(data[offset]) {
	case 0xfe:
		if len(data[offset+1:]) < 8 {
			return 0, 0, false, nil
		}
		return uint64(data[offset+1]) | uint64(data[offset+2])<<8 |
				uint64(data[offset+2])<<16 | uint64(data[offset+3])<<24 |
				uint64(data[offset+4])<<32 | uint64(data[offset+5])<<40 |
				uint64(data[offset+6])<<48 | uint64(data[offset+7])<<56,
			offset + 9, true, nil
	case 0xfd:
		if len(data[offset+1:]) < 3 {
			return 0, 0, false, nil
		}
		return uint64(data[offset+1]) | uint64(data[offset+2])<<8 |
			uint64(data[offset+3])<<16, offset + 4, true, nil
	case 0xfc:
		if len(data[offset+1:]) < 2 {
			return 0, 0, false, nil
		}
		return uint64(data[offset+1]) | uint64(data[offset+2])<<8, offset + 3, true, nil
	}

	if uint64(data[offset]) >= 0xfb {
		return 0, 0, false, fmt.Errorf("Unexpected value in read_linteger")
	}

	return uint64(data[offset]), offset + 1, true, nil
}

// Read a mysql length field (3 bytes LE)
func read_length(data []byte, offset int) (int, error) {
	if len(data[offset:]) < 3 {
		return 0, errors.New("Data too small to contain a valid length")
	}
	length := uint32(data[offset]) |
		uint32(data[offset+1])<<8 |
		uint32(data[offset+2])<<16
	return int(length), nil
}
