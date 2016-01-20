package pgsql

import (
	"errors"
	"strings"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/publisher"

	"github.com/elastic/beats/packetbeat/config"
	"github.com/elastic/beats/packetbeat/procs"
	"github.com/elastic/beats/packetbeat/protos"
	"github.com/elastic/beats/packetbeat/protos/tcp"
)

type PgsqlMessage struct {
	start         int
	end           int
	isSSLResponse bool
	isSSLRequest  bool
	toExport      bool

	Ts             time.Time
	IsRequest      bool
	Query          string
	Size           uint64
	Fields         []string
	FieldsFormat   []byte
	Rows           [][]string
	NumberOfRows   int
	NumberOfFields int
	IsOK           bool
	IsError        bool
	ErrorInfo      string
	ErrorCode      string
	ErrorSeverity  string
	Notes          []string

	Direction    uint8
	TcpTuple     common.TcpTuple
	CmdlineTuple *common.CmdlineTuple
}

type PgsqlTransaction struct {
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
	BytesOut     uint64
	BytesIn      uint64
	Notes        []string

	Pgsql common.MapStr

	Request_raw  string
	Response_raw string
}

type PgsqlStream struct {
	tcptuple *common.TcpTuple

	data []byte

	parseOffset       int
	parseState        int
	seenSSLRequest    bool
	expectSSLResponse bool

	message *PgsqlMessage
}

const (
	PgsqlStartState = iota
	PgsqlGetDataState
)

const (
	SSLRequest = iota
	StartupMessage
	CancelRequest
)

var (
	errInvalidLength = errors.New("invalid length")
)

type Pgsql struct {

	// config
	Ports         []int
	maxStoreRows  int
	maxRowLength  int
	Send_request  bool
	Send_response bool

	transactions       *common.Cache
	transactionTimeout time.Duration

	results publisher.Client

	// function pointer for mocking
	handlePgsql func(pgsql *Pgsql, m *PgsqlMessage, tcp *common.TcpTuple,
		dir uint8, raw_msg []byte)
}

func (pgsql *Pgsql) getTransaction(k common.HashableTcpTuple) []*PgsqlTransaction {
	v := pgsql.transactions.Get(k)
	if v != nil {
		return v.([]*PgsqlTransaction)
	}
	return nil
}

func (pgsql *Pgsql) InitDefaults() {
	pgsql.maxRowLength = 1024
	pgsql.maxStoreRows = 10
	pgsql.Send_request = false
	pgsql.Send_response = false
	pgsql.transactionTimeout = protos.DefaultTransactionExpiration
}

func (pgsql *Pgsql) setFromConfig(config config.Pgsql) error {

	pgsql.Ports = config.Ports

	if config.Max_row_length != nil {
		pgsql.maxRowLength = *config.Max_row_length
	}
	if config.Max_rows != nil {
		pgsql.maxStoreRows = *config.Max_rows
	}
	if config.SendRequest != nil {
		pgsql.Send_request = *config.SendRequest
	}
	if config.SendResponse != nil {
		pgsql.Send_response = *config.SendResponse
	}
	if config.TransactionTimeout != nil && *config.TransactionTimeout > 0 {
		pgsql.transactionTimeout = time.Duration(*config.TransactionTimeout) * time.Second
	}
	return nil
}

func (pgsql *Pgsql) GetPorts() []int {
	return pgsql.Ports
}

func (pgsql *Pgsql) Init(test_mode bool, results publisher.Client) error {

	pgsql.InitDefaults()
	if !test_mode {
		err := pgsql.setFromConfig(config.ConfigSingleton.Protocols.Pgsql)
		if err != nil {
			return err
		}
	}

	pgsql.transactions = common.NewCache(
		pgsql.transactionTimeout,
		protos.DefaultTransactionHashSize)
	pgsql.transactions.StartJanitor(pgsql.transactionTimeout)
	pgsql.handlePgsql = handlePgsql
	pgsql.results = results

	return nil
}

