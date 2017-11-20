package mysql

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/monitoring"

	"github.com/elastic/beats/packetbeat/procs"
	"github.com/elastic/beats/packetbeat/protos"
	"github.com/elastic/beats/packetbeat/protos/tcp"
)

// Packet types
const (
	mysqlCmdQuery       = 3
	mysqlCmdStmtPrepare = 22
	mysqlCmdStmtExecute = 23
	mysqlCmdStmtClose   = 25
)

const maxPayloadSize = 100 * 1024

var (
	unmatchedRequests  = monitoring.NewInt(nil, "mysql.unmatched_requests")
	unmatchedResponses = monitoring.NewInt(nil, "mysql.unmatched_responses")
)

type mysqlMessage struct {
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

	direction    uint8
	isTruncated  bool
	tcpTuple     common.TCPTuple
	cmdlineTuple *common.CmdlineTuple
	raw          []byte
	notes        []string

	statementID       int
	numberOfParameter int
	param             string
}

type mysqlTransaction struct {
	tuple        common.TCPTuple
	src          common.Endpoint
	dst          common.Endpoint
	responseTime int32
	ts           time.Time
	query        string
	method       string
	path         string // for mysql, Path refers to the mysql table queried
	bytesOut     uint64
	bytesIn      uint64
	notes        []string

	mysql common.MapStr

	requestRaw  string
	responseRaw string

	statementID int    // for prepare statement
	param       string // for execute statement param
}

type mysqlStream struct {
	data []byte

	parseOffset int
	parseState  parseState
	isClient    bool

	message *mysqlMessage
}

type parseState int

const (
	mysqlStateStart parseState = iota
	mysqlStateEatMessage
	mysqlStateEatFields
	mysqlStateEatRows

	mysqlStateMax
)

var stateStrings = []string{
	"Start",
	"EatMessage",
	"EatFields",
	"EatRows",
}

func (state parseState) String() string {
	return stateStrings[state]
}

type mysqlPlugin struct {

	// config
	ports        []int
	maxStoreRows int
	maxRowLength int
	sendRequest  bool
	sendResponse bool

	transactions       *common.Cache
	transactionTimeout time.Duration

	// prepare statement cache
	preparestatements       *common.Cache
	preparestatementTimeout time.Duration

	results protos.Reporter

	// function pointer for mocking
	handleMysql func(mysql *mysqlPlugin, m *mysqlMessage, tcp *common.TCPTuple,
		dir uint8, raw_msg []byte)
}

func init() {
	protos.Register("mysql", New)
}

