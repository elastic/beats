package dns

import (
	"github.com/elastic/beats/libbeat/logp"

	"github.com/elastic/beats/packetbeat/procs"
	"github.com/elastic/beats/packetbeat/protos"
)

// Only EDNS packets should have their size beyond this value
const maxDNSPacketSize = (1 << 9) // 512 (bytes)

func (dns *dnsPlugin) ParseUDP(pkt *protos.Packet) {
	defer logp.Recover("Dns ParseUdp")
	packetSize := len(pkt.Payload)

	debugf("Parsing packet addressed with %s of length %d.",
		pkt.Tuple.String(), packetSize)

	dnsPkt, err := decodeDNSData(transportUDP, pkt.Payload)
	if err != nil {
		// This means that malformed requests or responses are being sent or
		// that someone is attempting to the DNS port for non-DNS traffic. Both
		// are issues that a monitoring system should report.
		debugf("%s", err.Error())
		return
	}

	dnsTuple := dnsTupleFromIPPort(&pkt.Tuple, transportUDP, dnsPkt.Id)
	dnsMsg := &dnsMessage{
		ts:           pkt.Ts,
		tuple:        pkt.Tuple,
		cmdlineTuple: procs.ProcWatcher.FindProcessesTuple(&pkt.Tuple),
		data:         dnsPkt,
		length:       packetSize,
	}

	if dnsMsg.data.Response {
		dns.receivedDNSResponse(&dnsTuple, dnsMsg)
	} else /* Query */ {
		dns.receivedDNSRequest(&dnsTuple, dnsMsg)
	}
}
