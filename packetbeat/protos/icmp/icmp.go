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

package icmp

import (
	"net"
	"time"

	"github.com/google/gopacket/layers"

	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/ecs"
	"github.com/menderesk/beats/v7/libbeat/logp"
	"github.com/menderesk/beats/v7/libbeat/monitoring"

	"github.com/menderesk/beats/v7/packetbeat/flows"
	"github.com/menderesk/beats/v7/packetbeat/pb"
	"github.com/menderesk/beats/v7/packetbeat/procs"
	"github.com/menderesk/beats/v7/packetbeat/protos"
)

type icmpPlugin struct {
	sendRequest  bool
	sendResponse bool

	localIps []net.IP

	// Active ICMP transactions.
	// The map key is the hashableIcmpTuple associated with the request.
	transactions       *common.Cache
	transactionTimeout time.Duration

	results protos.Reporter
	watcher procs.ProcessesWatcher
}

type ICMPv4Processor interface {
	ProcessICMPv4(flowID *flows.FlowID, hdr *layers.ICMPv4, pkt *protos.Packet)
}

type ICMPv6Processor interface {
	ProcessICMPv6(flowID *flows.FlowID, hdr *layers.ICMPv6, pkt *protos.Packet)
}

const (
	directionLocalOnly = iota
	directionFromInside
	directionFromOutside
)

// Notes that are added to messages during exceptional conditions.
const (
	duplicateRequestMsg = "Another request with the same Id and Seq was received so this request was closed without receiving a response."
	orphanedRequestMsg  = "Request was received without an associated response."
	orphanedResponseMsg = "Response was received without an associated request."
)

var (
	unmatchedRequests  = monitoring.NewInt(nil, "icmp.unmatched_requests")
	unmatchedResponses = monitoring.NewInt(nil, "icmp.unmatched_responses")
	duplicateRequests  = monitoring.NewInt(nil, "icmp.duplicate_requests")
)

