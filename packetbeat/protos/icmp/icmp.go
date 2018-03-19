package icmp

import (
	"net"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/monitoring"

	"github.com/elastic/beats/packetbeat/flows"
	"github.com/elastic/beats/packetbeat/protos"

	"github.com/tsg/gopacket/layers"
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

func New(testMode bool, results protos.Reporter, cfg *common.Config) (*icmpPlugin, error) {
	p := &icmpPlugin{}
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

func (icmp *icmpPlugin) init(results protos.Reporter, config *icmpConfig) error {
	icmp.setFromConfig(config)

	var err error
	icmp.localIps, err = common.LocalIPAddrs()
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
		length: len(icmp4.BaseLayer.Payload),
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
		length: len(icmp6.BaseLayer.Payload),
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

func (icmp *icmpPlugin) getTransaction(k hashableIcmpTuple) *icmpTransaction {
	v := icmp.transactions.Get(k)
	if v != nil {
		return v.(*icmpTransaction)
	}
	return nil
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

	fields := common.MapStr{}

	// common fields - group "env"
	fields["client_ip"] = trans.tuple.srcIP
	fields["ip"] = trans.tuple.dstIP

	// common fields - group "event"
	fields["type"] = "icmp"            // protocol name
	fields["path"] = trans.tuple.dstIP // what is requested (dst ip)
	if trans.HasError() {
		fields["status"] = common.ERROR_STATUS
	} else {
		fields["status"] = common.OK_STATUS
	}
	if len(trans.notes) > 0 {
		fields["notes"] = trans.notes
	}

	// common fields - group "measurements"
	responsetime, hasResponseTime := trans.ResponseTimeMillis()
	if hasResponseTime {
		fields["responsetime"] = responsetime
	}
	switch icmp.direction(trans) {
	case directionFromInside:
		if trans.request != nil {
			fields["bytes_out"] = trans.request.length
		}
		if trans.response != nil {
			fields["bytes_in"] = trans.response.length
		}
	case directionFromOutside:
		if trans.request != nil {
			fields["bytes_in"] = trans.request.length
		}
		if trans.response != nil {
			fields["bytes_out"] = trans.response.length
		}
	}

	// event fields - group "icmp"
	icmpEvent := common.MapStr{}
	fields["icmp"] = icmpEvent

	icmpEvent["version"] = trans.tuple.icmpVersion

	if trans.request != nil {
		request := common.MapStr{}
		icmpEvent["request"] = request

		request["message"] = humanReadable(&trans.tuple, trans.request)
		request["type"] = trans.request.Type
		request["code"] = trans.request.code

		// TODO: Add more info. The IPv4/IPv6 payload could be interesting.
		// if icmp.SendRequest {
		//     request["payload"] = ""
		// }
	}

	if trans.response != nil {
		response := common.MapStr{}
		icmpEvent["response"] = response

		response["message"] = humanReadable(&trans.tuple, trans.response)
		response["type"] = trans.response.Type
		response["code"] = trans.response.code

		// TODO: Add more info. The IPv4/IPv6 payload could be interesting.
		// if icmp.SendResponse {
		//     response["payload"] = ""
		// }
	}

	icmp.results(beat.Event{
		Timestamp: trans.ts,
		Fields:    fields,
	})
}