func (stream *PgsqlStream) PrepareForNewMessage() {
	stream.data = stream.data[stream.message.end:]
	stream.parseState = PgsqlStartState
	stream.parseOffset = 0
	stream.message = nil
}

// Parse a list of commands separated by semicolon from the query
func pgsqlQueryParser(query string) []string {
	array := strings.Split(query, ";")

	queries := []string{}

	for _, q := range array {
		qt := strings.TrimSpace(q)
		if len(qt) > 0 {
			queries = append(queries, qt)
		}
	}
	return queries
}

// Extract the method from a SQL query
func getQueryMethod(q string) string {

	index := strings.Index(q, " ")
	var method string
	if index > 0 {
		method = strings.ToUpper(q[:index])
	} else {
		method = strings.ToUpper(q)
	}
	return method
}

func pgsqlFieldsParser(s *PgsqlStream) {
	m := s.message

	// read field count (int16)
	field_count := int(common.Bytes_Ntohs(s.data[s.parseOffset : s.parseOffset+2]))
	s.parseOffset += 2
	logp.Debug("pgsqldetailed", "Row Description field count=%d", field_count)

	fields := []string{}
	fields_format := []byte{}

	for i := 0; i < field_count; i++ {
		// read field name (null terminated string)
		field_name, err := common.ReadString(s.data[s.parseOffset:])
		if err != nil {
			logp.Err("Fail to read the column field")
		}
		fields = append(fields, field_name)
		m.NumberOfFields += 1
		s.parseOffset += len(field_name) + 1

		// read Table OID (int32)
		s.parseOffset += 4

		// read Column Index (int16)
		s.parseOffset += 2

		// read Type OID (int32)
		s.parseOffset += 4

		// read column length (int16)
		s.parseOffset += 2

		// read type modifier (int32)
		s.parseOffset += 4

		// read format (int16)
		format := common.Bytes_Ntohs(s.data[s.parseOffset : s.parseOffset+2])
		fields_format = append(fields_format, byte(format))
		s.parseOffset += 2

		logp.Debug("pgsqldetailed", "Field name=%s, format=%d", field_name, format)
	}
	m.Fields = fields
	m.FieldsFormat = fields_format
	if m.NumberOfFields != field_count {
		logp.Err("Missing fields from RowDescription. Expected %d. Received %d", field_count, m.NumberOfFields)
	}
}

func (pgsql *Pgsql) pgsqlRowsParser(s *PgsqlStream) error {
	m := s.message

	// read field count (int16)
	field_count := int(common.Bytes_Ntohs(s.data[s.parseOffset : s.parseOffset+2]))
	s.parseOffset += 2
	logp.Debug("pgsqldetailed", "DataRow field count=%d", field_count)

	row := []string{}
	var row_len int

	for i := 0; i < field_count; i++ {

		// read column length (int32)
		column_length := int32(common.Bytes_Ntohl(s.data[s.parseOffset : s.parseOffset+4]))
		s.parseOffset += 4

		if column_length > 0 && int(column_length) > len(s.data[s.parseOffset:]) {
			logp.Err("Pgsql invalid column_length=%v, buffer_length=%v, i=%v",
				column_length, len(s.data[s.parseOffset:]), i)
			return errInvalidLength
		}

		// read column value (byten)
		column_value := []byte{}

		if m.FieldsFormat[i] == 0 {
			// field value in text format
			if column_length > 0 {
				column_value = s.data[s.parseOffset : s.parseOffset+int(column_length)]
			} else if column_length == -1 {
				column_value = nil
			}
		}

		if row_len < pgsql.maxRowLength {
			if row_len+len(column_value) > pgsql.maxRowLength {
				column_value = column_value[:pgsql.maxRowLength-row_len]
			}
			row = append(row, string(column_value))
			row_len += len(column_value)
		}

		if column_length > 0 {
			s.parseOffset += int(column_length)
		}

		logp.Debug("pgsqldetailed", "Value %s, length=%d", string(column_value), column_length)

	}
	m.NumberOfRows += 1
	if len(m.Rows) < pgsql.maxStoreRows {
		m.Rows = append(m.Rows, row)
	}

	return nil
}

