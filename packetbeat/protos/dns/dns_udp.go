package dns

import (
	"github.com/elastic/beats/libbeat/logp"

	"github.com/elastic/beats/packetbeat/procs"
	"github.com/elastic/beats/packetbeat/protos"
)

// Only EDNS packets should have their size beyond this value
const MaxDnsPacketSize = (1 << 9) // 512 (bytes)

func (dns *Dns) ParseUdp(pkt *protos.Packet) {
	defer logp.Recover("Dns ParseUdp")
	packetSize := len(pkt.Payload)

	debugf("Parsing packet addressed with %s of length %d.",
		pkt.Tuple.String(), packetSize)

	dnsPkt, err := decodeDnsData(TransportUdp, pkt.Payload)
	if err != nil {
		// This means that malformed requests or responses are being sent or
		// that someone is attempting to the DNS port for non-DNS traffic. Both
		// are issues that a monitoring system should report.
		debugf("%s", err.Error())
		return
	}

	dnsTuple := DnsTupleFromIpPort(&pkt.Tuple, TransportUdp, dnsPkt.Id)
	dnsMsg := &DnsMessage{
		Ts:           pkt.Ts,
		Tuple:        pkt.Tuple,
		CmdlineTuple: procs.ProcWatcher.FindProcessesTuple(&pkt.Tuple),
		Data:         dnsPkt,
		Length:       packetSize,
	}

	if dnsMsg.Data.Response {
		dns.receivedDnsResponse(&dnsTuple, dnsMsg)
	} else /* Query */ {
		dns.receivedDnsRequest(&dnsTuple, dnsMsg)
	}
}