func New(
	testMode bool,
	results protos.Reporter,
	cfg *common.Config,
) (protos.Plugin, error) {
	p := &mysqlPlugin{}
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

func (mysql *mysqlPlugin) init(results protos.Reporter, config *mysqlConfig) error {
	mysql.setFromConfig(config)

	mysql.transactions = common.NewCache(
		mysql.transactionTimeout,
		protos.DefaultTransactionHashSize)
	mysql.transactions.StartJanitor(mysql.transactionTimeout)

	// preparement cache
	mysql.preparestatements = common.NewCache(
		mysql.preparestatementTimeout,
		protos.DefaultTransactionHashSize)
	mysql.preparestatements.StartJanitor(mysql.preparestatementTimeout)

	mysql.handleMysql = handleMysql
	mysql.results = results

	return nil
}

func (mysql *mysqlPlugin) setFromConfig(config *mysqlConfig) {
	mysql.ports = config.Ports
	mysql.maxRowLength = config.MaxRowLength
	mysql.maxStoreRows = config.MaxRows
	mysql.sendRequest = config.SendRequest
	mysql.sendResponse = config.SendResponse
	mysql.transactionTimeout = config.TransactionTimeout
	mysql.preparestatementTimeout = config.TransactionTimeout
}

func (mysql *mysqlPlugin) getTransaction(k common.HashableTCPTuple) *mysqlTransaction {
	v := mysql.transactions.Get(k)
	if v != nil {
		return v.(*mysqlTransaction)
	}
	return nil
}

// cache the prepare statement info
type mysqlStmtData struct {
	query           string
	numOfParameters int
	nparamType      []uint8
}
type mysqlStmtMap map[int]*mysqlStmtData

func (mysql *mysqlPlugin) getStmtsMap(k common.HashableTCPTuple) mysqlStmtMap {
	v := mysql.preparestatements.Get(k)
	if v != nil {
		return v.(mysqlStmtMap)
	}
	return nil
}

func (mysql *mysqlPlugin) GetPorts() []int {
	return mysql.ports
}

func (stream *mysqlStream) prepareForNewMessage() {
	stream.data = stream.data[stream.parseOffset:]
	stream.parseState = mysqlStateStart
	stream.parseOffset = 0
	stream.isClient = false
	stream.message = nil
}

func mysqlMessageParser(s *mysqlStream) (bool, bool) {
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
			m.packetLength = uint32(hdr[0]) | uint32(hdr[1])<<8 | uint32(hdr[2])<<16
			m.seq = hdr[3]
			m.typ = hdr[4]

			logp.Debug("mysqldetailed", "MySQL Header: Packet length %d, Seq %d, Type=%d", m.packetLength, m.seq, m.typ)

			if m.seq == 0 {
				// starts Command Phase

				if m.typ == mysqlCmdQuery || m.typ == mysqlCmdStmtPrepare ||
					m.typ == mysqlCmdStmtExecute || m.typ == mysqlCmdStmtClose {
					// parse request
					m.isRequest = true
					m.start = s.parseOffset
					s.parseState = mysqlStateEatMessage

				} else {
					// ignore command
					m.ignoreMessage = true
					s.parseState = mysqlStateEatMessage
				}

				if !s.isClient {
					s.isClient = true
				}

			} else if !s.isClient {
				// parse response
				m.isRequest = false

				if hdr[4] == 0x00 || hdr[4] == 0xfe {
					logp.Debug("mysqldetailed", "Received OK response")
					m.start = s.parseOffset
					s.parseState = mysqlStateEatMessage
					m.isOK = true
				} else if hdr[4] == 0xff {
					logp.Debug("mysqldetailed", "Received ERR response")
					m.start = s.parseOffset
					s.parseState = mysqlStateEatMessage
					m.isError = true
				} else if m.packetLength == 1 {
					logp.Debug("mysqldetailed", "Query response. Number of fields %d", hdr[4])
					m.numberOfFields = int(hdr[4])
					m.start = s.parseOffset
					s.parseOffset += 5
					s.parseState = mysqlStateEatFields
				} else {
					// something else. ignore
					m.ignoreMessage = true
					s.parseState = mysqlStateEatMessage
				}

			} else {
				// something else, not expected
				logp.Debug("mysql", "Unexpected MySQL message of type %d received.", m.typ)
				return false, false
			}

		case mysqlStateEatMessage:
			if len(s.data[s.parseOffset:]) < int(m.packetLength)+4 {
				// wait for more data
				return true, false
			}

			s.parseOffset += 4 //header
			s.parseOffset += int(m.packetLength)
			m.end = s.parseOffset
			if m.isRequest {
				// get the statement id
				if m.typ == mysqlCmdStmtExecute || m.typ == mysqlCmdStmtClose {
					m.statementID = int(s.data[m.start+5]) | int(s.data[m.start+6])<<8 | int(s.data[m.start+7])<<16 | int(s.data[m.start+8])<<24
				} else {
					m.query = string(s.data[m.start+5 : m.end])
				}

			} else if m.isOK {
				// affected rows
				affectedRows, off, complete, err := readLinteger(s.data, m.start+5)
				if !complete {
					return true, false
				}
				if err != nil {
					logp.Debug("mysql", "Error on read_linteger: %s", err)
					return false, false
				}
				m.affectedRows = affectedRows

				// last insert id
				insertID, _, complete, err := readLinteger(s.data, off)
				if !complete {
					return true, false
				}
				if err != nil {
					logp.Debug("mysql", "Error on read_linteger: %s", err)
					return false, false
				}
				m.insertID = insertID
			} else if m.isError {
				// int<1>header (0xff)
				// int<2>error code
				// string[1] sql state marker
				// string[5] sql state
				// string<EOF> error message
				m.errorCode = uint16(s.data[m.start+6])<<8 | uint16(s.data[m.start+5])

				m.errorInfo = string(s.data[m.start+8:m.start+13]) + ": " + string(s.data[m.start+13:])
			}
			m.size = uint64(m.end - m.start)
			logp.Debug("mysqldetailed", "Message complete. remaining=%d",
				len(s.data[s.parseOffset:]))

			// PREPARE_OK packet for Prepared Statement
			// a trick for classify special OK packet
			if m.isOK && m.packetLength == 12 {
				idPtr := s.data[m.start+5 : m.start+13]
				m.statementID = int(idPtr[0]) | int(idPtr[1])<<8 | int(idPtr[2])<<16 | int(idPtr[3])<<24
				m.numberOfFields = int(idPtr[4]) | int(idPtr[5])<<8
				m.numberOfParameter = int(idPtr[6]) | int(idPtr[7])<<8
				if m.numberOfFields > 0 {
					s.parseState = mysqlStateEatFields
				} else {
					if m.numberOfParameter > 0 {
						s.parseState = mysqlStateEatRows
					}
				}
				logp.Debug("mysqldetailed", "Prepare Statement Response statementID = %d, numberOfFields = %d,numberOfParameter = %d", m.statementID, m.numberOfFields, m.numberOfParameter)
			} else {
				return true, true
			}
		case mysqlStateEatFields:
			if len(s.data[s.parseOffset:]) < 4 {
				// wait for more
				return true, false
			}

			hdr := s.data[s.parseOffset : s.parseOffset+4]
			m.packetLength = uint32(hdr[0]) | uint32(hdr[1])<<8 | uint32(hdr[2])<<16
			m.seq = hdr[3]
			logp.Debug("mysqldetailed", "Fields: packet length %d, packet number %d", m.packetLength, m.seq)

			if len(s.data[s.parseOffset:]) >= int(m.packetLength)+4 {
				s.parseOffset += 4 // header

				if s.data[s.parseOffset] == 0xfe {
					logp.Debug("mysqldetailed", "Received EOF packet")
					// EOF marker
					s.parseOffset += int(m.packetLength)

					s.parseState = mysqlStateEatRows
				} else {
					_ /* catalog */, off, complete, err := readLstring(s.data, s.parseOffset)
					if !complete {
						return true, false
					}
					if err != nil {
						logp.Debug("mysql", "Error on read_lstring: %s", err)
						return false, false
					}
					db /*schema */, off, complete, err := readLstring(s.data, off)
					if !complete {
						return true, false
					}
					if err != nil {
						logp.Debug("mysql", "Error on read_lstring: %s", err)
						return false, false
					}
					table /* table */, _ /*off*/, complete, err := readLstring(s.data, off)
					if !complete {
						return true, false
					}
					if err != nil {
						logp.Debug("mysql", "Error on read_lstring: %s", err)
						return false, false
					}

					dbTable := string(db) + "." + string(table)

					if len(m.tables) == 0 {
						m.tables = dbTable
					} else if !strings.Contains(m.tables, dbTable) {
						m.tables = m.tables + ", " + dbTable
					}
					logp.Debug("mysqldetailed", "db=%s, table=%s", db, table)
					s.parseOffset += int(m.packetLength)
					// go to next field
				}
			} else {
				// wait for more
				return true, false
			}

		case mysqlStateEatRows:
			if len(s.data[s.parseOffset:]) < 4 {
				// wait for more
				return true, false
			}
			hdr := s.data[s.parseOffset : s.parseOffset+4]
			m.packetLength = uint32(hdr[0]) | uint32(hdr[1])<<8 | uint32(hdr[2])<<16
			m.seq = hdr[3]

			logp.Debug("mysqldetailed", "Rows: packet length %d, packet number %d", m.packetLength, m.seq)

			if len(s.data[s.parseOffset:]) < int(m.packetLength)+4 {
				// wait for more
				return true, false
			}

			s.parseOffset += 4 //header

			if s.data[s.parseOffset] == 0xfe {
				logp.Debug("mysqldetailed", "Received EOF packet")
				// EOF marker
				s.parseOffset += int(m.packetLength)

				if m.end == 0 {
					m.end = s.parseOffset
				} else {
					m.isTruncated = true
				}
				if !m.isError {
					// in case the response was sent successfully
					m.isOK = true
				}
				m.size = uint64(m.end - m.start)
				return true, true
			}

			s.parseOffset += int(m.packetLength)
			if m.end == 0 && s.parseOffset > maxPayloadSize {
				// only send up to here, but read until the end
				m.end = s.parseOffset
			}
			m.numberOfRows++
			// go to next row
		}
	}

	return true, false
}