func pgsqlErrorParser(s *PgsqlStream) {

	m := s.message

	for len(s.data[s.parseOffset:]) > 0 {
		// read field type(byte1)
		field_type := s.data[s.parseOffset]
		s.parseOffset += 1

		if field_type == 0 {
			break
		}

		// read field value(string)
		field_value, err := common.ReadString(s.data[s.parseOffset:])
		if err != nil {
			logp.Err("Fail to read the column field")
		}
		s.parseOffset += len(field_value) + 1

		if field_type == 'M' {
			m.ErrorInfo = field_value
		} else if field_type == 'C' {
			m.ErrorCode = field_value
		} else if field_type == 'S' {
			m.ErrorSeverity = field_value
		}

	}
	logp.Debug("pgsqldetailed", "%s %s %s", m.ErrorSeverity, m.ErrorCode, m.ErrorInfo)
}

func isSpecialPgsqlCommand(data []byte) (bool, int) {

	if len(data) < 8 {
		// 8 bytes required
		return false, 0
	}

	// read length
	length := int(common.Bytes_Ntohl(data[0:4]))

	// read command identifier
	code := int(common.Bytes_Ntohl(data[4:8]))

	if length == 16 && code == 80877102 {
		// Cancel Request
		logp.Debug("pgsqldetailed", "Cancel Request, length=%d", length)
		return true, CancelRequest
	} else if length == 8 && code == 80877103 {
		// SSL Request
		logp.Debug("pgsqldetailed", "SSL Request, length=%d", length)
		return true, SSLRequest
	} else if code == 196608 {
		// Startup Message
		logp.Debug("pgsqldetailed", "Startup Message, length=%d", length)
		return true, StartupMessage
	}
	return false, 0
}

