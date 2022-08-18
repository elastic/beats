package oracle

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/packetbeat/pb"
	"github.com/elastic/beats/v7/packetbeat/procs"
	"github.com/elastic/beats/v7/packetbeat/protos"
	"github.com/elastic/beats/v7/packetbeat/protos/tcp"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

// type PacketType uint8

var (
	unmatchedRequests  = monitoring.NewInt(nil, "mysql.unmatched_requests")
	unmatchedResponses = monitoring.NewInt(nil, "mysql.unmatched_responses")
)

// Packet types
const (
	CONNECT  = 1
	ACCEPT   = 2
	ACK      = 3
	REFUSE   = 4
	REDIRECT = 5
	DATA     = 6
	NULL     = 7
	ABORT    = 9
	RESEND   = 11
	MARKER   = 12
	ATTN     = 13
	CTRL     = 14
	HIGHEST  = 19
)

const maxPayloadSize = 100 * 1024

const (
	oracleCmdQuery       = 3
	oracleCmdStmtPrepare = 22
	oracleCmdStmtExecute = 23
	oracleCmdStmtClose   = 25
)

type oracleMessage struct {
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

type oracleTransaction struct {
	tuple    common.TCPTuple
	src      common.Endpoint
	dst      common.Endpoint
	ts       time.Time
	endTime  time.Time
	query    string
	method   string
	path     string // for oracle, Path refers to the oracle table queried
	bytesOut uint64
	bytesIn  uint64
	notes    []string
	isError  bool

	oracle mapstr.M

	requestRaw  string
	responseRaw string

	statementID int      // for prepare statement
	params      []string // for execute statement param
}

type oracleStream struct {
	//sessionCtx SessionContext
	data       []byte
	dataOffset int
	parseState parseState
	length     uint16
	packetType uint8
	flag       uint16
	//NSPFSID    int
	//buffer     []byte
	//SID        []byte

	isClient bool

	message *oracleMessage
}

type parseState int

const (
	oracleStateStart parseState = iota
	oracleStateEatMessage
	oracleStateEatFields
	oracleStateEatRows

	oracleStateMax
)

type oraclePlugin struct {

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
	handleOracle func(oracle *oraclePlugin, m *oracleMessage, tcp *common.TCPTuple,
		dir uint8, raw_msg []byte)
}

func init() {
	protos.Register("oracle", New)
}

func New(
	testMode bool,
	results protos.Reporter,
	watcher procs.ProcessesWatcher,
	cfg *conf.C,
) (protos.Plugin, error) {
	p := &oraclePlugin{}
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

func (oracle *oraclePlugin) init(results protos.Reporter, watcher procs.ProcessesWatcher, config *oracleConfig) error {
	oracle.setFromConfig(config)

	oracle.transactions = common.NewCache(
		oracle.transactionTimeout,
		protos.DefaultTransactionHashSize)
	oracle.transactions.StartJanitor(oracle.transactionTimeout)

	// prepare statements cache
	oracle.prepareStatements = common.NewCache(
		oracle.prepareStatementTimeout,
		protos.DefaultTransactionHashSize)
	oracle.prepareStatements.StartJanitor(oracle.prepareStatementTimeout)

	oracle.handleOracle = handleOracle
	oracle.results = results
	oracle.watcher = watcher

	return nil
}

func (oracle *oraclePlugin) setFromConfig(config *oracleConfig) {
	oracle.ports = config.Ports
	oracle.maxRowLength = config.MaxRowLength
	oracle.maxStoreRows = config.MaxRows
	oracle.sendRequest = config.SendRequest
	oracle.sendResponse = config.SendResponse
	oracle.transactionTimeout = config.TransactionTimeout
	oracle.prepareStatementTimeout = config.StatementTimeout
}

func (oracle *oraclePlugin) getTransaction(k common.HashableTCPTuple) *oracleTransaction {
	v := oracle.transactions.Get(k)
	if v != nil {
		return v.(*oracleTransaction)
	}
	return nil
}

// cache the prepare statement info
type oracleStmtData struct {
	query           string
	numOfParameters int
	nparamType      []uint8
}

type oracleStmtMap map[int]*oracleStmtData

func (oracle *oraclePlugin) getStmtsMap(k common.HashableTCPTuple) oracleStmtMap {
	v := oracle.prepareStatements.Get(k)
	if v != nil {
		return v.(oracleStmtMap)
	}
	return nil
}

func (oracle *oraclePlugin) GetPorts() []int {
	return oracle.ports
}

func (stream *oracleStream) prepareForNewMessage() {
	stream.data = stream.data[stream.dataOffset:]
	stream.parseState = oracleStateStart
	stream.dataOffset = 0
	stream.message = nil
}

func (oracle *oraclePlugin) isServerPort(port uint16) bool {
	for _, sPort := range oracle.ports {
		if uint16(sPort) == port {
			return true
		}
	}
	return false
}

func isRequest(typ uint8) bool {
	if typ == CONNECT {
		return true
	}
	return false
}

func oracleMessageParser(s *oracleStream) (bool, bool) {
	logp.Debug("oracledetailed", "Oracle parser called. parseState = %s", s.parseState)

	m := s.message
	for s.dataOffset < len(s.data) {
		switch s.parseState {
		case oracleStateStart:
			m.start = s.dataOffset
			if len(s.data[s.dataOffset:]) < 5 {
				logp.Warn("MySQL Message too short. Ignore it.")
				return false, false
			}
			hdr := s.data[s.dataOffset : s.dataOffset+5]
			m.packetLength = leUint24(hdr[0:3])
			m.seq = hdr[3]
			m.typ = hdr[4]

			logp.Debug("mysqldetailed", "MySQL Header: Packet length %d, Seq %d, Type=%d isClient=%v", m.packetLength, m.seq, m.typ, s.isClient)

			if s.isClient {
				// starts Command Phase

				if m.seq == 0 && isRequest(m.typ) {
					// parse request
					m.isRequest = true
					m.start = s.dataOffset
					s.parseState = oracleStateEatMessage
				} else {
					// ignore command
					m.ignoreMessage = true
					s.parseState = oracleStateEatMessage
				}
			} else if !s.isClient {
				// parse response
				m.isRequest = false

				if hdr[4] == 0x00 || hdr[4] == 0xfe {
					logp.Debug("mysqldetailed", "Received OK response")
					m.start = s.dataOffset
					s.parseState = oracleStateEatMessage
					m.isOK = true
				} else if hdr[4] == 0xff {
					logp.Debug("mysqldetailed", "Received ERR response")
					m.start = s.dataOffset
					s.parseState = oracleStateEatMessage
					m.isError = true
				} else if m.packetLength == 1 {
					logp.Debug("mysqldetailed", "Query response. Number of fields %d", hdr[4])
					m.numberOfFields = int(hdr[4])
					m.start = s.dataOffset
					s.dataOffset += 5
					s.parseState = oracleStateEatFields
				} else {
					// something else. ignore
					m.ignoreMessage = true
					s.parseState = oracleStateEatMessage
				}
			}

		case oracleStateEatMessage:
			if len(s.data[s.dataOffset:]) < int(m.packetLength)+4 {
				// wait for more data
				return true, false
			}

			s.dataOffset += 4 // header
			s.dataOffset += int(m.packetLength)
			m.end = s.dataOffset
			if m.isRequest {
				// get the statement id
				if m.typ == oracleCmdStmtExecute || m.typ == oracleCmdStmtClose {
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
				len(s.data[s.dataOffset:]))

			// PREPARE_OK packet for Prepared Statement
			// a trick for classify special OK packet
			if m.isOK && m.packetLength == 12 {
				m.statementID = int(binary.LittleEndian.Uint32(s.data[m.start+5:]))
				m.numberOfFields = int(binary.LittleEndian.Uint16(s.data[m.start+9:]))
				m.numberOfParams = int(binary.LittleEndian.Uint16(s.data[m.start+11:]))
				if m.numberOfFields > 0 {
					s.parseState = oracleStateEatFields
				} else if m.numberOfParams > 0 {
					s.parseState = oracleStateEatRows
				}
			} else {
				return true, true
			}
		case oracleStateEatFields:
			if len(s.data[s.dataOffset:]) < 4 {
				// wait for more
				return true, false
			}

			hdr := s.data[s.dataOffset : s.dataOffset+4]
			m.packetLength = leUint24(hdr[:3])
			m.seq = hdr[3]

			if len(s.data[s.dataOffset:]) >= int(m.packetLength)+4 {
				s.dataOffset += 4 // header

				if s.data[s.dataOffset] == 0xfe {
					logp.Debug("mysqldetailed", "Received EOF packet")
					// EOF marker
					s.dataOffset += int(m.packetLength)

					s.parseState = oracleStateEatRows
				} else {
					_ /* catalog */, off, complete, err := readLstring(s.data, s.dataOffset)
					if !complete {
						return true, false
					}
					if err != nil {
						logp.Debug("oracle", "Error on read_lstring: %s", err)
						return false, false
					}
					db /*schema */, off, complete, err := readLstring(s.data, off)
					if !complete {
						return true, false
					}
					if err != nil {
						logp.Debug("oracle", "Error on read_lstring: %s", err)
						return false, false
					}
					table /* table */, _ /*off*/, complete, err := readLstring(s.data, off)
					if !complete {
						return true, false
					}
					if err != nil {
						logp.Debug("oracle", "Error on read_lstring: %s", err)
						return false, false
					}

					dbTable := string(db) + "." + string(table)

					if len(m.tables) == 0 {
						m.tables = dbTable
					} else if !strings.Contains(m.tables, dbTable) {
						m.tables = m.tables + ", " + dbTable
					}
					logp.Debug("oracledetailed", "db=%s, table=%s", db, table)
					s.dataOffset += int(m.packetLength)
					// go to next field
				}
			} else {
				// wait for more
				return true, false
			}

		case oracleStateEatRows:
			if len(s.data[s.dataOffset:]) < 4 {
				// wait for more
				return true, false
			}
			hdr := s.data[s.dataOffset : s.dataOffset+4]
			m.packetLength = leUint24(hdr[:3])
			m.seq = hdr[3]

			logp.Debug("mysqldetailed", "Rows: packet length %d, packet number %d", m.packetLength, m.seq)

			if len(s.data[s.dataOffset:]) < int(m.packetLength)+4 {
				// wait for more
				return true, false
			}

			s.dataOffset += 4 // header

			if s.data[s.dataOffset] == 0xfe {
				logp.Debug("mysqldetailed", "Received EOF packet")
				// EOF marker
				s.dataOffset += int(m.packetLength)

				if m.end == 0 {
					m.end = s.dataOffset
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

			s.dataOffset += int(m.packetLength)
			if m.end == 0 && s.dataOffset > maxPayloadSize {
				// only send up to here, but read until the end
				m.end = s.dataOffset
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
func (oracle *oraclePlugin) messageGap(s *oracleStream, nbytes int) (complete bool) {
	m := s.message
	switch s.parseState {
	case oracleStateStart, oracleStateEatMessage:
		// not enough data yet to be useful
		return false
	case oracleStateEatFields, oracleStateEatRows:
		// enough data here
		m.end = int(s.dataOffset)
		if m.isRequest {
			m.notes = append(m.notes, "Packet loss while capturing the request")
		} else {
			m.notes = append(m.notes, "Packet loss while capturing the response")
		}
		return true
	}

	return true
}

type oraclePrivateData struct {
	data [2]*oracleStream
}

// Called when the parser has identified a full message.
func (oracle *oraclePlugin) messageComplete(tcptuple *common.TCPTuple, dir uint8, stream *oracleStream) {
	// all ok, ship it
	msg := stream.data[stream.message.start:stream.message.end]

	if !stream.message.ignoreMessage {
		oracle.handleOracle(oracle, stream.message, tcptuple, dir, msg)
	}

	// and reset message
	stream.prepareForNewMessage()
}

func (oracle *oraclePlugin) ConnectionTimeout() time.Duration {
	return oracle.transactionTimeout
}

func (oracle *oraclePlugin) Parse(pkt *protos.Packet, tcptuple *common.TCPTuple,
	dir uint8, private protos.ProtocolData,
) protos.ProtocolData {
	defer logp.Recover("ParseMysql exception")

	priv := oraclePrivateData{}
	if private != nil {
		var ok bool
		priv, ok = private.(oraclePrivateData)
		if !ok {
			priv = oraclePrivateData{}
		}
	}

	if priv.data[dir] == nil {
		dstPort := tcptuple.DstPort
		if dir == tcp.TCPDirectionReverse {
			dstPort = tcptuple.SrcPort
		}
		priv.data[dir] = &oracleStream{
			data:     pkt.Payload,
			message:  &oracleMessage{ts: pkt.Ts},
			isClient: oracle.isServerPort(dstPort),
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
			stream.message = &oracleMessage{ts: pkt.Ts}
		}

		ok, complete := oracleMessageParser(priv.data[dir])
		logp.Debug("mysqldetailed", "mysqlMessageParser returned ok=%v complete=%v", ok, complete)
		if !ok {
			// drop this tcp stream. Will retry parsing with the next
			// segment in it
			priv.data[dir] = nil
			logp.Debug("mysql", "Ignore MySQL message. Drop tcp stream. Try parsing with the next segment")
			return priv
		}

		if complete {
			oracle.messageComplete(tcptuple, dir, stream)
		} else {
			// wait for more data
			break
		}
	}
	return priv
}

func (oracle *oraclePlugin) GapInStream(tcptuple *common.TCPTuple, dir uint8,
	nbytes int, private protos.ProtocolData) (priv protos.ProtocolData, drop bool,
) {
	defer logp.Recover("GapInStream(mysql) exception")

	if private == nil {
		return private, false
	}
	oracleData, ok := private.(oraclePrivateData)
	if !ok {
		return private, false
	}
	stream := oracleData.data[dir]
	if stream == nil || stream.message == nil {
		// nothing to do
		return private, false
	}

	if oracle.messageGap(stream, nbytes) {
		// we need to publish from here
		oracle.messageComplete(tcptuple, dir, stream)
	}

	// we always drop the TCP stream. Because it's binary and len based,
	// there are too few cases in which we could recover the stream (maybe
	// for very large blobs, leaving that as TODO)
	return private, true
}

func (oracle *oraclePlugin) ReceivedFin(tcptuple *common.TCPTuple, dir uint8,
	private protos.ProtocolData,
) protos.ProtocolData {
	// TODO: check if we have data pending and either drop it to free
	// memory or send it up the stack.
	return private
}

func handleOracle(oracle *oraclePlugin, m *oracleMessage, tcptuple *common.TCPTuple,
	dir uint8, rawMsg []byte,
) {
	m.tcpTuple = *tcptuple
	m.direction = dir
	m.cmdlineTuple = oracle.watcher.FindProcessesTupleTCP(tcptuple.IPPort())
	m.raw = rawMsg

	if m.isRequest {
		oracle.receivedOracleRequest(m)
	} else {
		oracle.receivedOracleResponse(m)
	}
}

func (mysql *oraclePlugin) receivedOracleRequest(msg *oracleMessage) {
	tuple := msg.tcpTuple
	trans := mysql.getTransaction(tuple.Hashable())
	if trans != nil {
		if trans.oracle != nil {
			logp.Debug("mysql", "Two requests without a Response. Dropping old request: %s", trans.oracle)
			unmatchedRequests.Add(1)
		}
	} else {
		trans = &oracleTransaction{tuple: tuple}
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
			case oracleCmdStmtExecute:
				trans.query = "Request Execute Statement"
			case oracleCmdStmtClose:
				trans.query = "Request Close Statement"
			}
			trans.notes = append(trans.notes, "The actual query being used is unknown")
			trans.requestRaw = msg.query
			trans.bytesIn = msg.size
			return
		}
		switch msg.typ {
		case oracleCmdStmtExecute:
			if value, ok := stmts[trans.statementID]; ok {
				trans.query = value.query
				// parse parameters
				trans.params = mysql.parseOracleExecuteStatement(msg.raw, value)
			} else {
				trans.query = "Request Execute Statement"
				trans.notes = append(trans.notes, "The actual query being used is unknown")
				trans.requestRaw = msg.query
				trans.bytesIn = msg.size
				return
			}
		case oracleCmdStmtClose:
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

	trans.oracle = mapstr.M{}

	trans.notes = msg.notes

	// save Raw message
	trans.requestRaw = msg.query
	trans.bytesIn = msg.size
}

func (oracle *oraclePlugin) receivedOracleResponse(msg *oracleMessage) {
	trans := oracle.getTransaction(msg.tcpTuple.Hashable())
	if trans == nil {
		logp.Debug("oracle", "Response from unknown transaction. Ignoring.")
		unmatchedResponses.Add(1)
		return
	}
	// check if the request was received
	if trans.oracle == nil {
		logp.Debug("mysql", "Response from unknown transaction. Ignoring.")
		unmatchedResponses.Add(1)
		return

	}
	// save json details
	trans.oracle.Update(mapstr.M{
		"affected_rows": msg.affectedRows,
		"insert_id":     msg.insertID,
		"num_rows":      msg.numberOfRows,
		"num_fields":    msg.numberOfFields,
	})
	trans.isError = msg.isError
	if trans.isError {
		trans.oracle["error_code"] = msg.errorCode
		trans.oracle["error_message"] = msg.errorInfo
	}
	if msg.statementID != 0 {
		// cache prepare statement response info
		stmts := oracle.getStmtsMap(msg.tcpTuple.Hashable())
		if stmts == nil {
			stmts = oracleStmtMap{}
		}
		if stmts[msg.statementID] == nil {
			stmtData := &oracleStmtData{
				query:           trans.query,
				numOfParameters: msg.numberOfParams,
			}
			stmts[msg.statementID] = stmtData
		}
		oracle.prepareStatements.Put(msg.tcpTuple.Hashable(), stmts)
		trans.notes = append(trans.notes, trans.query)
		trans.query = "Request Prepare Statement"
	}

	trans.bytesOut = msg.size
	trans.path = msg.tables
	trans.endTime = msg.ts

	// save Raw message
	if len(msg.raw) > 0 {
		fields, rows := oracle.parseOracleResponse(msg.raw)

		trans.responseRaw = common.DumpInCSVFormat(fields, rows)
	}

	trans.notes = append(trans.notes, msg.notes...)

	oracle.publishTransaction(trans)
	oracle.transactions.Delete(trans.tuple.Hashable())

	logp.Debug("oracle", "Oracle transaction completed: %s %s %s", trans.query, trans.params, trans.oracle)
}

func (oracle *oraclePlugin) parseOracleExecuteStatement(data []byte, stmtdata *oracleStmtData) []string {
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
	// oracle hdr
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
		logp.Debug("oracle", "Data too small")
		return nil
	}
	stmtBound := data[offset]
	offset++
	paramOffset := offset
	if stmtBound == 1 {
		paramOffset += nparam * 2
		if dataLen <= paramOffset {
			logp.Debug("oracle", "Data too small to contain parameters")
			return nil
		}
		// First call or rebound (1)
		for stmtPos := 0; stmtPos < nparam; stmtPos++ {
			paramType = uint8(data[offset])
			offset++
			nparamType = append(nparamType, paramType)
			logp.Debug("oracledetailed", "type = %d", paramType)
			paramUnsigned = uint8(data[offset])
			offset++
			if paramUnsigned != 0 {
				logp.Debug("oracle", "Illegal param unsigned")
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
				logp.Debug("oracle", "Data too small")
				return nil
			}
			valueString := strconv.Itoa(int(binary.LittleEndian.Uint16(data[paramOffset:])))
			paramString = append(paramString, valueString)
			paramOffset += 2
		// FIELD_TYPE_LONG
		case 0x03:
			if dataLen < paramOffset+4 {
				logp.Debug("oracle", "Data too small")
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
				logp.Debug("oracle", "Data too small")
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
				logp.Debug("oracle", "Data too small")
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
				logp.Debug("oracle", "Data too small")
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
					logp.Debug("oracle", "Data too small")
					return nil
				}
				paramOffset += 2
				paramString = append(paramString, string(data[paramOffset:paramOffset+paramLen16]))
				paramOffset += paramLen16
			case 0xfd: /* 64k - 16M chars */
				paramLen24 := int(leUint24(data[paramOffset : paramOffset+3]))
				if dataLen < paramOffset+paramLen24+3 {
					logp.Debug("oracle", "Data too small")
					return nil
				}
				paramOffset += 3
				paramString = append(paramString, string(data[paramOffset:paramOffset+paramLen24]))
				paramOffset += paramLen24
			default: /* < 252 chars     */
				if dataLen < paramOffset+paramLen {
					logp.Debug("oracle", "Data too small")
					return nil
				}
				paramString = append(paramString, string(data[paramOffset:paramOffset+paramLen]))
				paramOffset += paramLen
			}
		default:
			logp.Debug("oracle", "Unknown param type")
			return nil
		}

	}
	return paramString
}

func (oracle *oraclePlugin) parseOracleResponse(data []byte) ([]string, [][]string) {
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

		logp.Debug("oracle", "Data len: %d", len(data))

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
				logp.Debug("oracle", "Reading field: %v %v", err, complete)
				return fields, rows
			}
			_ /*database*/, off, complete, err = readLstring(data, off)
			if err != nil || !complete {
				logp.Debug("oracle", "Reading field: %v %v", err, complete)
				return fields, rows
			}
			_ /*table*/, off, complete, err = readLstring(data, off)
			if err != nil || !complete {
				logp.Debug("oracle", "Reading field: %v %v", err, complete)
				return fields, rows
			}
			_ /*org table*/, off, complete, err = readLstring(data, off)
			if err != nil || !complete {
				logp.Debug("oracle", "Reading field: %v %v", err, complete)
				return fields, rows
			}
			name, off, complete, err := readLstring(data, off)
			if err != nil || !complete {
				logp.Debug("oracle", "Reading field: %v %v", err, complete)
				return fields, rows
			}
			_ /* org name */, _ /*off*/, complete, err = readLstring(data, off)
			if err != nil || !complete {
				logp.Debug("oracle", "Reading field: %v %v", err, complete)
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
						logp.Debug("oracle", "Error parsing rows: %+v %t", err, complete)
						// nevertheless, return what we have so far
						return fields, rows
					}
				}

				if rowLen < oracle.maxRowLength {
					if rowLen+len(text) > oracle.maxRowLength {
						text = text[:oracle.maxRowLength-rowLen]
					}
					row = append(row, string(text))
					rowLen += len(text)
				}
			}

			logp.Debug("oracledetailed", "Append row: %v", row)

			rows = append(rows, row)
			if len(rows) >= oracle.maxStoreRows {
				break
			}

			offset += length + 4
		}
	}
	return fields, rows
}

func (oracle *oraclePlugin) publishTransaction(t *oracleTransaction) {
	if oracle.results == nil {
		return
	}

	logp.Debug("oracle", "oracle.results exists")

	evt, pbf := pb.NewBeatEvent(t.ts)
	pbf.SetSource(&t.src)
	pbf.AddIP(t.src.IP)
	pbf.SetDestination(&t.dst)
	pbf.AddIP(t.dst.IP)
	pbf.Source.Bytes = int64(t.bytesIn)
	pbf.Destination.Bytes = int64(t.bytesOut)
	pbf.Event.Dataset = "oracle"
	pbf.Event.Start = t.ts
	pbf.Event.End = t.endTime
	pbf.Network.Transport = "tcp"
	pbf.Network.Protocol = "oracle"
	pbf.Error.Message = t.notes

	fields := evt.Fields
	fields["type"] = pbf.Event.Dataset
	fields["method"] = t.method
	fields["query"] = t.query
	fields["oracle"] = t.oracle
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

	if oracle.sendRequest {
		fields["request"] = t.requestRaw
	}
	if oracle.sendResponse {
		fields["response"] = t.responseRaw
	}

	oracle.results(evt)
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
