package udp

import (
	"fmt"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/elastic/beats/packetbeat/flows"
	"github.com/elastic/beats/packetbeat/protos"
)

type Udp struct {
	protocols protos.Protocols
	portMap   map[uint16]protos.Protocol
}

type Processor interface {
	Process(id *flows.FlowID, pkt *protos.Packet)
}

// decideProtocol determines the protocol based on the source and destination
// ports. If the protocol cannot be determined then protos.UnknownProtocol
// is returned.
func (udp *Udp) decideProtocol(tuple *common.IpPortTuple) protos.Protocol {
	protocol, exists := udp.portMap[tuple.Src_port]
	if exists {
		return protocol
	}

	protocol, exists = udp.portMap[tuple.Dst_port]
	if exists {
		return protocol
	}

	return protos.UnknownProtocol
}

// Process handles UDP packets that have been received. It attempts to
// determine the protocol type and then invokes the associated
// UdpProtocolPlugin's ParseUdp method. If the protocol cannot be determined
// or the payload is empty then the method is a noop.
func (udp *Udp) Process(id *flows.FlowID, pkt *protos.Packet) {
	protocol := udp.decideProtocol(&pkt.Tuple)
	if protocol == protos.UnknownProtocol {
		logp.Debug("udp", "unknown protocol")
		return
	}

	plugin := udp.protocols.GetUdp(protocol)
	if plugin == nil {
		logp.Debug("udp", "Ignoring protocol for which we have no module loaded: %s", protocol)
		return
	}

	if len(pkt.Payload) > 0 {
		logp.Debug("udp", "Parsing packet from %v of length %d.",
			pkt.Tuple.String(), len(pkt.Payload))
		plugin.ParseUdp(pkt)
	}
}

// buildPortsMap creates a mapping of port numbers to protocol identifiers. If
// any two UdpProtocolPlugins operate on the same port number then an error
// will be returned.
func buildPortsMap(plugins map[protos.Protocol]protos.UdpPlugin) (map[uint16]protos.Protocol, error) {
	var res = map[uint16]protos.Protocol{}

	for proto, protoPlugin := range plugins {
		for _, port := range protoPlugin.GetPorts() {
			old_proto, exists := res[uint16(port)]
			if exists {
				if old_proto == proto {
					continue
				}
				return nil, fmt.Errorf("Duplicate port (%d) exists in %s and %s protocols",
					port, old_proto, proto)
			}
			res[uint16(port)] = proto
		}
	}

	return res, nil
}

// NewUdp creates and returns a new Udp.
func NewUdp(p protos.Protocols) (*Udp, error) {
	portMap, err := buildPortsMap(p.GetAllUdp())
	if err != nil {
		return nil, err
	}

	udp := &Udp{protocols: p, portMap: portMap}
	logp.Debug("udp", "Port map: %v", portMap)

	return udp, nil
}