func (pgsql *Pgsql) pgsqlMessageParser(s *PgsqlStream) (bool, bool) {

	m := s.message
	for s.parseOffset < len(s.data) {
		switch s.parseState {
		case PgsqlStartState:
			if len(s.data[s.parseOffset:]) < 5 {
				logp.Warn("Postgresql Message too short. %X (length=%d). Wait for more.", s.data[s.parseOffset:], len(s.data[s.parseOffset:]))
				return true, false
			}

			is_special, command := isSpecialPgsqlCommand(s.data[s.parseOffset:])

			if is_special {
				// In case of Commands: StartupMessage, SSLRequest, CancelRequest that don't have
				// their type in the first byte

				// read length
				length := int(common.Bytes_Ntohl(s.data[s.parseOffset : s.parseOffset+4]))

				// ignore command
				if len(s.data[s.parseOffset:]) >= length {

					if command == SSLRequest {
						// if SSLRequest is received, expect for one byte reply (S or N)
						m.start = s.parseOffset
						s.parseOffset += length
						m.end = s.parseOffset
						m.isSSLRequest = true
						m.Size = uint64(m.end - m.start)

						return true, true
					}
					s.parseOffset += length
				} else {
					// wait for more
					logp.Debug("pgsqldetailed", "Wait for more data 1")
					return true, false
				}

			} else {
				// In case of Commands that have their type in the first byte

				// read type
				typ := byte(s.data[s.parseOffset])

				if s.expectSSLResponse {
					// SSLRequest was received in the other stream
					if typ == 'N' || typ == 'S' {
						// one byte reply to SSLRequest
						logp.Debug("pgsqldetailed", "Reply for SSLRequest %c", typ)
						m.start = s.parseOffset
						s.parseOffset += 1
						m.end = s.parseOffset
						m.isSSLResponse = true
						m.Size = uint64(m.end - m.start)

						return true, true
					}
				}

				// read length
				length := int(common.Bytes_Ntohl(s.data[s.parseOffset+1 : s.parseOffset+5]))

				if length < 4 {
					// length should include the size of itself (int32)
					logp.Debug("pgsqldetailed", "Invalid pgsql command length.")
					return false, false
				}

				logp.Debug("pgsqldetailed", "Pgsql type %c, length=%d", typ, length)

				if typ == 'Q' {
					// SimpleQuery
					m.start = s.parseOffset
					m.IsRequest = true

					if len(s.data[s.parseOffset:]) >= length+1 {
						s.parseOffset += 1 //type
						s.parseOffset += length
						m.end = s.parseOffset
						m.Size = uint64(m.end - m.start)

						m.Query = string(s.data[m.start+5 : m.end-1]) //without string termination

						m.toExport = true
						logp.Debug("pgsqldetailed", "Simple Query: %s", m.Query)
						return true, true
					} else {
						// wait for more
						logp.Debug("pgsqldetailed", "Wait for more data 2")
						return true, false
					}
				} else if typ == 'T' {
					// RowDescription

					m.start = s.parseOffset
					m.IsRequest = false
					m.IsOK = true
					m.toExport = true

					if len(s.data[s.parseOffset:]) >= length+1 {
						s.parseOffset += 1 //type
						s.parseOffset += 4 //length

						pgsqlFieldsParser(s)
						logp.Debug("pgsqldetailed", "Fields: %s", m.Fields)

						s.parseState = PgsqlGetDataState
					} else {
						// wait for more
						logp.Debug("pgsqldetailed", "Wait for more data 3")
						return true, false
					}

				} else if typ == 'I' {
					// EmptyQueryResponse, appears as a response for empty queries
					// substitutes CommandComplete

					logp.Debug("pgsqldetailed", "EmptyQueryResponse")
					m.start = s.parseOffset
					m.IsOK = true
					m.IsRequest = false
					m.toExport = true
					s.parseOffset += 5 // type + length
					m.end = s.parseOffset
					m.Size = uint64(m.end - m.start)

					return true, true

				} else if typ == 'E' {
					// ErrorResponse

					logp.Debug("pgsqldetailed", "ErrorResponse")
					m.start = s.parseOffset
					m.IsRequest = false
					m.IsError = true
					m.toExport = true

					if len(s.data[s.parseOffset:]) >= length+1 {
						s.parseOffset += 1 //type
						s.parseOffset += 4 //length

						pgsqlErrorParser(s)

						m.end = s.parseOffset
						m.Size = uint64(m.end - m.start)

						return true, true
					} else {
						// wait for more
						logp.Debug("pgsqldetailed", "Wait for more data 4")
						return true, false
					}
				} else if typ == 'C' {
					// CommandComplete -> Successful response

					m.start = s.parseOffset
					m.IsRequest = false
					m.IsOK = true
					m.toExport = true

					if len(s.data[s.parseOffset:]) >= length+1 {
						s.parseOffset += 1 //type

						name := string(s.data[s.parseOffset+4 : s.parseOffset+length-1]) //without \0
						logp.Debug("pgsqldetailed", "CommandComplete length=%d, tag=%s", length, name)

						s.parseOffset += length
						m.end = s.parseOffset
						m.Size = uint64(m.end - m.start)

						return true, true
					} else {
						// wait for more
						logp.Debug("pgsqldetailed", "Wait for more data 5")
						return true, false
					}
				} else if typ == 'Z' {
					// ReadyForQuery -> backend ready for a new query cycle
					if len(s.data[s.parseOffset:]) >= length+1 {
						m.start = s.parseOffset
						s.parseOffset += 1 // type
						s.parseOffset += length
						m.end = s.parseOffset
						m.Size = uint64(m.end - m.start)

						return true, true
					} else {
						// wait for more
						logp.Debug("pgsqldetailed", "Wait for more 5b")
						return true, false
					}
				} else {
					// TODO: add info from NoticeResponse in case there are warning messages for a query
					// ignore command
					if len(s.data[s.parseOffset:]) >= length+1 {
						s.parseOffset += 1 //type
						s.parseOffset += length
						m.end = s.parseOffset
						m.Size = uint64(m.end - m.start)

						// ok and complete, but ignore
						m.toExport = false
						return true, true
					} else {
						// wait for more
						logp.Debug("pgsqldetailed", "Wait for more data 6")
						return true, false
					}
				}
			}

			break

		case PgsqlGetDataState:

			// The response to queries that return row sets contains:
			// RowDescription
			// zero or more DataRow
			// CommandComplete
			// ReadyForQuery

			if len(s.data[s.parseOffset:]) < 5 {
				logp.Warn("Postgresql Message too short (length=%d). Wait for more.", len(s.data[s.parseOffset:]))
				return true, false
			}

			// read type
			typ := byte(s.data[s.parseOffset])

			// read message length
			length := int(common.Bytes_Ntohl(s.data[s.parseOffset+1 : s.parseOffset+5]))

			if typ == 'D' {
				// DataRow

				if len(s.data[s.parseOffset:]) >= length+1 {
					// skip type
					s.parseOffset += 1
					// skip length size
					s.parseOffset += 4

					if err := pgsql.pgsqlRowsParser(s); err != nil {
						return false, false
					}

				} else {
					// wait for more
					logp.Debug("pgsqldetailed", "Wait for more data 7")
					return true, false
				}

			} else if typ == 'C' {
				// CommandComplete

				if len(s.data[s.parseOffset:]) >= length+1 {

					// skip type
					s.parseOffset += 1

					name := string(s.data[s.parseOffset+4 : s.parseOffset+length-1]) //without \0
					logp.Debug("pgsqldetailed", "CommandComplete length=%d, tag=%s", length, name)

					s.parseOffset += length
					m.end = s.parseOffset
					m.Size = uint64(m.end - m.start)

					s.parseState = PgsqlStartState

					logp.Debug("pgsqldetailed", "Rows: %s", m.Rows)

					return true, true
				} else {
					// wait for more
					logp.Debug("pgsqldetailed", "Wait for more data 8")
					return true, false
				}
			} else {
				// shouldn't happen
				logp.Debug("pgsqldetailed", "Skip command of type %c", typ)
				s.parseState = PgsqlStartState
			}
			break
		}
	}

	return true, false
}