// messageGap is called when a gap of size `nbytes` is found in the
// tcp stream. Returns true if there is already enough data in the message
// read so far that we can use it further in the stack.
func (mysql *mysqlPlugin) messageGap(s *mysqlStream, nbytes int) (complete bool) {
	m := s.message
	switch s.parseState {
	case mysqlStateStart, mysqlStateEatMessage:
		// not enough data yet to be useful
		return false
	case mysqlStateEatFields, mysqlStateEatRows:
		// enough data here
		m.end = s.parseOffset
		if m.isRequest {
			m.notes = append(m.notes, "Packet loss while capturing the request")
		} else {
			m.notes = append(m.notes, "Packet loss while capturing the response")
		}
		return true
	}

	return true
}

type mysqlPrivateData struct {
	data [2]*mysqlStream
}

// Called when the parser has identified a full message.
func (mysql *mysqlPlugin) messageComplete(tcptuple *common.TCPTuple, dir uint8, stream *mysqlStream) {
	// all ok, ship it
	msg := stream.data[stream.message.start:stream.message.end]

	if !stream.message.ignoreMessage {
		mysql.handleMysql(mysql, stream.message, tcptuple, dir, msg)
	}

	// and reset message
	stream.prepareForNewMessage()
}

func (mysql *mysqlPlugin) ConnectionTimeout() time.Duration {
	return mysql.transactionTimeout
}

