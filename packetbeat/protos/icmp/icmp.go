package icmp

import (
	"expvar"
	"net"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/elastic/beats/packetbeat/flows"
	"github.com/elastic/beats/packetbeat/protos"
	"github.com/elastic/beats/packetbeat/publish"

	"github.com/tsg/gopacket/layers"
)

type ICMPv4Processor interface {
	ProcessICMPv4(flowID *flows.FlowID, hdr *layers.ICMPv4, pkt *protos.Packet)
}

type ICMPv6Processor interface {
	ProcessICMPv6(flowID *flows.FlowID, hdr *layers.ICMPv6, pkt *protos.Packet)
}

type Icmp struct {
	sendRequest  bool
	sendResponse bool

	localIps []net.IP

	// Active ICMP transactions.
	// The map key is the hashableIcmpTuple associated with the request.
	transactions       *common.Cache
	transactionTimeout time.Duration

	results publish.Transactions
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
	unmatchedRequests  = expvar.NewInt("icmp.unmatched_requests")
	unmatchedResponses = expvar.NewInt("icmp.unmatched_responses")
	duplicateRequests  = expvar.NewInt("icmp.duplicate_requests")
)

func New(testMode bool, results publish.Transactions, cfg *common.Config) (*Icmp, error) {
	p := &Icmp{}
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

func (icmp *Icmp) init(results publish.Transactions, config *icmpConfig) error {
	icmp.setFromConfig(config)

	var err error
	icmp.localIps, err = common.LocalIpAddrs()
	if err != nil {
		logp.Err("icmp", "Error getting local IP addresses: %s", err)
		icmp.localIps = []net.IP{}
	}
	logp.Debug("icmp", "Local IP addresses: %s", icmp.localIps)

	var removalListener = func(k common.Key, v common.Value) {
		icmp.expireTransaction(k.(hashableIcmpTuple), v.(*icmpTransaction))
	}

	icmp.transactions = common.NewCacheWithRemovalListener(
		icmp.transactionTimeout,
		protos.DefaultTransactionHashSize,
		removalListener)

	icmp.transactions.StartJanitor(icmp.transactionTimeout)

	icmp.results = results

	return nil
}

func (icmp *Icmp) setFromConfig(config *icmpConfig) {
	icmp.sendRequest = config.SendRequest
	icmp.sendResponse = config.SendResponse
	icmp.transactionTimeout = config.TransactionTimeout
}

func (icmp *Icmp) ProcessICMPv4(
	flowID *flows.FlowID,
	icmp4 *layers.ICMPv4,
	pkt *protos.Packet,
) {
	typ := uint8(icmp4.TypeCode >> 8)
	code := uint8(icmp4.TypeCode)
	id, seq := extractTrackingData(4, typ, &icmp4.BaseLayer)

	tuple := &icmpTuple{
		IcmpVersion: 4,
		SrcIp:       pkt.Tuple.Src_ip,
		DstIp:       pkt.Tuple.Dst_ip,
		Id:          id,
		Seq:         seq,
	}
	msg := &icmpMessage{
		Ts:     pkt.Ts,
		Type:   typ,
		Code:   code,
		Length: len(icmp4.BaseLayer.Payload),
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

func (icmp *Icmp) ProcessICMPv6(
	flowID *flows.FlowID,
	icmp6 *layers.ICMPv6,
	pkt *protos.Packet,
) {
	typ := uint8(icmp6.TypeCode >> 8)
	code := uint8(icmp6.TypeCode)
	id, seq := extractTrackingData(6, typ, &icmp6.BaseLayer)
	tuple := &icmpTuple{
		IcmpVersion: 6,
		SrcIp:       pkt.Tuple.Src_ip,
		DstIp:       pkt.Tuple.Dst_ip,
		Id:          id,
		Seq:         seq,
	}
	msg := &icmpMessage{
		Ts:     pkt.Ts,
		Type:   typ,
		Code:   code,
		Length: len(icmp6.BaseLayer.Payload),
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

func (icmp *Icmp) processRequest(tuple *icmpTuple, msg *icmpMessage) {
	logp.Debug("icmp", "Processing request. %s", tuple)

	trans := icmp.deleteTransaction(tuple.Hashable())
	if trans != nil {
		trans.Notes = append(trans.Notes, duplicateRequestMsg)
		logp.Debug("icmp", duplicateRequestMsg+" %s", tuple)
		duplicateRequests.Add(1)
		icmp.publishTransaction(trans)
	}

	trans = &icmpTransaction{Ts: msg.Ts, Tuple: *tuple}
	trans.Request = msg

	if requiresCounterpart(tuple, msg) {
		icmp.transactions.Put(tuple.Hashable(), trans)
	} else {
		icmp.publishTransaction(trans)
	}
}

func (icmp *Icmp) processResponse(tuple *icmpTuple, msg *icmpMessage) {
	logp.Debug("icmp", "Processing response. %s", tuple)

	revTuple := tuple.Reverse()
	trans := icmp.deleteTransaction(revTuple.Hashable())
	if trans == nil {
		trans = &icmpTransaction{Ts: msg.Ts, Tuple: revTuple}
		trans.Notes = append(trans.Notes, orphanedResponseMsg)
		logp.Debug("icmp", orphanedResponseMsg+" %s", tuple)
		unmatchedResponses.Add(1)
	}

	trans.Response = msg
	icmp.publishTransaction(trans)
}

func (icmp *Icmp) direction(t *icmpTransaction) uint8 {
	if !icmp.isLocalIp(t.Tuple.SrcIp) {
		return directionFromOutside
	}
	if !icmp.isLocalIp(t.Tuple.DstIp) {
		return directionFromInside
	}
	return directionLocalOnly
}

func (icmp *Icmp) isLocalIp(ip net.IP) bool {
	if ip.IsLoopback() {
		return true
	}

	for _, localIp := range icmp.localIps {
		if ip.Equal(localIp) {
			return true
		}
	}

	return false
}

func (icmp *Icmp) getTransaction(k hashableIcmpTuple) *icmpTransaction {
	v := icmp.transactions.Get(k)
	if v != nil {
		return v.(*icmpTransaction)
	}
	return nil
}

func (icmp *Icmp) deleteTransaction(k hashableIcmpTuple) *icmpTransaction {
	v := icmp.transactions.Delete(k)
	if v != nil {
		return v.(*icmpTransaction)
	}
	return nil
}

func (icmp *Icmp) expireTransaction(tuple hashableIcmpTuple, trans *icmpTransaction) {
	trans.Notes = append(trans.Notes, orphanedRequestMsg)
	logp.Debug("icmp", orphanedRequestMsg+" %s", &trans.Tuple)
	unmatchedRequests.Add(1)
	icmp.publishTransaction(trans)
}

func (icmp *Icmp) publishTransaction(trans *icmpTransaction) {
	if icmp.results == nil {
		return
	}

	logp.Debug("icmp", "Publishing transaction. %s", &trans.Tuple)

	event := common.MapStr{}

	// common fields - group "env"
	event["client_ip"] = trans.Tuple.SrcIp
	event["ip"] = trans.Tuple.DstIp

	// common fields - group "event"
	event["@timestamp"] = common.Time(trans.Ts) // timestamp of the first packet
	event["type"] = "icmp"                      // protocol name
	event["path"] = trans.Tuple.DstIp           // what is requested (dst ip)
	if trans.HasError() {
		event["status"] = common.ERROR_STATUS
	} else {
		event["status"] = common.OK_STATUS
	}
	if len(trans.Notes) > 0 {
		event["notes"] = trans.Notes
	}

	// common fields - group "measurements"
	responsetime, hasResponseTime := trans.ResponseTimeMillis()
	if hasResponseTime {
		event["responsetime"] = responsetime
	}
	switch icmp.direction(trans) {
	case directionFromInside:
		if trans.Request != nil {
			event["bytes_out"] = trans.Request.Length
		}
		if trans.Response != nil {
			event["bytes_in"] = trans.Response.Length
		}
	case directionFromOutside:
		if trans.Request != nil {
			event["bytes_in"] = trans.Request.Length
		}
		if trans.Response != nil {
			event["bytes_out"] = trans.Response.Length
		}
	}

	// event fields - group "icmp"
	icmpEvent := common.MapStr{}
	event["icmp"] = icmpEvent

	icmpEvent["version"] = trans.Tuple.IcmpVersion

	if trans.Request != nil {
		request := common.MapStr{}
		icmpEvent["request"] = request

		request["message"] = humanReadable(&trans.Tuple, trans.Request)
		request["type"] = trans.Request.Type
		request["code"] = trans.Request.Code

		// TODO: Add more info. The IPv4/IPv6 payload could be interesting.
		// if icmp.SendRequest {
		//     request["payload"] = ""
		// }
	}

	if trans.Response != nil {
		response := common.MapStr{}
		icmpEvent["response"] = response

		response["message"] = humanReadable(&trans.Tuple, trans.Response)
		response["type"] = trans.Response.Type
		response["code"] = trans.Response.Code

		// TODO: Add more info. The IPv4/IPv6 payload could be interesting.
		// if icmp.SendResponse {
		//     response["payload"] = ""
		// }
	}

	icmp.results.PublishTransaction(event)
}