type pgsqlPrivateData struct {
	Data [2]*PgsqlStream
}

func (pgsql *Pgsql) ConnectionTimeout() time.Duration {
	return pgsql.transactionTimeout
}

func (pgsql *Pgsql) Parse(pkt *protos.Packet, tcptuple *common.TcpTuple,
	dir uint8, private protos.ProtocolData) protos.ProtocolData {

	defer logp.Recover("ParsePgsql exception")

	priv := pgsqlPrivateData{}
	if private != nil {
		var ok bool
		priv, ok = private.(pgsqlPrivateData)
		if !ok {
			priv = pgsqlPrivateData{}
		}
	}

	if priv.Data[dir] == nil {
		priv.Data[dir] = &PgsqlStream{
			tcptuple: tcptuple,
			data:     pkt.Payload,
			message:  &PgsqlMessage{Ts: pkt.Ts},
		}
		logp.Debug("pgsqldetailed", "New stream created")
	} else {
		// concatenate bytes
		priv.Data[dir].data = append(priv.Data[dir].data, pkt.Payload...)
		logp.Debug("pgsqldetailed", "Len data: %d cap data: %d", len(priv.Data[dir].data), cap(priv.Data[dir].data))
		if len(priv.Data[dir].data) > tcp.TCP_MAX_DATA_IN_STREAM {
			logp.Debug("pgsql", "Stream data too large, dropping TCP stream")
			priv.Data[dir] = nil
			return priv
		}
	}

	stream := priv.Data[dir]

	if priv.Data[1-dir] != nil && priv.Data[1-dir].seenSSLRequest {
		stream.expectSSLResponse = true
	}

	for len(stream.data) > 0 {

		if stream.message == nil {
			stream.message = &PgsqlMessage{Ts: pkt.Ts}
		}

		ok, complete := pgsql.pgsqlMessageParser(priv.Data[dir])
		//logp.Debug("pgsqldetailed", "MessageParser returned ok=%v complete=%v", ok, complete)
		if !ok {
			// drop this tcp stream. Will retry parsing with the next
			// segment in it
			priv.Data[dir] = nil
			logp.Debug("pgsql", "Ignore Postgresql message. Drop tcp stream. Try parsing with the next segment")
			return priv
		}

		if complete {
			// all ok, ship it
			msg := stream.data[stream.message.start:stream.message.end]

			if stream.message.isSSLRequest {
				// SSL request
				stream.seenSSLRequest = true
			} else if stream.message.isSSLResponse {
				// SSL request answered
				stream.expectSSLResponse = false
				priv.Data[1-dir].seenSSLRequest = false
			} else {
				if stream.message.toExport {
					pgsql.handlePgsql(pgsql, stream.message, tcptuple, dir, msg)
				}
			}

			// and reset message
			stream.PrepareForNewMessage()

		} else {
			// wait for more data
			break
		}
	}
	return priv
}

