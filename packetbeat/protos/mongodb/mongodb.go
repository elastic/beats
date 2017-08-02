package mongodb

import (
	"fmt"
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

var debugf = logp.MakeDebug("mongodb")

type mongodbPlugin struct {
	// config
	ports        []int
	sendRequest  bool
	sendResponse bool
	maxDocs      int
	maxDocLength int

	requests           *common.Cache
	responses          *common.Cache
	transactionTimeout time.Duration

	results protos.Reporter
}

type transactionKey struct {
	tcp common.HashableTCPTuple
	id  int
}

var (
	unmatchedRequests = monitoring.NewInt(nil, "mongodb.unmatched_requests")
)

func init() {
	protos.Register("mongodb", New)
}

func New(
	testMode bool,
	results protos.Reporter,
	cfg *common.Config,
) (protos.Plugin, error) {
	p := &mongodbPlugin{}
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

func (mongodb *mongodbPlugin) init(results protos.Reporter, config *mongodbConfig) error {
	debugf("Init a MongoDB protocol parser")
	mongodb.setFromConfig(config)

	mongodb.requests = common.NewCache(
		mongodb.transactionTimeout,
		protos.DefaultTransactionHashSize)
	mongodb.requests.StartJanitor(mongodb.transactionTimeout)
	mongodb.responses = common.NewCache(
		mongodb.transactionTimeout,
		protos.DefaultTransactionHashSize)
	mongodb.responses.StartJanitor(mongodb.transactionTimeout)
	mongodb.results = results

	return nil
}

func (mongodb *mongodbPlugin) setFromConfig(config *mongodbConfig) {
	mongodb.ports = config.Ports
	mongodb.sendRequest = config.SendRequest
	mongodb.sendResponse = config.SendResponse
	mongodb.maxDocs = config.MaxDocs
	mongodb.maxDocLength = config.MaxDocLength
	mongodb.transactionTimeout = config.TransactionTimeout
}

func (mongodb *mongodbPlugin) GetPorts() []int {
	return mongodb.ports
}

func (mongodb *mongodbPlugin) ConnectionTimeout() time.Duration {
	return mongodb.transactionTimeout
}

func (mongodb *mongodbPlugin) Parse(
	pkt *protos.Packet,
	tcptuple *common.TCPTuple,
	dir uint8,
	private protos.ProtocolData,
) protos.ProtocolData {
	defer logp.Recover("ParseMongodb exception")
	debugf("Parse method triggered")

	conn := ensureMongodbConnection(private)
	conn = mongodb.doParse(conn, pkt, tcptuple, dir)
	if conn == nil {
		return nil
	}
	return conn
}

func ensureMongodbConnection(private protos.ProtocolData) *mongodbConnectionData {
	if private == nil {
		return &mongodbConnectionData{}
	}

	priv, ok := private.(*mongodbConnectionData)
	if !ok {
		logp.Warn("mongodb connection data type error, create new one")
		return &mongodbConnectionData{}
	}
	if priv == nil {
		debugf("Unexpected: mongodb connection data not set, create new one")
		return &mongodbConnectionData{}
	}

	return priv
}

func (mongodb *mongodbPlugin) doParse(
	conn *mongodbConnectionData,
	pkt *protos.Packet,
	tcptuple *common.TCPTuple,
	dir uint8,
) *mongodbConnectionData {
	st := conn.streams[dir]
	if st == nil {
		st = newStream(pkt, tcptuple)
		conn.streams[dir] = st
		debugf("new stream: %p (dir=%v, len=%v)", st, dir, len(pkt.Payload))
	} else {
		// concatenate bytes
		st.data = append(st.data, pkt.Payload...)
		if len(st.data) > tcp.TCPMaxDataInStream {
			debugf("Stream data too large, dropping TCP stream")
			conn.streams[dir] = nil
			return conn
		}
	}

	for len(st.data) > 0 {
		if st.message == nil {
			st.message = &mongodbMessage{ts: pkt.Ts}
		}

		ok, complete := mongodbMessageParser(st)
		if !ok {
			// drop this tcp stream. Will retry parsing with the next
			// segment in it
			conn.streams[dir] = nil
			debugf("Ignore Mongodb message. Drop tcp stream. Try parsing with the next segment")
			return conn
		}

		if !complete {
			// wait for more data
			debugf("MongoDB wait for more data before parsing message")
			break
		}

		// all ok, go to next level and reset stream for new message
		debugf("MongoDB message complete")
		mongodb.handleMongodb(conn, st.message, tcptuple, dir)
		st.PrepareForNewMessage()
	}

	return conn
}

func newStream(pkt *protos.Packet, tcptuple *common.TCPTuple) *stream {
	s := &stream{
		tcptuple: tcptuple,
		data:     pkt.Payload,
		message:  &mongodbMessage{ts: pkt.Ts},
	}
	return s
}

func (mongodb *mongodbPlugin) handleMongodb(
	conn *mongodbConnectionData,
	m *mongodbMessage,
	tcptuple *common.TCPTuple,
	dir uint8,
) {

	m.tcpTuple = *tcptuple
	m.direction = dir
	m.cmdlineTuple = procs.ProcWatcher.FindProcessesTuple(tcptuple.IPPort())

	if m.isResponse {
		debugf("MongoDB response message")
		mongodb.onResponse(conn, m)
	} else {
		debugf("MongoDB request message")
		mongodb.onRequest(conn, m)
	}
}

func (mongodb *mongodbPlugin) onRequest(conn *mongodbConnectionData, msg *mongodbMessage) {
	// publish request only transaction
	if !awaitsReply(msg.opCode) {
		mongodb.onTransComplete(msg, nil)
		return
	}

	id := msg.requestID
	key := transactionKey{tcp: msg.tcpTuple.Hashable(), id: id}

	// try to find matching response potentially inserted before
	if v := mongodb.responses.Delete(key); v != nil {
		resp := v.(*mongodbMessage)
		mongodb.onTransComplete(msg, resp)
		return
	}

	// insert into cache for correlation
	old := mongodb.requests.Put(key, msg)
	if old != nil {
		debugf("Two requests without a Response. Dropping old request")
		unmatchedRequests.Add(1)
	}
}

func (mongodb *mongodbPlugin) onResponse(conn *mongodbConnectionData, msg *mongodbMessage) {
	id := msg.responseTo
	key := transactionKey{tcp: msg.tcpTuple.Hashable(), id: id}

	// try to find matching request
	if v := mongodb.requests.Delete(key); v != nil {
		requ := v.(*mongodbMessage)
		mongodb.onTransComplete(requ, msg)
		return
	}

	// insert into cache for correlation
	mongodb.responses.Put(key, msg)
}

func (mongodb *mongodbPlugin) onTransComplete(requ, resp *mongodbMessage) {
	trans := newTransaction(requ, resp)
	debugf("Mongodb transaction completed: %s", trans.mongodb)

	mongodb.publishTransaction(trans)
}

func newTransaction(requ, resp *mongodbMessage) *transaction {
	trans := &transaction{}

	// fill request
	if requ != nil {
		trans.mongodb = common.MapStr{}
		trans.event = requ.event
		trans.method = requ.method

		trans.cmdline = requ.cmdlineTuple
		trans.ts = requ.ts
		trans.src = common.Endpoint{
			IP:   requ.tcpTuple.SrcIP.String(),
			Port: requ.tcpTuple.SrcPort,
			Proc: string(requ.cmdlineTuple.Src),
		}
		trans.dst = common.Endpoint{
			IP:   requ.tcpTuple.DstIP.String(),
			Port: requ.tcpTuple.DstPort,
			Proc: string(requ.cmdlineTuple.Dst),
		}
		if requ.direction == tcp.TCPDirectionReverse {
			trans.src, trans.dst = trans.dst, trans.src
		}
		trans.params = requ.params
		trans.resource = requ.resource
		trans.bytesIn = requ.messageLength
	}

	// fill response
	if resp != nil {
		for k, v := range resp.event {
			trans.event[k] = v
		}

		trans.error = resp.error
		trans.documents = resp.documents

		trans.responseTime = int32(resp.ts.Sub(trans.ts).Nanoseconds() / 1e6) // resp_time in milliseconds
		trans.bytesOut = resp.messageLength

	}

	return trans
}

func (mongodb *mongodbPlugin) GapInStream(tcptuple *common.TCPTuple, dir uint8,
	nbytes int, private protos.ProtocolData) (priv protos.ProtocolData, drop bool) {
	return private, true
}

func (mongodb *mongodbPlugin) ReceivedFin(tcptuple *common.TCPTuple, dir uint8,
	private protos.ProtocolData) protos.ProtocolData {
	return private
}

func copyMapWithoutKey(d map[string]interface{}, key string) map[string]interface{} {
	res := map[string]interface{}{}
	for k, v := range d {
		if k != key {
			res[k] = v
		}
	}
	return res
}

func reconstructQuery(t *transaction, full bool) (query string) {
	query = t.resource + "." + t.method + "("
	if len(t.params) > 0 {
		var err error
		var params string
		if !full {
			// remove the actual data.
			// TODO: review if we need to add other commands here
			if t.method == "insert" {
				params, err = doc2str(copyMapWithoutKey(t.params, "documents"))
			} else if t.method == "update" {
				params, err = doc2str(copyMapWithoutKey(t.params, "updates"))
			} else if t.method == "findandmodify" {
				params, err = doc2str(copyMapWithoutKey(t.params, "update"))
			}
		} else {
			params, err = doc2str(t.params)
		}
		if err != nil {
			debugf("Error marshaling params: %v", err)
		} else {
			query += params
		}
	}
	query += ")"
	skip, _ := t.event["numberToSkip"].(int)
	if skip > 0 {
		query += fmt.Sprintf(".skip(%d)", skip)
	}

	limit, _ := t.event["numberToReturn"].(int)
	if limit > 0 && limit < 0x7fffffff {
		query += fmt.Sprintf(".limit(%d)", limit)
	}
	return
}

func (mongodb *mongodbPlugin) publishTransaction(t *transaction) {
	if mongodb.results == nil {
		debugf("Try to publish transaction with null results")
		return
	}

	timestamp := t.ts
	fields := common.MapStr{}
	fields["type"] = "mongodb"
	if t.error == "" {
		fields["status"] = common.OK_STATUS
	} else {
		t.event["error"] = t.error
		fields["status"] = common.ERROR_STATUS
	}
	fields["mongodb"] = t.event
	fields["method"] = t.method
	fields["resource"] = t.resource
	fields["query"] = reconstructQuery(t, false)
	fields["responsetime"] = t.responseTime
	fields["bytes_in"] = uint64(t.bytesIn)
	fields["bytes_out"] = uint64(t.bytesOut)
	fields["src"] = &t.src
	fields["dst"] = &t.dst

	if mongodb.sendRequest {
		fields["request"] = reconstructQuery(t, true)
	}
	if mongodb.sendResponse {
		if len(t.documents) > 0 {
			// response field needs to be a string
			docs := make([]string, 0, len(t.documents))
			for i, doc := range t.documents {
				if mongodb.maxDocs > 0 && i >= mongodb.maxDocs {
					docs = append(docs, "[...]")
					break
				}
				str, err := doc2str(doc)
				if err != nil {
					logp.Warn("Failed to JSON marshal document from Mongo: %v (error: %v)", doc, err)
				} else {
					if mongodb.maxDocLength > 0 && len(str) > mongodb.maxDocLength {
						str = str[:mongodb.maxDocLength] + " ..."
					}
					docs = append(docs, str)
				}
			}
			fields["response"] = strings.Join(docs, "\n")
		}
	}

	mongodb.results(beat.Event{
		Timestamp: timestamp,
		Fields:    fields,
	})
}
