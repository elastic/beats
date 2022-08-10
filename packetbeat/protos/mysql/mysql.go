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

package mysql

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/monitoring"

	"github.com/elastic/beats/v7/packetbeat/pb"
	"github.com/elastic/beats/v7/packetbeat/procs"
	"github.com/elastic/beats/v7/packetbeat/protos"
	"github.com/elastic/beats/v7/packetbeat/protos/tcp"
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
	cmdlineTuple *common.ProcessTuple
	raw          []byte
	notes        []string

	statementID    int
	numberOfParams int
}

type mysqlTransaction struct {
	tuple    common.TCPTuple
	src      common.Endpoint
	dst      common.Endpoint
	ts       time.Time
	endTime  time.Time
	query    string
	method   string
	path     string // for mysql, Path refers to the mysql table queried
	bytesOut uint64
	bytesIn  uint64
	notes    []string
	isError  bool

	mysql common.MapStr

	requestRaw  string
	responseRaw string

	statementID int      // for prepare statement
	params      []string // for execute statement param
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

	// prepare statements cache
	prepareStatements       *common.Cache
	prepareStatementTimeout time.Duration

	results protos.Reporter
	watcher procs.ProcessesWatcher

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
	watcher procs.ProcessesWatcher,
	cfg *common.Config,
) (protos.Plugin, error) {
	p := &mysqlPlugin{}
	config := defaultConfig
	if !testMode {
		if err := cfg.Unpack(&config); err != nil {
			return nil, err
		}
	}

	if err := p.init(results, watcher, &config); err != nil {
		return nil, err
	}
	return p, nil
}

func (mysql *mysqlPlugin) init(results protos.Reporter, watcher procs.ProcessesWatcher, config *mysqlConfig) error {
	mysql.setFromConfig(config)

	mysql.transactions = common.NewCache(
		mysql.transactionTimeout,
		protos.DefaultTransactionHashSize)
	mysql.transactions.StartJanitor(mysql.transactionTimeout)

	// prepare statements cache
	mysql.prepareStatements = common.NewCache(
		mysql.prepareStatementTimeout,
		protos.DefaultTransactionHashSize)
	mysql.prepareStatements.StartJanitor(mysql.prepareStatementTimeout)

	mysql.handleMysql = handleMysql
	mysql.results = results
	mysql.watcher = watcher

	return nil
}

func (mysql *mysqlPlugin) setFromConfig(config *mysqlConfig) {
	mysql.ports = config.Ports
	mysql.maxRowLength = config.MaxRowLength
	mysql.maxStoreRows = config.MaxRows
	mysql.sendRequest = config.SendRequest
	mysql.sendResponse = config.SendResponse
	mysql.transactionTimeout = config.TransactionTimeout
	mysql.prepareStatementTimeout = config.StatementTimeout
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
	v := mysql.prepareStatements.Get(k)
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
	stream.message = nil
}

func (mysql *mysqlPlugin) isServerPort(port uint16) bool {
	for _, sPort := range mysql.ports {
		if uint16(sPort) == port {
			return true
		}
	}
	return false
}

