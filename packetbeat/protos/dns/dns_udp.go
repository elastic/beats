package dns

import (
	"github.com/elastic/beats/libbeat/logp"

	"github.com/elastic/beats/packetbeat/procs"
	"github.com/elastic/beats/packetbeat/protos"
)

func (dns *Dns) ParseUdp(pkt *protos.Packet) {
	defer logp.Recover("Dns ParseUdp")

	logp.Debug("dns", "Parsing packet addressed with %s of length %d.",
		pkt.Tuple.String(), len(pkt.Payload))

	dnsPkt, err := decodeDnsData(TransportUdp, pkt.Payload)
	if err != nil {
		// This means that malformed requests or responses are being sent or
		// that someone is attempting to the DNS port for non-DNS traffic. Both
		// are issues that a monitoring system should report.
		logp.Debug("dns", err.Error())
		return
	}

	dnsTuple := DnsTupleFromIpPort(&pkt.Tuple, TransportUdp, dnsPkt.ID)
	dnsMsg := &DnsMessage{
		Ts:           pkt.Ts,
		Tuple:        pkt.Tuple,
		CmdlineTuple: procs.ProcWatcher.FindProcessesTuple(&pkt.Tuple),
		Data:         dnsPkt,
		Length:       len(pkt.Payload),
	}

	if dnsMsg.Data.QR == Query {
		dns.receivedDnsRequest(&dnsTuple, dnsMsg)
	} else /* Response */ {
		dns.receivedDnsResponse(&dnsTuple, dnsMsg)
	}
}