func (mysql *mysqlPlugin) Parse(pkt *protos.Packet, tcptuple *common.TCPTuple,
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

	if priv.data[dir] == nil {
		priv.data[dir] = &mysqlStream{
			data:    pkt.Payload,
			message: &mysqlMessage{ts: pkt.Ts},
		}
	} else {
		// concatenate bytes
		priv.data[dir].data = append(priv.data[dir].data, pkt.Payload...)
		if len(priv.data[dir].data) > tcp.TCPMaxDataInStream {
			logp.Debug("mysql", "Stream data too large, dropping TCP stream")
			priv.data[dir] = nil
			return priv
		}
	}

	stream := priv.data[dir]
	for len(stream.data) > 0 {
		if stream.message == nil {
			stream.message = &mysqlMessage{ts: pkt.Ts}
		}

		ok, complete := mysqlMessageParser(priv.data[dir])
		//logp.Debug("mysqldetailed", "mysqlMessageParser returned ok=%b complete=%b", ok, complete)
		if !ok {
			// drop this tcp stream. Will retry parsing with the next
			// segment in it
			priv.data[dir] = nil
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

func (mysql *mysqlPlugin) GapInStream(tcptuple *common.TCPTuple, dir uint8,
	nbytes int, private protos.ProtocolData) (priv protos.ProtocolData, drop bool) {

	defer logp.Recover("GapInStream(mysql) exception")

	if private == nil {
		return private, false
	}
	mysqlData, ok := private.(mysqlPrivateData)
	if !ok {
		return private, false
	}
	stream := mysqlData.data[dir]
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

func (mysql *mysqlPlugin) ReceivedFin(tcptuple *common.TCPTuple, dir uint8,
	private protos.ProtocolData) protos.ProtocolData {

	// TODO: check if we have data pending and either drop it to free
	// memory or send it up the stack.
	return private
}

func handleMysql(mysql *mysqlPlugin, m *mysqlMessage, tcptuple *common.TCPTuple,
	dir uint8, rawMsg []byte) {

	m.tcpTuple = *tcptuple
	m.direction = dir
	m.cmdlineTuple = procs.ProcWatcher.FindProcessesTuple(tcptuple.IPPort())
	m.raw = rawMsg

	if m.isRequest {
		mysql.receivedMysqlRequest(m)
	} else {
		mysql.receivedMysqlResponse(m)
	}
}

func (mysql *mysqlPlugin) receivedMysqlRequest(msg *mysqlMessage) {
	tuple := msg.tcpTuple
	trans := mysql.getTransaction(tuple.Hashable())
	if trans != nil {
		if trans.mysql != nil {
			logp.Debug("mysql", "Two requests without a Response. Dropping old request: %s", trans.mysql)
			unmatchedRequests.Add(1)
		}
	} else {
		trans = &mysqlTransaction{tuple: tuple}
		mysql.transactions.Put(tuple.Hashable(), trans)
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

	// try to get query string for Execute statement from cache
	// and delete statement id for Close statement from cache
	if msg.statementID != 0 {
		trans.statementID = msg.statementID
		stmts := mysql.getStmtsMap(msg.tcpTuple.Hashable())
		if stmts == nil {
			logp.Debug("mysqldetailed", "Request execute statement for no stream map. Ignoring.")
			return
		}
		if msg.typ == mysqlCmdStmtExecute {
			if value, ok := stmts[trans.statementID]; ok {
				trans.query = value.query
				// parse parameters
				parameters := mysql.parseMysqlExecuteStatement(msg.raw, value)
				trans.param = strings.Join(parameters, "#")
				logp.Debug("mysqldetailed", "parameters: %s", trans.param)
			} else {
				logp.Debug("mysqldetailed", "Request execute statement from unknown prepare statement ID. Ignoring.")
				mysql.transactions.Delete(tuple.Hashable())
				return
			}
		} else if msg.typ == mysqlCmdStmtClose {
			delete(stmts, trans.statementID)
			mysql.transactions.Delete(tuple.Hashable())
		}
	} else {
		trans.query = msg.query
	}

	// Extract the method, by simply taking the first word and
	// making it upper case.
	query := strings.Trim(trans.query, " \n\t")
	index := strings.IndexAny(query, " \n\t")
	var method string
	if index > 0 {
		method = strings.ToUpper(query[:index])
	} else {
		method = strings.ToUpper(query)
	}

	trans.query = query
	trans.method = method

	trans.mysql = common.MapStr{}

	trans.notes = msg.notes

	// save Raw message
	trans.requestRaw = msg.query
	trans.bytesIn = msg.size
}

func (mysql *mysqlPlugin) receivedMysqlResponse(msg *mysqlMessage) {
	trans := mysql.getTransaction(msg.tcpTuple.Hashable())
	if trans == nil {
		logp.Debug("mysql", "Response from unknown transaction. Ignoring.")
		unmatchedResponses.Add(1)
		return
	}
	// check if the request was received
	if trans.mysql == nil {
		logp.Debug("mysql", "Response from unknown transaction. Ignoring.")
		unmatchedResponses.Add(1)
		return

	}
	// save json details
	trans.mysql.Update(common.MapStr{
		"affected_rows": msg.affectedRows,
		"insert_id":     msg.insertID,
		"num_rows":      msg.numberOfRows,
		"num_fields":    msg.numberOfFields,
		"iserror":       msg.isError,
		"error_code":    msg.errorCode,
		"error_message": msg.errorInfo,
	})
	if msg.statementID != 0 {
		// cache prepare statement response info
		stmts := mysql.getStmtsMap(msg.tcpTuple.Hashable())
		if stmts == nil {
			stmts = mysqlStmtMap{}
			stmtData := &mysqlStmtData{
				query:           trans.query,
				numOfParameters: msg.numberOfParameter,
			}
			stmts[msg.statementID] = stmtData
			mysql.preparestatements.Put(msg.tcpTuple.Hashable(), stmts)
		} else {
			if stmts[msg.statementID] == nil {
				stmtData := &mysqlStmtData{
					query:           trans.query,
					numOfParameters: msg.numberOfParameter,
				}
				stmts[msg.statementID] = stmtData
			}
			mysql.preparestatements.Put(msg.tcpTuple.Hashable(), stmts)
		}
		// not publish prepare statement
		mysql.transactions.Delete(trans.tuple.Hashable())
		return
	}

	trans.bytesOut = msg.size
	trans.path = msg.tables

	trans.responseTime = int32(msg.ts.Sub(trans.ts).Nanoseconds() / 1e6) // resp_time in milliseconds

	// save Raw message
	if len(msg.raw) > 0 {
		fields, rows := mysql.parseMysqlResponse(msg.raw)

		trans.responseRaw = common.DumpInCSVFormat(fields, rows)
	}

	trans.notes = append(trans.notes, msg.notes...)

	mysql.publishTransaction(trans)
	mysql.transactions.Delete(trans.tuple.Hashable())

	logp.Debug("mysql", "Mysql transaction completed: %s %s %s", trans.query, trans.param, trans.mysql)
	//	logp.Debug("mysql", "%s", trans.responseRaw)
}

func (mysql *mysqlPlugin) parseMysqlExecuteStatement(data []byte, stmtdata *mysqlStmtData) []string {

	var paramType, paramUnsigned uint8
	nparamType := []uint8{}
	paramString := []string{}
	nparam := stmtdata.numOfParameters
	offset := 0
	// mysql hdr
	offset += 4
	// cmd type
	offset++
	// stmt id
	offset += 4
	// flags
	offset++
	// iterations
	offset += 4
	// null-bitmap
	if nparam > 0 {
		offset += (nparam + 7) / 8
	}
	// stmt bound
	stmtBound := data[offset]
	offset++
	paramOffset := offset
	if stmtBound == 1 {
		paramOffset += nparam * 2
		// First call or rebound (1)
		for stmtPos := 0; stmtPos < nparam; stmtPos++ {
			paramType = uint8(data[offset])
			offset++
			nparamType = append(nparamType, paramType)
			logp.Debug("mysqldetailed", "type = %d", paramType)
			paramUnsigned = uint8(data[offset])
			offset++
			if paramUnsigned != 0 {
				logp.Debug("mysql", "Illegal param unsigned")
				return []string{}
			}
		}
		// Save param type info
		stmtdata.nparamType = nparamType
	} else {
		// Subsequent call (0)
		if len(stmtdata.nparamType) > 0 {
			// get saved param type info
			nparamType = stmtdata.nparamType
		} else {
			return []string{}
		}
	}

	for stmtPos := 0; stmtPos < nparam; stmtPos++ {
		paramType = nparamType[stmtPos]
		// dissect parameter on paramType
		switch paramType {
		// FIELD_TYPE_TINY
		case 0x01:
			paramOffset++
			valueString := strconv.Itoa(int(data[paramOffset]))
			paramString = append(paramString, valueString)
		// FIELD_TYPE_SHORT
		case 0x02:
			valueString := strconv.Itoa(int(data[paramOffset]) | int(data[paramOffset+1])<<8)
			paramString = append(paramString, valueString)
			paramOffset += 2
		// FIELD_TYPE_LONG
		case 0x03:
			valueString := strconv.Itoa(int(data[paramOffset]) | int(data[paramOffset+1])<<8 | int(data[paramOffset+2])<<16 | int(data[paramOffset+3])<<24)
			paramString = append(paramString, valueString)
			paramOffset += 4
		//FIELD_TYPE_FLOAT
		case 0x04:
			paramString = append(paramString, "TYPE_FLOAT")
			paramOffset += 4
		// FIELD_TYPE_DOUBLE
		case 0x05:
			paramString = append(paramString, "TYPE_DOUBLE")
			paramOffset += 4
		// FIELD_TYPE_NULL
		case 0x06:
			paramString = append(paramString, "TYPE_NULL")
		//  FIELD_TYPE_LONGLONG
		case 0x08:
			valueString := strconv.FormatInt(int64(data[paramOffset])|int64(data[paramOffset+1])<<8|
				int64(data[paramOffset+2])<<16|int64(data[paramOffset+3])<<24|
				int64(data[paramOffset+4])<<32|int64(data[paramOffset+5])<<40|
				int64(data[paramOffset+6])<<48|int64(data[paramOffset+7])<<56, 10)
			paramString = append(paramString, valueString)
			paramOffset += 8
		// FIELD_TYPE_TIMESTAMP
		// FIELD_TYPE_DATETIME
		// FIELD_TYPE_DATE
		case 0x07, 0x0c, 0x0a:
			var year, month, day, hour, minute, second string
			paramLen := int(data[paramOffset])
			paramOffset++
			if paramLen >= 2 {
				year = strconv.Itoa((int(data[paramOffset]) | int(data[paramOffset+1])<<8))
			}
			if paramLen >= 4 {
				month = strconv.Itoa(int(data[paramOffset+2]))
				day = strconv.Itoa(int(data[paramOffset+3]))
			}
			if paramLen >= 7 {
				hour = strconv.Itoa(int(data[paramOffset+4]))
				minute = strconv.Itoa(int(data[paramOffset+5]))
				second = strconv.Itoa(int(data[paramOffset+6]))
			}
			if paramLen >= 11 {
				// Billionth of a second
				// Skip
			}
			datetime := year + "/" + month + "/" + day + " " + hour + ":" + minute + ":" + second
			paramString = append(paramString, datetime)
			paramOffset += paramLen
		// FIELD_TYPE_TIME
		case 0x0b:
			paramLen := int(data[paramOffset])
			paramOffset++
			paramString = append(paramString, "TYPE_TIME")
			paramOffset += paramLen
		// FIELD_TYPE_VAR_STRING
		// FIELD_TYPE_BLOB
		// FIELD_TYPE_STRING
		case 0xf6, 0xfc, 0xfd, 0xfe:
			paramLen := int(data[paramOffset])
			paramOffset++
			switch paramLen {
			case 0xfc: /* 252 - 64k chars */
				paramLen16 := int(data[paramOffset]) | int(data[paramOffset+1])<<8
				paramOffset += 2
				paramString = append(paramString, string(data[paramOffset:paramOffset+paramLen16]))
				paramOffset += paramLen16
			case 0xfd: /* 64k - 16M chars */
				paramLen24 := int(data[paramOffset]) | int(data[paramOffset+1])<<8 | int(data[paramOffset+2])<<16
				paramOffset += 3
				paramString = append(paramString, string(data[paramOffset:paramOffset+paramLen24]))
				paramOffset += paramLen24
			default: /* < 252 chars     */
				paramString = append(paramString, string(data[paramOffset:paramOffset+paramLen]))
				//logp.Debug("mysql", "Field_type_var_string : %s", string(data[paramOffset:paramOffset+paramLen]))
				paramOffset += paramLen
			}
		default:
			logp.Debug("mysql", "Unknown param type")
			return []string{}
		}

	}
	return paramString
}

func (mysql *mysqlPlugin) parseMysqlResponse(data []byte) ([]string, [][]string) {
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
	rows := [][]string{}

	if len(data) < 5 {
		logp.Warn("Invalid response: data less than 4 bytes")
		return []string{}, [][]string{}
	}

	if data[4] == 0x00 {
		// OK response
	} else if data[4] == 0xff {
		// Error response
	} else {
		offset := 5

		logp.Debug("mysql", "Data len: %d", len(data))

		// Read fields
		for {
			length, err = readLength(data, offset)
			if err != nil {
				logp.Warn("Invalid response: %v", err)
				return []string{}, [][]string{}
			}

			if len(data[offset:]) < 5 {
				logp.Warn("Invalid response.")
				return []string{}, [][]string{}
			}

			if data[offset+4] == 0xfe {
				// EOF
				offset += length + 4
				break
			}

			_ /* catalog */, off, complete, err := readLstring(data, offset+4)
			if err != nil || !complete {
				logp.Debug("mysql", "Reading field: %v %v", err, complete)
				return fields, rows
			}
			_ /*database*/, off, complete, err = readLstring(data, off)
			if err != nil || !complete {
				logp.Debug("mysql", "Reading field: %v %v", err, complete)
				return fields, rows
			}
			_ /*table*/, off, complete, err = readLstring(data, off)
			if err != nil || !complete {
				logp.Debug("mysql", "Reading field: %v %v", err, complete)
				return fields, rows
			}
			_ /*org table*/, off, complete, err = readLstring(data, off)
			if err != nil || !complete {
				logp.Debug("mysql", "Reading field: %v %v", err, complete)
				return fields, rows
			}
			name, off, complete, err := readLstring(data, off)
			if err != nil || !complete {
				logp.Debug("mysql", "Reading field: %v %v", err, complete)
				return fields, rows
			}
			_ /* org name */, _ /*off*/, complete, err = readLstring(data, off)
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
			var rowLen int

			if len(data[offset:]) < 5 {
				logp.Warn("Invalid response.")
				break
			}

			if data[offset+4] == 0xfe {
				// EOF
				offset += length + 4
				break
			}

			length, err = readLength(data, offset)
			if err != nil {
				logp.Warn("Invalid response: %v", err)
				break
			}
			off := offset + 4 // skip length + packet number
			start := off
			for off < start+length {
				var text []byte

				if data[off] == 0xfb {
					text = []byte("NULL")
					off++
				} else {
					var err error
					var complete bool
					text, off, complete, err = readLstring(data, off)
					if err != nil || !complete {
						logp.Debug("mysql", "Error parsing rows: %s %b", err, complete)
						// nevertheless, return what we have so far
						return fields, rows
					}
				}

				if rowLen < mysql.maxRowLength {
					if rowLen+len(text) > mysql.maxRowLength {
						text = text[:mysql.maxRowLength-rowLen]
					}
					row = append(row, string(text))
					rowLen += len(text)
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

func (mysql *mysqlPlugin) publishTransaction(t *mysqlTransaction) {
	if mysql.results == nil {
		return
	}

	logp.Debug("mysql", "mysql.results exists")

	fields := common.MapStr{}
	fields["type"] = "mysql"

	if t.mysql["iserror"].(bool) {
		fields["status"] = common.ERROR_STATUS
	} else {
		fields["status"] = common.OK_STATUS
	}

	fields["responsetime"] = t.responseTime
	if mysql.sendRequest {
		fields["request"] = t.requestRaw
	}
	if mysql.sendResponse {
		fields["response"] = t.responseRaw
	}
	fields["method"] = t.method
	fields["query"] = t.query
	fields["param"] = t.param
	fields["mysql"] = t.mysql
	fields["path"] = t.path
	fields["bytes_out"] = t.bytesOut
	fields["bytes_in"] = t.bytesIn

	if len(t.notes) > 0 {
		fields["notes"] = t.notes
	}

	fields["src"] = &t.src
	fields["dst"] = &t.dst

	mysql.results(beat.Event{
		Timestamp: t.ts,
		Fields:    fields,
	})
}

func readLstring(data []byte, offset int) ([]byte, int, bool, error) {
	length, off, complete, err := readLinteger(data, offset)
	if err != nil {
		return nil, 0, false, err
	}
	if !complete || len(data[off:]) < int(length) {
		return nil, 0, false, nil
	}

	return data[off : off+int(length)], off + int(length), true, nil
}
func readLinteger(data []byte, offset int) (uint64, int, bool, error) {
	if len(data) < offset+1 {
		return 0, 0, false, nil
	}
	switch data[offset] {
	case 0xfe:
		if len(data[offset+1:]) < 8 {
			return 0, 0, false, nil
		}
		return uint64(data[offset+1]) | uint64(data[offset+2])<<8 |
				uint64(data[offset+3])<<16 | uint64(data[offset+4])<<24 |
				uint64(data[offset+5])<<32 | uint64(data[offset+6])<<40 |
				uint64(data[offset+7])<<48 | uint64(data[offset+8])<<56,
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
func readLength(data []byte, offset int) (int, error) {
	if len(data[offset:]) < 3 {
		return 0, errors.New("Data too small to contain a valid length")
	}
	length := uint32(data[offset]) |
		uint32(data[offset+1])<<8 |
		uint32(data[offset+2])<<16
	return int(length), nil
}