func isRequest(typ uint8) bool {
	if typ == mysqlCmdQuery || typ == mysqlCmdStmtPrepare ||
		typ == mysqlCmdStmtExecute || typ == mysqlCmdStmtClose {
		return true
	}
	return false
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
			m.packetLength = leUint24(hdr[0:3])
			m.seq = hdr[3]
			m.typ = hdr[4]

			logp.Debug("mysqldetailed", "MySQL Header: Packet length %d, Seq %d, Type=%d isClient=%v", m.packetLength, m.seq, m.typ, s.isClient)

			if s.isClient {
				// starts Command Phase

				if m.seq == 0 && isRequest(m.typ) {
					// parse request
					m.isRequest = true
					m.start = s.parseOffset
					s.parseState = mysqlStateEatMessage
				} else {
					// ignore command
					m.ignoreMessage = true
					s.parseState = mysqlStateEatMessage
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
			}

		case mysqlStateEatMessage:
			if len(s.data[s.parseOffset:]) < int(m.packetLength)+4 {
				// wait for more data
				return true, false
			}

			s.parseOffset += 4 // header
			s.parseOffset += int(m.packetLength)
			m.end = s.parseOffset
			if m.isRequest {
				// get the statement id
				if m.typ == mysqlCmdStmtExecute || m.typ == mysqlCmdStmtClose {
					m.statementID = int(binary.LittleEndian.Uint32(s.data[m.start+5:]))
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
				m.errorCode = binary.LittleEndian.Uint16(s.data[m.start+5 : m.start+7])

				m.errorInfo = string(s.data[m.start+8:m.start+13]) + ": " + string(s.data[m.start+13:])
			}
			m.size = uint64(m.end - m.start)
			logp.Debug("mysqldetailed", "Message complete. remaining=%d",
				len(s.data[s.parseOffset:]))

			// PREPARE_OK packet for Prepared Statement
			// a trick for classify special OK packet
			if m.isOK && m.packetLength == 12 {
				m.statementID = int(binary.LittleEndian.Uint32(s.data[m.start+5:]))
				m.numberOfFields = int(binary.LittleEndian.Uint16(s.data[m.start+9:]))
				m.numberOfParams = int(binary.LittleEndian.Uint16(s.data[m.start+11:]))
				if m.numberOfFields > 0 {
					s.parseState = mysqlStateEatFields
				} else if m.numberOfParams > 0 {
					s.parseState = mysqlStateEatRows
				}
			} else {
				return true, true
			}
		case mysqlStateEatFields:
			if len(s.data[s.parseOffset:]) < 4 {
				// wait for more
				return true, false
			}

			hdr := s.data[s.parseOffset : s.parseOffset+4]
			m.packetLength = leUint24(hdr[:3])
			m.seq = hdr[3]

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
			m.packetLength = leUint24(hdr[:3])
			m.seq = hdr[3]

			logp.Debug("mysqldetailed", "Rows: packet length %d, packet number %d", m.packetLength, m.seq)

			if len(s.data[s.parseOffset:]) < int(m.packetLength)+4 {
				// wait for more
				return true, false
			}

			s.parseOffset += 4 // header

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
	dir uint8, private protos.ProtocolData,
) protos.ProtocolData {
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
		dstPort := tcptuple.DstPort
		if dir == tcp.TCPDirectionReverse {
			dstPort = tcptuple.SrcPort
		}
		priv.data[dir] = &mysqlStream{
			data:     pkt.Payload,
			message:  &mysqlMessage{ts: pkt.Ts},
			isClient: mysql.isServerPort(dstPort),
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
		logp.Debug("mysqldetailed", "mysqlMessageParser returned ok=%v complete=%v", ok, complete)
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
	nbytes int, private protos.ProtocolData) (priv protos.ProtocolData, drop bool,
) {
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
	private protos.ProtocolData,
) protos.ProtocolData {
	// TODO: check if we have data pending and either drop it to free
	// memory or send it up the stack.
	return private
}

func handleMysql(mysql *mysqlPlugin, m *mysqlMessage, tcptuple *common.TCPTuple,
	dir uint8, rawMsg []byte,
) {
	m.tcpTuple = *tcptuple
	m.direction = dir
	m.cmdlineTuple = mysql.watcher.FindProcessesTupleTCP(tcptuple.IPPort())
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
	trans.src, trans.dst = common.MakeEndpointPair(msg.tcpTuple.BaseTuple, msg.cmdlineTuple)
	if msg.direction == tcp.TCPDirectionReverse {
		trans.src, trans.dst = trans.dst, trans.src
	}

	// try to get query string for Execute statement from cache
	// and delete statement id for Close statement from cache
	if msg.statementID != 0 {
		trans.statementID = msg.statementID
		stmts := mysql.getStmtsMap(msg.tcpTuple.Hashable())
		if stmts == nil {
			switch msg.typ {
			case mysqlCmdStmtExecute:
				trans.query = "Request Execute Statement"
			case mysqlCmdStmtClose:
				trans.query = "Request Close Statement"
			}
			trans.notes = append(trans.notes, "The actual query being used is unknown")
			trans.requestRaw = msg.query
			trans.bytesIn = msg.size
			return
		}
		switch msg.typ {
		case mysqlCmdStmtExecute:
			if value, ok := stmts[trans.statementID]; ok {
				trans.query = value.query
				// parse parameters
				trans.params = mysql.parseMysqlExecuteStatement(msg.raw, value)
			} else {
				trans.query = "Request Execute Statement"
				trans.notes = append(trans.notes, "The actual query being used is unknown")
				trans.requestRaw = msg.query
				trans.bytesIn = msg.size
				return
			}
		case mysqlCmdStmtClose:
			delete(stmts, trans.statementID)
			trans.query = "CmdStmtClose"
			mysql.transactions.Delete(tuple.Hashable())
		}
	} else {
		trans.query = msg.query
	}

	// Extract the method, by simply taking the first word and
	// making it upper case.
	query := strings.Trim(trans.query, " \r\n\t")
	index := strings.IndexAny(query, " \r\n\t")
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
	})
	trans.isError = msg.isError
	if trans.isError {
		trans.mysql["error_code"] = msg.errorCode
		trans.mysql["error_message"] = msg.errorInfo
	}
	if msg.statementID != 0 {
		// cache prepare statement response info
		stmts := mysql.getStmtsMap(msg.tcpTuple.Hashable())
		if stmts == nil {
			stmts = mysqlStmtMap{}
		}
		if stmts[msg.statementID] == nil {
			stmtData := &mysqlStmtData{
				query:           trans.query,
				numOfParameters: msg.numberOfParams,
			}
			stmts[msg.statementID] = stmtData
		}
		mysql.prepareStatements.Put(msg.tcpTuple.Hashable(), stmts)
		trans.notes = append(trans.notes, trans.query)
		trans.query = "Request Prepare Statement"
	}

	trans.bytesOut = msg.size
	trans.path = msg.tables
	trans.endTime = msg.ts

	// save Raw message
	if len(msg.raw) > 0 {
		fields, rows := mysql.parseMysqlResponse(msg.raw)

		trans.responseRaw = common.DumpInCSVFormat(fields, rows)
	}

	trans.notes = append(trans.notes, msg.notes...)

	mysql.publishTransaction(trans)
	mysql.transactions.Delete(trans.tuple.Hashable())

	logp.Debug("mysql", "Mysql transaction completed: %s %s %s", trans.query, trans.params, trans.mysql)
}

func (mysql *mysqlPlugin) parseMysqlExecuteStatement(data []byte, stmtdata *mysqlStmtData) []string {
	dataLen := len(data)
	if dataLen < 14 {
		logp.Debug("mysql", "Data too small")
		return nil
	}
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
	} else {
		return nil
	}
	// stmt bound
	if dataLen <= offset {
		logp.Debug("mysql", "Data too small")
		return nil
	}
	stmtBound := data[offset]
	offset++
	paramOffset := offset
	if stmtBound == 1 {
		paramOffset += nparam * 2
		if dataLen <= paramOffset {
			logp.Debug("mysql", "Data too small to contain parameters")
			return nil
		}
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
				return nil
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
			return nil
		}
	}

	for stmtPos := 0; stmtPos < nparam; stmtPos++ {
		paramType = nparamType[stmtPos]
		// dissect parameter on paramType
		switch paramType {
		// FIELD_TYPE_TINY
		case 0x01:
			valueString := strconv.Itoa(int(data[paramOffset]))
			paramString = append(paramString, valueString)
			paramOffset++
		// FIELD_TYPE_SHORT
		case 0x02:
			if dataLen < paramOffset+2 {
				logp.Debug("mysql", "Data too small")
				return nil
			}
			valueString := strconv.Itoa(int(binary.LittleEndian.Uint16(data[paramOffset:])))
			paramString = append(paramString, valueString)
			paramOffset += 2
		// FIELD_TYPE_LONG
		case 0x03:
			if dataLen < paramOffset+4 {
				logp.Debug("mysql", "Data too small")
				return nil
			}
			valueString := strconv.Itoa(int(binary.LittleEndian.Uint32(data[paramOffset:])))
			paramString = append(paramString, valueString)
			paramOffset += 4
		// FIELD_TYPE_FLOAT
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
			if dataLen < paramOffset+8 {
				logp.Debug("mysql", "Data too small")
				return nil
			}
			valueString := strconv.FormatInt(int64(binary.LittleEndian.Uint64(data[paramOffset:paramOffset+8])), 10)
			paramString = append(paramString, valueString)
			paramOffset += 8
		// FIELD_TYPE_TIMESTAMP
		// FIELD_TYPE_DATETIME
		// FIELD_TYPE_DATE
		case 0x07, 0x0c, 0x0a:
			var year, month, day, hour, minute, second string
			paramLen := int(data[paramOffset])
			if dataLen < paramOffset+paramLen+1 {
				logp.Debug("mysql", "Data too small")
				return nil
			}
			paramOffset++
			if paramLen >= 2 {
				year = strconv.Itoa(int(binary.LittleEndian.Uint16(data[paramOffset:])))
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

			// If paramLen is greater or equal to 11
			// then nanoseconds are also available.
			// We do not handle them.

			datetime := year + "/" + month + "/" + day + " " + hour + ":" + minute + ":" + second
			paramString = append(paramString, datetime)
			paramOffset += paramLen
		// FIELD_TYPE_TIME
		case 0x0b:
			paramLen := int(data[paramOffset])
			if dataLen < paramOffset+paramLen+1 {
				logp.Debug("mysql", "Data too small")
				return nil
			}
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
				paramLen16 := int(binary.LittleEndian.Uint16(data[paramOffset : paramOffset+2]))
				if dataLen < paramOffset+paramLen16+2 {
					logp.Debug("mysql", "Data too small")
					return nil
				}
				paramOffset += 2
				paramString = append(paramString, string(data[paramOffset:paramOffset+paramLen16]))
				paramOffset += paramLen16
			case 0xfd: /* 64k - 16M chars */
				paramLen24 := int(leUint24(data[paramOffset : paramOffset+3]))
				if dataLen < paramOffset+paramLen24+3 {
					logp.Debug("mysql", "Data too small")
					return nil
				}
				paramOffset += 3
				paramString = append(paramString, string(data[paramOffset:paramOffset+paramLen24]))
				paramOffset += paramLen24
			default: /* < 252 chars     */
				if dataLen < paramOffset+paramLen {
					logp.Debug("mysql", "Data too small")
					return nil
				}
				paramString = append(paramString, string(data[paramOffset:paramOffset+paramLen]))
				paramOffset += paramLen
			}
		default:
			logp.Debug("mysql", "Unknown param type")
			return nil
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

	if len(data) < 5 {
		logp.Warn("Invalid response: data less than 5 bytes")
		return []string{}, [][]string{}
	}

	fields := []string{}
	rows := [][]string{}
	switch data[4] {
	case 0x00:
		// OK response
	case 0xff:
		// Error response
	default:
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
				offset += length + 4 // ineffassign
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
						logp.Debug("mysql", "Error parsing rows: %+v %t", err, complete)
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

	evt, pbf := pb.NewBeatEvent(t.ts)
	pbf.SetSource(&t.src)
	pbf.AddIP(t.src.IP)
	pbf.SetDestination(&t.dst)
	pbf.AddIP(t.dst.IP)
	pbf.Source.Bytes = int64(t.bytesIn)
	pbf.Destination.Bytes = int64(t.bytesOut)
	pbf.Event.Dataset = "mysql"
	pbf.Event.Start = t.ts
	pbf.Event.End = t.endTime
	pbf.Network.Transport = "tcp"
	pbf.Network.Protocol = "mysql"
	pbf.Error.Message = t.notes

	fields := evt.Fields
	fields["type"] = pbf.Event.Dataset
	fields["method"] = t.method
	fields["query"] = t.query
	fields["mysql"] = t.mysql
	if len(t.path) > 0 {
		fields["path"] = t.path
	}
	if len(t.params) > 0 {
		fields["params"] = t.params
	}

	if t.isError {
		fields["status"] = common.ERROR_STATUS
	} else {
		fields["status"] = common.OK_STATUS
	}

	if mysql.sendRequest {
		fields["request"] = t.requestRaw
	}
	if mysql.sendResponse {
		fields["response"] = t.responseRaw
	}

	mysql.results(evt)
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

func leUint24(data []byte) uint32 {
	return uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16
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
		return binary.LittleEndian.Uint64(data[offset+1:]),
			offset + 9, true, nil
	case 0xfd:
		if len(data[offset+1:]) < 3 {
			return 0, 0, false, nil
		}
		return uint64(leUint24(data[offset+1 : offset+4])), offset + 4, true, nil
	case 0xfc:
		if len(data[offset+1:]) < 2 {
			return 0, 0, false, nil
		}
		return uint64(binary.LittleEndian.Uint16(data[offset+1:])), offset + 3, true, nil
	}

	if uint64(data[offset]) >= 0xfb {
		return 0, 0, false, fmt.Errorf("unexpected value in read_linteger")
	}

	return uint64(data[offset]), offset + 1, true, nil
}

// Read a mysql length field (3 bytes LE)
func readLength(data []byte, offset int) (int, error) {
	if len(data[offset:]) < 3 {
		return 0, errors.New("data too small to contain a valid length")
	}
	return int(leUint24(data[offset : offset+3])), nil
}
