package mongodb

import (
	"fmt"
	"strings"
	"time"

	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/logp"
	"github.com/elastic/libbeat/publisher"
	"github.com/elastic/packetbeat/config"
	"github.com/elastic/packetbeat/procs"
	"github.com/elastic/packetbeat/protos"
	"github.com/elastic/packetbeat/protos/tcp"
)

var debugf = logp.MakeDebug("mongodb")

type Mongodb struct {
	// config
	Ports        []int
	SendRequest  bool
	SendResponse bool
	MaxDocs      int
	MaxDocLength int

	transactions       *common.Cache
	transactionTimeout time.Duration

	results publisher.Client
}

func (mongodb *Mongodb) getTransaction(k common.HashableTcpTuple) *transaction {
	v := mongodb.transactions.Get(k)
	if v != nil {
		return v.(*transaction)
	}
	return nil
}

func (mongodb *Mongodb) InitDefaults() {
	mongodb.SendRequest = false
	mongodb.SendResponse = false
	mongodb.MaxDocs = 10
	mongodb.MaxDocLength = 5000
	mongodb.transactionTimeout = protos.DefaultTransactionExpiration
}

func (mongodb *Mongodb) setFromConfig(config config.Mongodb) error {
	mongodb.Ports = config.Ports

	if config.SendRequest != nil {
		mongodb.SendRequest = *config.SendRequest
	}
	if config.SendResponse != nil {
		mongodb.SendResponse = *config.SendResponse
	}
	if config.Max_docs != nil {
		mongodb.MaxDocs = *config.Max_docs
	}
	if config.Max_doc_length != nil {
		mongodb.MaxDocLength = *config.Max_doc_length
	}
	if config.TransactionTimeout != nil && *config.TransactionTimeout > 0 {
		mongodb.transactionTimeout = time.Duration(*config.TransactionTimeout) * time.Second
	}
	return nil
}

func (mongodb *Mongodb) GetPorts() []int {
	return mongodb.Ports
}

func (mongodb *Mongodb) Init(test_mode bool, results publisher.Client) error {
	debugf("Init a MongoDB protocol parser")

	mongodb.InitDefaults()
	if !test_mode {
		err := mongodb.setFromConfig(config.ConfigSingleton.Protocols.Mongodb)
		if err != nil {
			return err
		}
	}

	mongodb.transactions = common.NewCache(
		mongodb.transactionTimeout,
		protos.DefaultTransactionHashSize)
	mongodb.transactions.StartJanitor(mongodb.transactionTimeout)
	mongodb.results = results

	return nil
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
		logp.Warn("Unexpected: mongodb connection data not set, create new one")
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
		mongodb.handleMongodb(st.message, tcptuple, dir)
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

func (mongodb *Mongodb) handleMongodb(m *mongodbMessage, tcptuple *common.TcpTuple,
	dir uint8) {

	m.TcpTuple = *tcptuple
	m.Direction = dir
	m.CmdlineTuple = procs.ProcWatcher.FindProcessesTuple(tcptuple.IpPort())

	if m.IsResponse {
		debugf("MongoDB response message")
		mongodb.receivedMongodbResponse(m)
	} else {
		debugf("MongoDB request message")
		mongodb.receivedMongodbRequest(m)
	}
}

func (mongodb *Mongodb) receivedMongodbRequest(msg *mongodbMessage) {
	// Add it to the HT
	tuple := msg.TcpTuple

	trans := mongodb.getTransaction(tuple.Hashable())
	if trans != nil {
		if trans.Mongodb != nil {
			logp.Warn("Two requests without a Response. Dropping old request")
		}
	} else {
		debugf("Initialize new transaction from request")
		trans = &transaction{Type: "mongodb", tuple: tuple}
		mongodb.transactions.Put(tuple.Hashable(), trans)
	}

	trans.Mongodb = common.MapStr{}

	trans.event = msg.event

	trans.method = msg.method

	trans.cmdline = msg.CmdlineTuple
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
	trans.params = msg.params
	trans.resource = msg.resource
	trans.BytesIn = msg.messageLength
}

func (mongodb *Mongodb) receivedMongodbResponse(msg *mongodbMessage) {

	trans := mongodb.getTransaction(msg.TcpTuple.Hashable())
	if trans == nil {
		logp.Warn("Response from unknown transaction. Ignoring.")
		return
	}
	// check if the request was received
	if trans.Mongodb == nil {
		logp.Warn("Response from unknown transaction. Ignoring.")
		return

	}

	// Merge request and response events attributes
	for k, v := range msg.event {
		trans.event[k] = v
	}

	trans.error = msg.error
	trans.documents = msg.documents

	trans.ResponseTime = int32(msg.Ts.Sub(trans.ts).Nanoseconds() / 1e6) // resp_time in milliseconds
	trans.BytesOut = msg.messageLength

	mongodb.publishTransaction(trans)
	mongodb.transactions.Delete(trans.tuple.Hashable())

	debugf("Mongodb transaction completed: %s", trans.Mongodb)
}

func (mongodb *Mongodb) GapInStream(tcptuple *common.TcpTuple, dir uint8,
	nbytes int, private protos.ProtocolData) (priv protos.ProtocolData, drop bool) {

	// TODO

	return private, true
}

func (mongodb *Mongodb) ReceivedFin(tcptuple *common.TcpTuple, dir uint8,
	private protos.ProtocolData) protos.ProtocolData {

	// TODO
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

	mongodb.results.PublishEvent(event)
}