func messageHasEnoughData(msg *PgsqlMessage) bool {
	if msg == nil {
		return false
	}
	if msg.isSSLRequest || msg.isSSLResponse {
		return false
	}
	if msg.IsRequest {
		return len(msg.Query) > 0
	} else {
		return len(msg.Rows) > 0
	}
}

// Called when there's a drop packet
func (pgsql *Pgsql) GapInStream(tcptuple *common.TcpTuple, dir uint8,
	nbytes int, private protos.ProtocolData) (priv protos.ProtocolData, drop bool) {

	defer logp.Recover("GapInPgsqlStream exception")

	if private == nil {
		return private, false
	}
	pgsqlData, ok := private.(pgsqlPrivateData)
	if !ok {
		return private, false
	}
	if pgsqlData.Data[dir] == nil {
		return pgsqlData, false
	}

	// If enough data was received, send it to the
	// next layer but mark it as incomplete.
	stream := pgsqlData.Data[dir]
	if messageHasEnoughData(stream.message) {
		logp.Debug("pgsql", "Message not complete, but sending to the next layer")
		m := stream.message
		m.toExport = true
		m.end = stream.parseOffset
		if m.IsRequest {
			m.Notes = append(m.Notes, "Packet loss while capturing the request")
		} else {
			m.Notes = append(m.Notes, "Packet loss while capturing the response")
		}

		msg := stream.data[stream.message.start:stream.message.end]
		pgsql.handlePgsql(pgsql, stream.message, tcptuple, dir, msg)

		// and reset message
		stream.PrepareForNewMessage()
	}
	return pgsqlData, true
}

func (pgsql *Pgsql) ReceivedFin(tcptuple *common.TcpTuple, dir uint8,
	private protos.ProtocolData) protos.ProtocolData {

	// TODO
	return private
}

var handlePgsql = func(pgsql *Pgsql, m *PgsqlMessage, tcptuple *common.TcpTuple,
	dir uint8, raw_msg []byte) {

	m.TcpTuple = *tcptuple
	m.Direction = dir
	m.CmdlineTuple = procs.ProcWatcher.FindProcessesTuple(tcptuple.IpPort())

	if m.IsRequest {
		pgsql.receivedPgsqlRequest(m)
	} else {
		pgsql.receivedPgsqlResponse(m)
	}
}

