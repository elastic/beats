package pgsql

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
	PgsqlExtendedQueryState
)

const (
	SSLRequest = iota
	StartupMessage
	CancelRequest
)

var (
	errInvalidLength = errors.New("invalid length")
)

var (
	debugf    = logp.MakeDebug("pgsql")
	detailedf = logp.MakeDebug("pgsqldetailed")
)

var (
	unmatchedResponses = expvar.NewInt("pgsql.unmatched_responses")
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

	results publish.Transactions

	// function pointer for mocking
	handlePgsql func(pgsql *Pgsql, m *PgsqlMessage, tcp *common.TcpTuple,
		dir uint8, raw_msg []byte)
}

func init() {
	protos.Register("pgsql", New)
}

func New(
	testMode bool,
	results publish.Transactions,
	cfg *common.Config,
) (protos.Plugin, error) {
	p := &Pgsql{}
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

func (pgsql *Pgsql) init(results publish.Transactions, config *pgsqlConfig) error {
	pgsql.setFromConfig(config)

	pgsql.transactions = common.NewCache(
		pgsql.transactionTimeout,
		protos.DefaultTransactionHashSize)
	pgsql.transactions.StartJanitor(pgsql.transactionTimeout)
	pgsql.handlePgsql = handlePgsql
	pgsql.results = results

	return nil
}

func (pgsql *Pgsql) setFromConfig(config *pgsqlConfig) {
	pgsql.Ports = config.Ports
	pgsql.maxRowLength = config.MaxRowLength
	pgsql.maxStoreRows = config.MaxRows
	pgsql.Send_request = config.SendRequest
	pgsql.Send_response = config.SendResponse
	pgsql.transactionTimeout = config.TransactionTimeout
}

func (pgsql *Pgsql) getTransaction(k common.HashableTcpTuple) []*PgsqlTransaction {
	v := pgsql.transactions.Get(k)
	if v != nil {
		return v.([]*PgsqlTransaction)
	}
	return nil
}

func (pgsql *Pgsql) GetPorts() []int {
	return pgsql.Ports
}

func (stream *PgsqlStream) PrepareForNewMessage() {
	stream.data = stream.data[stream.message.end:]
	stream.parseState = PgsqlStartState
	stream.parseOffset = 0
	stream.message = nil
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
			debugf("Stream data too large, dropping TCP stream")
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
			debugf("Ignore Postgresql message. Drop tcp stream. Try parsing with the next segment")
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
		debugf("Message not complete, but sending to the next layer")
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
		debugf("Response from unknown transaction. Ignoring.")
		unmatchedResponses.Add(1)
		return
	}

	// extract the first transaction from the array
	trans := pgsql.removeTransaction(transList, tuple, 0)

	// check if the request was received
	if trans.Pgsql == nil {
		debugf("Response from unknown transaction. Ignoring.")
		unmatchedResponses.Add(1)
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

	debugf("Postgres transaction completed: %s\n%s", trans.Pgsql, trans.Response_raw)
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

	pgsql.results.PublishTransaction(event)
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
