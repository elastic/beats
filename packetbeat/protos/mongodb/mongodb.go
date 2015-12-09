package mongodb

import (
	"fmt"
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

type Mongodb struct {
	// config
	Ports          []int
	Send_request   bool
	Send_response  bool
	Max_docs       int
	Max_doc_length int

	transactions       *common.Cache
	transactionTimeout time.Duration

	results publisher.Client
}

func (mongodb *Mongodb) getTransaction(k common.HashableTcpTuple) *MongodbTransaction {
	v := mongodb.transactions.Get(k)
	if v != nil {
		return v.(*MongodbTransaction)
	}
	return nil
}

func (mongodb *Mongodb) InitDefaults() {
	mongodb.Send_request = false
	mongodb.Send_response = false
	mongodb.Max_docs = 10
	mongodb.Max_doc_length = 5000
	mongodb.transactionTimeout = protos.DefaultTransactionExpiration
}

func (mongodb *Mongodb) setFromConfig(config config.Mongodb) error {
	mongodb.Ports = config.Ports

	if config.SendRequest != nil {
		mongodb.Send_request = *config.SendRequest
	}
	if config.SendResponse != nil {
		mongodb.Send_response = *config.SendResponse
	}
	if config.Max_docs != nil {
		mongodb.Max_docs = *config.Max_docs
	}
	if config.Max_doc_length != nil {
		mongodb.Max_doc_length = *config.Max_doc_length
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
	logp.Debug("mongodb", "Init a MongoDB protocol parser")

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

func (mongodb *Mongodb) Parse(pkt *protos.Packet, tcptuple *common.TcpTuple, dir uint8,
	private protos.ProtocolData) protos.ProtocolData {

	logp.Debug("mongodb", "Parse method triggered")

	defer logp.Recover("ParseMongodb exception")

	// Either fetch or initialize current data struct for this parser
	priv := mongodbPrivateData{}
	if private != nil {
		var ok bool
		priv, ok = private.(mongodbPrivateData)
		if !ok {
			priv = mongodbPrivateData{}
		}
	}

	if priv.Data[dir] == nil {
		priv.Data[dir] = &MongodbStream{
			tcptuple: tcptuple,
			data:     pkt.Payload,
			message:  &MongodbMessage{Ts: pkt.Ts},
		}
	} else {
		// concatenate bytes
		priv.Data[dir].data = append(priv.Data[dir].data, pkt.Payload...)
		if len(priv.Data[dir].data) > tcp.TCP_MAX_DATA_IN_STREAM {
			logp.Debug("mongodb", "Stream data too large, dropping TCP stream")
			priv.Data[dir] = nil
			return priv
		}
	}

	stream := priv.Data[dir]
	for len(stream.data) > 0 {
		if stream.message == nil {
			stream.message = &MongodbMessage{Ts: pkt.Ts}
		}

		ok, complete := mongodbMessageParser(priv.Data[dir])

		if !ok {
			// drop this tcp stream. Will retry parsing with the next
			// segment in it
			priv.Data[dir] = nil
			logp.Debug("mongodb", "Ignore Mongodb message. Drop tcp stream. Try parsing with the next segment")
			return priv
		}

		if complete {

			logp.Debug("mongodb", "MongoDB message complete")

			// all ok, go to next level
			mongodb.handleMongodb(stream.message, tcptuple, dir)

			// and reset message
			stream.PrepareForNewMessage()
		} else {
			// wait for more data
			logp.Debug("mongodb", "MongoDB wait for more data before parsing message")
			break
		}
	}

	return priv
}

func (mongodb *Mongodb) handleMongodb(m *MongodbMessage, tcptuple *common.TcpTuple,
	dir uint8) {

	m.TcpTuple = *tcptuple
	m.Direction = dir
	m.CmdlineTuple = procs.ProcWatcher.FindProcessesTuple(tcptuple.IpPort())

	if m.IsResponse {
		logp.Debug("mongodb", "MongoDB response message")
		mongodb.receivedMongodbResponse(m)
	} else {
		logp.Debug("mongodb", "MongoDB request message")
		mongodb.receivedMongodbRequest(m)
	}
}

func (mongodb *Mongodb) receivedMongodbRequest(msg *MongodbMessage) {
	// Add it to the HT
	tuple := msg.TcpTuple

	trans := mongodb.getTransaction(tuple.Hashable())
	if trans != nil {
		if trans.Mongodb != nil {
			logp.Warn("Two requests without a Response. Dropping old request")
		}
	} else {
		logp.Debug("mongodb", "Initialize new transaction from request")
		trans = &MongodbTransaction{Type: "mongodb", tuple: tuple}
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

func (mongodb *Mongodb) receivedMongodbResponse(msg *MongodbMessage) {

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

	logp.Debug("mongodb", "Mongodb transaction completed: %s", trans.Mongodb)
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

func reconstructQuery(t *MongodbTransaction, full bool) (query string) {
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
			logp.Debug("mongodb", "Error marshaling params: %v", err)
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

func (mongodb *Mongodb) publishTransaction(t *MongodbTransaction) {

	if mongodb.results == nil {
		logp.Debug("mongodb", "Try to publish transaction with null results")
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

	if mongodb.Send_request {
		event["request"] = reconstructQuery(t, true)
	}
	if mongodb.Send_response {
		if len(t.documents) > 0 {
			// response field needs to be a string
			docs := make([]string, 0, len(t.documents))
			for i, doc := range t.documents {
				if mongodb.Max_docs > 0 && i >= mongodb.Max_docs {
					docs = append(docs, "[...]")
					break
				}
				str, err := doc2str(doc)
				if err != nil {
					logp.Warn("Failed to JSON marshal document from Mongo: %v (error: %v)", doc, err)
				} else {
					if mongodb.Max_doc_length > 0 && len(str) > mongodb.Max_doc_length {
						str = str[:mongodb.Max_doc_length] + " ..."
					}
					docs = append(docs, str)
				}
			}
			event["response"] = strings.Join(docs, "\n")
		}
	}

	mongodb.results.PublishEvent(event)
}
