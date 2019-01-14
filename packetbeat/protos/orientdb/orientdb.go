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

package orientdb

import (
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/monitoring"
	"github.com/elastic/beats/packetbeat/procs"
	"github.com/elastic/beats/packetbeat/protos"
	"github.com/elastic/beats/packetbeat/protos/tcp"
)

var debugf = logp.MakeDebug("orientdb")

type orientdbPlugin struct {
	// config
	ports        []int
	sendRequest  bool
	sendResponse bool

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
	unmatchedRequests = monitoring.NewInt(nil, "orientdb.unmatched_requests")
)

func init() {
	protos.Register("orientdb", New)
}

// New OrientDB plugin
func New(
	testMode bool,
	results protos.Reporter,
	cfg *common.Config,
) (protos.Plugin, error) {
	p := &orientdbPlugin{}
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

func (orientdb *orientdbPlugin) init(results protos.Reporter, config *orientdbConfig) error {
	debugf("Init an OrientDB binary protocol parser")
	orientdb.setFromConfig(config)

	orientdb.requests = common.NewCache(
		orientdb.transactionTimeout,
		protos.DefaultTransactionHashSize)
	orientdb.requests.StartJanitor(orientdb.transactionTimeout)
	orientdb.responses = common.NewCache(
		orientdb.transactionTimeout,
		protos.DefaultTransactionHashSize)
	orientdb.responses.StartJanitor(orientdb.transactionTimeout)
	orientdb.results = results

	return nil
}

func (orientdb *orientdbPlugin) setFromConfig(config *orientdbConfig) {
	orientdb.ports = config.Ports
	orientdb.sendRequest = config.SendRequest
	orientdb.sendResponse = config.SendResponse
	orientdb.transactionTimeout = config.TransactionTimeout
}

func (orientdb *orientdbPlugin) GetPorts() []int {
	return orientdb.ports
}

func (orientdb *orientdbPlugin) ConnectionTimeout() time.Duration {
	return orientdb.transactionTimeout
}

func (orientdb *orientdbPlugin) Parse(
	pkt *protos.Packet,
	tcptuple *common.TCPTuple,
	dir uint8,
	private protos.ProtocolData,
) protos.ProtocolData {
	defer logp.Recover("ParseOrientdb exception")
	debugf("Parse method triggered")

	conn := ensureOrientdbConnection(private)
	conn = orientdb.doParse(conn, pkt, tcptuple, dir)
	if conn == nil {
		return nil
	}
	return conn
}

func ensureOrientdbConnection(private protos.ProtocolData) *orientdbConnectionData {
	if private == nil {
		return &orientdbConnectionData{}
	}

	priv, ok := private.(*orientdbConnectionData)
	if !ok {
		logp.Warn("orientdb connection data type error, create new one")
		return &orientdbConnectionData{}
	}
	if priv == nil {
		debugf("Unexpected: orientdb connection data not set, create new one")
		return &orientdbConnectionData{}
	}
	return priv
}

func (orientdb *orientdbPlugin) doParse(
	conn *orientdbConnectionData,
	pkt *protos.Packet,
	tcptuple *common.TCPTuple,
	dir uint8,
) *orientdbConnectionData {
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
			st.message = &orientdbMessage{ts: pkt.Ts}
		}

		ok, complete := orientdbMessageParser(st)
		if !ok {
			// drop this tcp stream. Will retry parsing with the next
			// segment in it
			conn.streams[dir] = nil
			debugf("Ignore Orientdb message. Drop tcp stream. Try parsing with the next segment")
			return conn
		}

		if !complete {
			// wait for more data
			debugf("OrientDB wait for more data before parsing message")
			break
		}

		// all ok, go to next level and reset stream for new message
		debugf("OrientDB message complete")
		orientdb.handleOrientdb(conn, st.message, tcptuple, dir)
		st.PrepareForNewMessage()
	}

	return conn
}

func newStream(pkt *protos.Packet, tcptuple *common.TCPTuple) *stream {
	s := &stream{
		tcpTuple: tcptuple,
		data:     pkt.Payload,
		message:  &orientdbMessage{ts: pkt.Ts},
	}
	return s
}

func (orientdb *orientdbPlugin) handleOrientdb(
	conn *orientdbConnectionData,
	o *orientdbMessage,
	tcptuple *common.TCPTuple,
	dir uint8,
) {
	o.tcpTuple = *tcptuple
	o.direction = dir
	o.cmdlineTuple = procs.ProcWatcher.FindProcessesTupleTCP(tcptuple.IPPort())

	debugf("OrientDB request message")
	orientdb.onRequest(conn, o)
}

func (orientdb *orientdbPlugin) onRequest(conn *orientdbConnectionData, msg *orientdbMessage) {
	// publish request only transaction
	if !awaitsReply(msg.opCode) {
		orientdb.onTransComplete(msg, nil)
		return
	}

	id := msg.sessionID
	key := transactionKey{tcp: msg.tcpTuple.Hashable(), id: id}

	// try to find matching response potentially inserted before
	if v := orientdb.responses.Delete(key); v != nil {
		resp := v.(*orientdbMessage)
		orientdb.onTransComplete(msg, resp)
		return
	}

	// insert into cache for correlation
	old := orientdb.requests.Put(key, msg)
	if old != nil {
		debugf("Two requests without a Response. Dropping old request")
		unmatchedRequests.Add(1)
	}
}

func (orientdb *orientdbPlugin) onTransComplete(requ, resp *orientdbMessage) {
	trans := newTransaction(requ, resp)
	debugf("Orientdb transaction completed: %s", trans.orientdb)

	orientdb.publishTransaction(trans)
}

func newTransaction(requ, resp *orientdbMessage) *transaction {
	trans := &transaction{}

	// fill request
	if requ != nil {
		trans.orientdb = common.MapStr{}
		trans.event = requ.event
		trans.method = requ.method

		trans.cmdline = requ.cmdlineTuple
		trans.ts = requ.ts
		trans.src, trans.dst = common.MakeEndpointPair(requ.tcpTuple.BaseTuple, requ.cmdlineTuple)
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

		trans.responseTime = int32(resp.ts.Sub(trans.ts).Nanoseconds() / 1e6) // resp_time in milliseconds
		trans.bytesOut = resp.messageLength

	}

	return trans
}

func (orientdb *orientdbPlugin) GapInStream(tcptuple *common.TCPTuple, dir uint8,
	nbytes int, private protos.ProtocolData) (priv protos.ProtocolData, drop bool) {
	return private, true
}

func (orientdb *orientdbPlugin) ReceivedFin(tcptuple *common.TCPTuple, dir uint8,
	private protos.ProtocolData) protos.ProtocolData {
	return private
}

func (orientdb *orientdbPlugin) publishTransaction(t *transaction) {
	if orientdb.results == nil {
		debugf("Try to publish transaction with null results")
		return
	}

	timestamp := t.ts
	fields := common.MapStr{}
	fields["type"] = "orientdb"
	if t.error == "" {
		fields["status"] = common.OK_STATUS
	} else {
		t.event["error"] = t.error
		fields["status"] = common.ERROR_STATUS
	}
	fields["orientdb"] = t.event
	fields["method"] = t.method
	fields["resource"] = t.resource
	if t.resource != "" {
		fields["query"] = t.event["query"]
	}
	fields["responsetime"] = t.responseTime
	fields["bytes_in"] = uint64(t.bytesIn)
	fields["bytes_out"] = uint64(t.bytesOut)
	fields["src"] = &t.src
	fields["dst"] = &t.dst

	orientdb.results(beat.Event{
		Timestamp: timestamp,
		Fields:    fields,
	})
}