func (pgsql *Pgsql) receivedPgsqlRequest(msg *PgsqlMessage) {

	tuple := msg.TcpTuple

	// parse the query, as it might contain a list of pgsql command
	// separated by ';'
	queries := pgsqlQueryParser(msg.Query)

	logp.Debug("pgsqldetailed", "Queries (%d) :%s", len(queries), queries)

	transList := pgsql.getTransaction(tuple.Hashable())
	if transList == nil {
		transList = []*PgsqlTransaction{}
	}

	for _, query := range queries {

		trans := &PgsqlTransaction{Type: "pgsql", tuple: tuple}

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

		trans.Pgsql = common.MapStr{}
		trans.Query = query
		trans.Method = getQueryMethod(query)
		trans.BytesIn = msg.Size

		trans.Notes = msg.Notes

		trans.Request_raw = query

		transList = append(transList, trans)
	}
	pgsql.transactions.Put(tuple.Hashable(), transList)
}

func (pgsql *Pgsql) receivedPgsqlResponse(msg *PgsqlMessage) {

	tuple := msg.TcpTuple
	transList := pgsql.getTransaction(tuple.Hashable())
	if transList == nil || len(transList) == 0 {
		logp.Warn("Response from unknown transaction. Ignoring.")
		return
	}

	// extract the first transaction from the array
	trans := pgsql.removeTransaction(transList, tuple, 0)

	// check if the request was received
	if trans.Pgsql == nil {
		logp.Warn("Response from unknown transaction. Ignoring.")
		return
	}

	trans.Pgsql.Update(common.MapStr{
		"iserror":        msg.IsError,
		"num_rows":       msg.NumberOfRows,
		"num_fields":     msg.NumberOfFields,
		"error_code":     msg.ErrorCode,
		"error_message":  msg.ErrorInfo,
		"error_severity": msg.ErrorSeverity,
	})
	trans.BytesOut = msg.Size

	trans.ResponseTime = int32(msg.Ts.Sub(trans.ts).Nanoseconds() / 1e6) // resp_time in milliseconds
	trans.Response_raw = common.DumpInCSVFormat(msg.Fields, msg.Rows)

	trans.Notes = append(trans.Notes, msg.Notes...)

	pgsql.publishTransaction(trans)

	logp.Debug("pgsql", "Postgres transaction completed: %s\n%s", trans.Pgsql, trans.Response_raw)
}

func (pgsql *Pgsql) publishTransaction(t *PgsqlTransaction) {

	if pgsql.results == nil {
		return
	}

	event := common.MapStr{}

	event["type"] = "pgsql"
	if t.Pgsql["iserror"].(bool) {
		event["status"] = common.ERROR_STATUS
	} else {
		event["status"] = common.OK_STATUS
	}
	event["responsetime"] = t.ResponseTime
	if pgsql.Send_request {
		event["request"] = t.Request_raw
	}
	if pgsql.Send_response {
		event["response"] = t.Response_raw
	}
	event["query"] = t.Query
	event["method"] = t.Method
	event["bytes_out"] = t.BytesOut
	event["bytes_in"] = t.BytesIn
	event["pgsql"] = t.Pgsql

	event["@timestamp"] = common.Time(t.ts)
	event["src"] = &t.Src
	event["dst"] = &t.Dst

	if len(t.Notes) > 0 {
		event["notes"] = t.Notes
	}

	pgsql.results.PublishEvent(event)
}

func (pgsql *Pgsql) removeTransaction(transList []*PgsqlTransaction,
	tuple common.TcpTuple, index int) *PgsqlTransaction {

	trans := transList[index]
	transList = append(transList[:index], transList[index+1:]...)
	if len(transList) == 0 {
		pgsql.transactions.Delete(trans.tuple.Hashable())
	} else {
		pgsql.transactions.Put(tuple.Hashable(), transList)
	}

	return trans
}
