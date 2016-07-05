package mongodb

import (
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

var debugf = logp.MakeDebug("mongodb")

type Mongodb struct {
	// config
	Ports        []int
	SendRequest  bool
	SendResponse bool
	MaxDocs      int
	MaxDocLength int

	requests           *common.Cache
	responses          *common.Cache
	transactionTimeout time.Duration

	results publish.Transactions
}

type transactionKey struct {
	tcp common.HashableTcpTuple
	id  int
}

var (
	unmatchedRequests = expvar.NewInt("mongodb.unmatched_requests")
)

func init() {
	protos.Register("mongodb", New)
}

func New(
	testMode bool,
	results publish.Transactions,
	cfg *common.Config,
) (protos.Plugin, error) {
	p := &Mongodb{}
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

func (mongodb *Mongodb) init(results publish.Transactions, config *mongodbConfig) error {
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

func (mongodb *Mongodb) setFromConfig(config *mongodbConfig) {
	mongodb.Ports = config.Ports
	mongodb.SendRequest = config.SendRequest
	mongodb.SendResponse = config.SendResponse
	mongodb.MaxDocs = config.MaxDocs
	mongodb.MaxDocLength = config.MaxDocLength
	mongodb.transactionTimeout = config.TransactionTimeout
}

func (mongodb *Mongodb) GetPorts() []int {
	return mongodb.Ports
}

func (mongodb *Mongodb) ConnectionTimeout() time.Duration {
	return mongodb.transactionTimeout
}

func (mongodb *Mongodb) Parse(
	pkt *protos.Packet,
	tcptuple *common.TcpTuple,
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

func (mongodb *Mongodb) doParse(
	conn *mongodbConnectionData,
	pkt *protos.Packet,
	tcptuple *common.TcpTuple,
	dir uint8,
) *mongodbConnectionData {
	st := conn.Streams[dir]
	if st == nil {
		st = newStream(pkt, tcptuple)
		conn.Streams[dir] = st
		debugf("new stream: %p (dir=%v, len=%v)", st, dir, len(pkt.Payload))
	} else {
		// concatenate bytes
		st.data = append(st.data, pkt.Payload...)
		if len(st.data) > tcp.TCP_MAX_DATA_IN_STREAM {
			debugf("Stream data too large, dropping TCP stream")
			conn.Streams[dir] = nil
			return conn
		}
	}

	for len(st.data) > 0 {
		if st.message == nil {
			st.message = &mongodbMessage{Ts: pkt.Ts}
		}

		ok, complete := mongodbMessageParser(st)
		if !ok {
			// drop this tcp stream. Will retry parsing with the next
			// segment in it
			conn.Streams[dir] = nil
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

func newStream(pkt *protos.Packet, tcptuple *common.TcpTuple) *stream {
	s := &stream{
		tcptuple: tcptuple,
		data:     pkt.Payload,
		message:  &mongodbMessage{Ts: pkt.Ts},
	}
	return s
}

func (mongodb *Mongodb) handleMongodb(
	conn *mongodbConnectionData,
	m *mongodbMessage,
	tcptuple *common.TcpTuple,
	dir uint8,
) {

	m.TcpTuple = *tcptuple
	m.Direction = dir
	m.CmdlineTuple = procs.ProcWatcher.FindProcessesTuple(tcptuple.IpPort())

	if m.IsResponse {
		debugf("MongoDB response message")
		mongodb.onResponse(conn, m)
	} else {
		debugf("MongoDB request message")
		mongodb.onRequest(conn, m)
	}
}

func (mongodb *Mongodb) onRequest(conn *mongodbConnectionData, msg *mongodbMessage) {
	// publish request only transaction
	if !awaitsReply(msg.opCode) {
		mongodb.onTransComplete(msg, nil)
		return
	}

	id := msg.requestId
	key := transactionKey{tcp: msg.TcpTuple.Hashable(), id: id}

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

func (mongodb *Mongodb) onResponse(conn *mongodbConnectionData, msg *mongodbMessage) {
	id := msg.responseTo
	key := transactionKey{tcp: msg.TcpTuple.Hashable(), id: id}

	// try to find matching request
	if v := mongodb.requests.Delete(key); v != nil {
		requ := v.(*mongodbMessage)
		mongodb.onTransComplete(requ, msg)
		return
	}

	// insert into cache for correlation
	mongodb.responses.Put(key, msg)
}

func (mongodb *Mongodb) onTransComplete(requ, resp *mongodbMessage) {
	trans := newTransaction(requ, resp)
	debugf("Mongodb transaction completed: %s", trans.Mongodb)

	mongodb.publishTransaction(trans)
}

func newTransaction(requ, resp *mongodbMessage) *transaction {
	trans := &transaction{Type: "mongodb"}

	// fill request
	if requ != nil {
		trans.tuple = requ.TcpTuple

		trans.Mongodb = common.MapStr{}
		trans.event = requ.event
		trans.method = requ.method

		trans.cmdline = requ.CmdlineTuple
		trans.ts = requ.Ts
		trans.Ts = int64(trans.ts.UnixNano() / 1000) // transactions have microseconds resolution
		trans.JsTs = requ.Ts
		trans.Src = common.Endpoint{
			Ip:   requ.TcpTuple.Src_ip.String(),
			Port: requ.TcpTuple.Src_port,
			Proc: string(requ.CmdlineTuple.Src),
		}
		trans.Dst = common.Endpoint{
			Ip:   requ.TcpTuple.Dst_ip.String(),
			Port: requ.TcpTuple.Dst_port,
			Proc: string(requ.CmdlineTuple.Dst),
		}
		if requ.Direction == tcp.TcpDirectionReverse {
			trans.Src, trans.Dst = trans.Dst, trans.Src
		}
		trans.params = requ.params
		trans.resource = requ.resource
		trans.BytesIn = requ.messageLength
	}

	// fill response
	if resp != nil {
		if requ == nil {
			// TODO: reverse tuple?
			trans.tuple = resp.TcpTuple
		}

		for k, v := range resp.event {
			trans.event[k] = v
		}

		trans.error = resp.error
		trans.documents = resp.documents

		trans.ResponseTime = int32(resp.Ts.Sub(trans.ts).Nanoseconds() / 1e6) // resp_time in milliseconds
		trans.BytesOut = resp.messageLength

	}

	return trans
}

func (mongodb *Mongodb) GapInStream(tcptuple *common.TcpTuple, dir uint8,
	nbytes int, private protos.ProtocolData) (priv protos.ProtocolData, drop bool) {
	return private, true
}

func (mongodb *Mongodb) ReceivedFin(tcptuple *common.TcpTuple, dir uint8,
	private protos.ProtocolData) protos.ProtocolData {
	return private
}

func copy_map_without_key(d map[string]interface{}, key string) map[string]interface{} {
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
				params, err = doc2str(copy_map_without_key(t.params, "documents"))
			} else if t.method == "update" {
				params, err = doc2str(copy_map_without_key(t.params, "updates"))
			} else if t.method == "findandmodify" {
				params, err = doc2str(copy_map_without_key(t.params, "update"))
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

func (mongodb *Mongodb) publishTransaction(t *transaction) {

	if mongodb.results == nil {
		debugf("Try to publish transaction with null results")
		return
	}

	event := common.MapStr{}
	event["type"] = "mongodb"
	if t.error == "" {
		event["status"] = common.OK_STATUS
	} else {
		t.event["error"] = t.error
		event["status"] = common.ERROR_STATUS
	}
	event["mongodb"] = t.event
	event["method"] = t.method
	event["resource"] = t.resource
	event["query"] = reconstructQuery(t, false)
	event["responsetime"] = t.ResponseTime
	event["bytes_in"] = uint64(t.BytesIn)
	event["bytes_out"] = uint64(t.BytesOut)
	event["@timestamp"] = common.Time(t.ts)
	event["src"] = &t.Src
	event["dst"] = &t.Dst

	if mongodb.SendRequest {
		event["request"] = reconstructQuery(t, true)
	}
	if mongodb.SendResponse {
		if len(t.documents) > 0 {
			// response field needs to be a string
			docs := make([]string, 0, len(t.documents))
			for i, doc := range t.documents {
				if mongodb.MaxDocs > 0 && i >= mongodb.MaxDocs {
					docs = append(docs, "[...]")
					break
				}
				str, err := doc2str(doc)
				if err != nil {
					logp.Warn("Failed to JSON marshal document from Mongo: %v (error: %v)", doc, err)
				} else {
					if mongodb.MaxDocLength > 0 && len(str) > mongodb.MaxDocLength {
						str = str[:mongodb.MaxDocLength] + " ..."
					}
					docs = append(docs, str)
				}
			}
			event["response"] = strings.Join(docs, "\n")
		}
	}

	mongodb.results.PublishTransaction(event)
}
