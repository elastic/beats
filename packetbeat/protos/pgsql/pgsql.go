package pgsql

import (
	"errors"
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

type pgsqlPlugin struct {

	// config
	ports        []int
	maxStoreRows int
	maxRowLength int
	sendRequest  bool
	sendResponse bool

	transactions       *common.Cache
	transactionTimeout time.Duration

	results protos.Reporter

	// function pointer for mocking
	handlePgsql func(pgsql *pgsqlPlugin, m *pgsqlMessage, tcp *common.TCPTuple,
		dir uint8, raw_msg []byte)
}

type pgsqlMessage struct {
	start         int
	end           int
	isSSLResponse bool
	isSSLRequest  bool
	toExport      bool

	ts             time.Time
	isRequest      bool
	query          string
	size           uint64
	fields         []string
	fieldsFormat   []byte
	rows           [][]string
	numberOfRows   int
	numberOfFields int
	isOK           bool
	isError        bool
	errorInfo      string
	errorCode      string
	errorSeverity  string
	notes          []string

	direction    uint8
	tcpTuple     common.TCPTuple
	cmdlineTuple *common.CmdlineTuple
}

type pgsqlTransaction struct {
	tuple        common.TCPTuple
	src          common.Endpoint
	dst          common.Endpoint
	responseTime int32
	ts           time.Time
	query        string
	method       string
	bytesOut     uint64
	bytesIn      uint64
	notes        []string

	pgsql common.MapStr

	requestRaw  string
	responseRaw string
}

type pgsqlStream struct {
	data []byte

	parseOffset       int
	parseState        int
	seenSSLRequest    bool
	expectSSLResponse bool

	message *pgsqlMessage
}

const (
	pgsqlStartState = iota
	pgsqlGetDataState
	pgsqlExtendedQueryState
)

const (
	sslRequest = iota
	startupMessage
	cancelRequest
)

var (
	errInvalidLength = errors.New("invalid length")
)

var (
	debugf    = logp.MakeDebug("pgsql")
	detailedf = logp.MakeDebug("pgsqldetailed")
)

var (
	unmatchedResponses = monitoring.NewInt(nil, "pgsql.unmatched_responses")
)

func init() {
	protos.Register("pgsql", New)
}

func New(
	testMode bool,
	results protos.Reporter,
	cfg *common.Config,
) (protos.Plugin, error) {
	p := &pgsqlPlugin{}
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

func (pgsql *pgsqlPlugin) init(results protos.Reporter, config *pgsqlConfig) error {
	pgsql.setFromConfig(config)

	pgsql.transactions = common.NewCache(
		pgsql.transactionTimeout,
		protos.DefaultTransactionHashSize)
	pgsql.transactions.StartJanitor(pgsql.transactionTimeout)
	pgsql.handlePgsql = handlePgsql
	pgsql.results = results

	return nil
}

func (pgsql *pgsqlPlugin) setFromConfig(config *pgsqlConfig) {
	pgsql.ports = config.Ports
	pgsql.maxRowLength = config.MaxRowLength
	pgsql.maxStoreRows = config.MaxRows
	pgsql.sendRequest = config.SendRequest
	pgsql.sendResponse = config.SendResponse
	pgsql.transactionTimeout = config.TransactionTimeout
}

func (pgsql *pgsqlPlugin) getTransaction(k common.HashableTCPTuple) []*pgsqlTransaction {
	v := pgsql.transactions.Get(k)
	if v != nil {
		return v.([]*pgsqlTransaction)
	}
	return nil
}

func (pgsql *pgsqlPlugin) GetPorts() []int {
	return pgsql.ports
}

func (stream *pgsqlStream) prepareForNewMessage() {
	stream.data = stream.data[stream.message.end:]
	stream.parseState = pgsqlStartState
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
	data [2]*pgsqlStream
}

func (pgsql *pgsqlPlugin) ConnectionTimeout() time.Duration {
	return pgsql.transactionTimeout
}

func (pgsql *pgsqlPlugin) Parse(pkt *protos.Packet, tcptuple *common.TCPTuple,
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

	if priv.data[dir] == nil {
		priv.data[dir] = &pgsqlStream{
			data:    pkt.Payload,
			message: &pgsqlMessage{ts: pkt.Ts},
		}
		logp.Debug("pgsqldetailed", "New stream created")
	} else {
		// concatenate bytes
		priv.data[dir].data = append(priv.data[dir].data, pkt.Payload...)
		logp.Debug("pgsqldetailed", "Len data: %d cap data: %d", len(priv.data[dir].data), cap(priv.data[dir].data))
		if len(priv.data[dir].data) > tcp.TCPMaxDataInStream {
			debugf("Stream data too large, dropping TCP stream")
			priv.data[dir] = nil
			return priv
		}
	}

	stream := priv.data[dir]

	if priv.data[1-dir] != nil && priv.data[1-dir].seenSSLRequest {
		stream.expectSSLResponse = true
	}

	for len(stream.data) > 0 {

		if stream.message == nil {
			stream.message = &pgsqlMessage{ts: pkt.Ts}
		}

		ok, complete := pgsql.pgsqlMessageParser(priv.data[dir])
		//logp.Debug("pgsqldetailed", "MessageParser returned ok=%v complete=%v", ok, complete)
		if !ok {
			// drop this tcp stream. Will retry parsing with the next
			// segment in it
			priv.data[dir] = nil
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
				priv.data[1-dir].seenSSLRequest = false
			} else {
				if stream.message.toExport {
					pgsql.handlePgsql(pgsql, stream.message, tcptuple, dir, msg)
				}
			}

			// and reset message
			stream.prepareForNewMessage()

		} else {
			// wait for more data
			break
		}
	}
	return priv
}

func messageHasEnoughData(msg *pgsqlMessage) bool {
	if msg == nil {
		return false
	}
	if msg.isSSLRequest || msg.isSSLResponse {
		return false
	}
	if msg.isRequest {
		return len(msg.query) > 0
	}
	return len(msg.rows) > 0
}

// Called when there's a drop packet
func (pgsql *pgsqlPlugin) GapInStream(tcptuple *common.TCPTuple, dir uint8,
	nbytes int, private protos.ProtocolData) (priv protos.ProtocolData, drop bool) {

	defer logp.Recover("GapInPgsqlStream exception")

	if private == nil {
		return private, false
	}
	pgsqlData, ok := private.(pgsqlPrivateData)
	if !ok {
		return private, false
	}
	if pgsqlData.data[dir] == nil {
		return pgsqlData, false
	}

	// If enough data was received, send it to the
	// next layer but mark it as incomplete.
	stream := pgsqlData.data[dir]
	if messageHasEnoughData(stream.message) {
		debugf("Message not complete, but sending to the next layer")
		m := stream.message
		m.toExport = true
		m.end = stream.parseOffset
		if m.isRequest {
			m.notes = append(m.notes, "Packet loss while capturing the request")
		} else {
			m.notes = append(m.notes, "Packet loss while capturing the response")
		}

		msg := stream.data[stream.message.start:stream.message.end]
		pgsql.handlePgsql(pgsql, stream.message, tcptuple, dir, msg)

		// and reset message
		stream.prepareForNewMessage()
	}
	return pgsqlData, true
}

func (pgsql *pgsqlPlugin) ReceivedFin(tcptuple *common.TCPTuple, dir uint8,
	private protos.ProtocolData) protos.ProtocolData {
	return private
}

var handlePgsql = func(pgsql *pgsqlPlugin, m *pgsqlMessage, tcptuple *common.TCPTuple,
	dir uint8, raw_msg []byte) {

	m.tcpTuple = *tcptuple
	m.direction = dir
	m.cmdlineTuple = procs.ProcWatcher.FindProcessesTuple(tcptuple.IPPort())

	if m.isRequest {
		pgsql.receivedPgsqlRequest(m)
	} else {
		pgsql.receivedPgsqlResponse(m)
	}
}

func (pgsql *pgsqlPlugin) receivedPgsqlRequest(msg *pgsqlMessage) {
	tuple := msg.tcpTuple

	// parse the query, as it might contain a list of pgsql command
	// separated by ';'
	queries := pgsqlQueryParser(msg.query)

	logp.Debug("pgsqldetailed", "Queries (%d) :%s", len(queries), queries)

	transList := pgsql.getTransaction(tuple.Hashable())
	if transList == nil {
		transList = []*pgsqlTransaction{}
	}

	for _, query := range queries {

		trans := &pgsqlTransaction{tuple: tuple}

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

		trans.pgsql = common.MapStr{}
		trans.query = query
		trans.method = getQueryMethod(query)
		trans.bytesIn = msg.size

		trans.notes = msg.notes

		trans.requestRaw = query

		transList = append(transList, trans)
	}
	pgsql.transactions.Put(tuple.Hashable(), transList)
}

func (pgsql *pgsqlPlugin) receivedPgsqlResponse(msg *pgsqlMessage) {
	tuple := msg.tcpTuple
	transList := pgsql.getTransaction(tuple.Hashable())
	if transList == nil || len(transList) == 0 {
		debugf("Response from unknown transaction. Ignoring.")
		unmatchedResponses.Add(1)
		return
	}

	// extract the first transaction from the array
	trans := pgsql.removeTransaction(transList, tuple, 0)

	// check if the request was received
	if trans.pgsql == nil {
		debugf("Response from unknown transaction. Ignoring.")
		unmatchedResponses.Add(1)
		return
	}

	trans.pgsql.Update(common.MapStr{
		"iserror":        msg.isError,
		"num_rows":       msg.numberOfRows,
		"num_fields":     msg.numberOfFields,
		"error_code":     msg.errorCode,
		"error_message":  msg.errorInfo,
		"error_severity": msg.errorSeverity,
	})
	trans.bytesOut = msg.size

	trans.responseTime = int32(msg.ts.Sub(trans.ts).Nanoseconds() / 1e6) // resp_time in milliseconds
	trans.responseRaw = common.DumpInCSVFormat(msg.fields, msg.rows)

	trans.notes = append(trans.notes, msg.notes...)

	pgsql.publishTransaction(trans)

	debugf("Postgres transaction completed: %s\n%s", trans.pgsql, trans.responseRaw)
}

func (pgsql *pgsqlPlugin) publishTransaction(t *pgsqlTransaction) {
	if pgsql.results == nil {
		return
	}

	fields := common.MapStr{}

	fields["type"] = "pgsql"
	if t.pgsql["iserror"].(bool) {
		fields["status"] = common.ERROR_STATUS
	} else {
		fields["status"] = common.OK_STATUS
	}
	fields["responsetime"] = t.responseTime
	if pgsql.sendRequest {
		fields["request"] = t.requestRaw
	}
	if pgsql.sendResponse {
		fields["response"] = t.responseRaw
	}
	fields["query"] = t.query
	fields["method"] = t.method
	fields["bytes_out"] = t.bytesOut
	fields["bytes_in"] = t.bytesIn
	fields["pgsql"] = t.pgsql

	fields["src"] = &t.src
	fields["dst"] = &t.dst

	if len(t.notes) > 0 {
		fields["notes"] = t.notes
	}

	pgsql.results(beat.Event{
		Timestamp: t.ts,
		Fields:    fields,
	})
}

func (pgsql *pgsqlPlugin) removeTransaction(transList []*pgsqlTransaction,
	tuple common.TCPTuple, index int) *pgsqlTransaction {

	trans := transList[index]
	transList = append(transList[:index], transList[index+1:]...)
	if len(transList) == 0 {
		pgsql.transactions.Delete(trans.tuple.Hashable())
	} else {
		pgsql.transactions.Put(tuple.Hashable(), transList)
	}

	return trans
}