func New(testMode bool, results protos.Reporter, watcher procs.ProcessesWatcher, cfg *common.Config) (*icmpPlugin, error) {
	p := &icmpPlugin{}
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

func (icmp *icmpPlugin) init(results protos.Reporter, watcher procs.ProcessesWatcher, config *icmpConfig) error {
	icmp.setFromConfig(config)

	var err error
	icmp.localIps, err = common.LocalIPAddrs()
	if err != nil {
		logp.Err("Error getting local IP addresses: %+v", err)
		icmp.localIps = []net.IP{}
	}
	logp.Debug("icmp", "Local IP addresses: %s", icmp.localIps)

	removalListener := func(k common.Key, v common.Value) {
		icmp.expireTransaction(k.(hashableIcmpTuple), v.(*icmpTransaction))
	}

	icmp.transactions = common.NewCacheWithRemovalListener(
		icmp.transactionTimeout,
		protos.DefaultTransactionHashSize,
		removalListener)

	icmp.transactions.StartJanitor(icmp.transactionTimeout)

	icmp.results = results
	icmp.watcher = watcher

	return nil
}

func (icmp *icmpPlugin) setFromConfig(config *icmpConfig) {
	icmp.sendRequest = config.SendRequest
	icmp.sendResponse = config.SendResponse
	icmp.transactionTimeout = config.TransactionTimeout
}

func (icmp *icmpPlugin) ProcessICMPv4(
	flowID *flows.FlowID,
	icmp4 *layers.ICMPv4,
	pkt *protos.Packet,
) {
	typ := uint8(icmp4.TypeCode >> 8)
	code := uint8(icmp4.TypeCode)
	id, seq := extractTrackingData(4, typ, &icmp4.BaseLayer)

	tuple := &icmpTuple{
		icmpVersion: 4,
		srcIP:       pkt.Tuple.SrcIP,
		dstIP:       pkt.Tuple.DstIP,
		id:          id,
		seq:         seq,
	}
	msg := &icmpMessage{
		ts:     pkt.Ts,
		Type:   typ,
		code:   code,
		length: len(icmp4.Payload),
	}

	if isRequest(tuple, msg) {
		if flowID != nil {
			flowID.AddICMPv4Request(id)
		}
		icmp.processRequest(tuple, msg)
	} else {
		if flowID != nil {
			flowID.AddICMPv4Response(id)
		}
		icmp.processResponse(tuple, msg)
	}
}

func (icmp *icmpPlugin) ProcessICMPv6(
	flowID *flows.FlowID,
	icmp6 *layers.ICMPv6,
	pkt *protos.Packet,
) {
	typ := uint8(icmp6.TypeCode >> 8)
	code := uint8(icmp6.TypeCode)
	id, seq := extractTrackingData(6, typ, &icmp6.BaseLayer)
	tuple := &icmpTuple{
		icmpVersion: 6,
		srcIP:       pkt.Tuple.SrcIP,
		dstIP:       pkt.Tuple.DstIP,
		id:          id,
		seq:         seq,
	}
	msg := &icmpMessage{
		ts:     pkt.Ts,
		Type:   typ,
		code:   code,
		length: len(icmp6.Payload),
	}

	if isRequest(tuple, msg) {
		if flowID != nil {
			flowID.AddICMPv6Request(id)
		}
		icmp.processRequest(tuple, msg)
	} else {
		if flowID != nil {
			flowID.AddICMPv6Response(id)
		}
		icmp.processResponse(tuple, msg)
	}
}

func (icmp *icmpPlugin) processRequest(tuple *icmpTuple, msg *icmpMessage) {
	logp.Debug("icmp", "Processing request. %s", tuple)

	trans := icmp.deleteTransaction(tuple.Hashable())
	if trans != nil {
		trans.notes = append(trans.notes, duplicateRequestMsg)
		logp.Debug("icmp", duplicateRequestMsg+" %s", tuple)
		duplicateRequests.Add(1)
		icmp.publishTransaction(trans)
	}

	trans = &icmpTransaction{ts: msg.ts, tuple: *tuple}
	trans.request = msg

	if requiresCounterpart(tuple, msg) {
		icmp.transactions.Put(tuple.Hashable(), trans)
	} else {
		icmp.publishTransaction(trans)
	}
}

func (icmp *icmpPlugin) processResponse(tuple *icmpTuple, msg *icmpMessage) {
	logp.Debug("icmp", "Processing response. %s", tuple)

	revTuple := tuple.Reverse()
	trans := icmp.deleteTransaction(revTuple.Hashable())
	if trans == nil {
		trans = &icmpTransaction{ts: msg.ts, tuple: revTuple}
		trans.notes = append(trans.notes, orphanedResponseMsg)
		logp.Debug("icmp", orphanedResponseMsg+" %s", tuple)
		unmatchedResponses.Add(1)
	}

	trans.response = msg
	icmp.publishTransaction(trans)
}

func (icmp *icmpPlugin) direction(t *icmpTransaction) uint8 {
	if !icmp.isLocalIP(t.tuple.srcIP) {
		return directionFromOutside
	}
	if !icmp.isLocalIP(t.tuple.dstIP) {
		return directionFromInside
	}
	return directionLocalOnly
}

func (icmp *icmpPlugin) isLocalIP(ip net.IP) bool {
	if ip.IsLoopback() {
		return true
	}

	for _, localIP := range icmp.localIps {
		if ip.Equal(localIP) {
			return true
		}
	}

	return false
}

func (icmp *icmpPlugin) deleteTransaction(k hashableIcmpTuple) *icmpTransaction {
	v := icmp.transactions.Delete(k)
	if v != nil {
		return v.(*icmpTransaction)
	}
	return nil
}

func (icmp *icmpPlugin) expireTransaction(tuple hashableIcmpTuple, trans *icmpTransaction) {
	trans.notes = append(trans.notes, orphanedRequestMsg)
	logp.Debug("icmp", orphanedRequestMsg+" %s", &trans.tuple)
	unmatchedRequests.Add(1)
	icmp.publishTransaction(trans)
}

func (icmp *icmpPlugin) publishTransaction(trans *icmpTransaction) {
	if icmp.results == nil {
		return
	}

	logp.Debug("icmp", "Publishing transaction. %s", &trans.tuple)

	evt, pbf := pb.NewBeatEvent(trans.ts)
	pbf.Source = &ecs.Source{IP: trans.tuple.srcIP.String()}
	pbf.Destination = &ecs.Destination{IP: trans.tuple.dstIP.String()}
	pbf.AddIP(trans.tuple.srcIP.String(), trans.tuple.dstIP.String())
	pbf.Event.Dataset = "icmp"
	pbf.Event.Type = []string{"connection"}
	pbf.Error.Message = trans.notes

	// common fields - group "event"
	fields := evt.Fields
	fields["type"] = pbf.Event.Dataset
	fields["path"] = trans.tuple.dstIP // what is requested (dst ip)
	if trans.HasError() {
		fields["status"] = common.ERROR_STATUS
	} else {
		fields["status"] = common.OK_STATUS
	}

	icmpEvent := common.MapStr{
		"version": trans.tuple.icmpVersion,
	}
	fields["icmp"] = icmpEvent

	pbf.Network.Transport = pbf.Event.Dataset
	if trans.tuple.icmpVersion == 6 {
		pbf.Network.Transport = "ipv6-icmp"
	}

	if trans.request != nil {
		pbf.Event.Start = trans.request.ts
		pbf.Source.Bytes = int64(trans.request.length)

		request := common.MapStr{
			"message": humanReadable(&trans.tuple, trans.request),
			"type":    trans.request.Type,
			"code":    trans.request.code,
		}
		icmpEvent["request"] = request

		pbf.ICMPType = trans.request.Type
		pbf.ICMPCode = trans.request.code
	}

	if trans.response != nil {
		pbf.Event.End = trans.response.ts
		pbf.Destination.Bytes = int64(trans.response.length)

		response := common.MapStr{
			"message": humanReadable(&trans.tuple, trans.response),
			"type":    trans.response.Type,
			"code":    trans.response.code,
		}
		icmpEvent["response"] = response

		if trans.request == nil {
			pbf.ICMPType = trans.response.Type
			pbf.ICMPCode = trans.response.code
		}
	}

	icmp.results(evt)
}
