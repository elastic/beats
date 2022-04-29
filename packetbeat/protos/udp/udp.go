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

package udp

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/elastic-agent-libs/logp"

	"github.com/elastic/beats/v7/packetbeat/flows"
	"github.com/elastic/beats/v7/packetbeat/protos"
)

type UDP struct {
	protocols protos.Protocols
	portMap   map[uint16]protos.Protocol
}

type Processor interface {
	Process(id *flows.FlowID, pkt *protos.Packet)
}

// decideProtocol determines the protocol based on the source and destination
// ports. If the protocol cannot be determined then protos.UnknownProtocol
// is returned.
func (udp *UDP) decideProtocol(tuple *common.IPPortTuple) protos.Protocol {
	protocol, exists := udp.portMap[tuple.SrcPort]
	if exists {
		return protocol
	}

	protocol, exists = udp.portMap[tuple.DstPort]
	if exists {
		return protocol
	}

	return protos.UnknownProtocol
}

// Process handles UDP packets that have been received. It attempts to
// determine the protocol type and then invokes the associated
// UdpProtocolPlugin's ParseUDP method. If the protocol cannot be determined
// or the payload is empty then the method is a noop.
func (udp *UDP) Process(id *flows.FlowID, pkt *protos.Packet) {
	protocol := udp.decideProtocol(&pkt.Tuple)
	if protocol == protos.UnknownProtocol {
		logp.Debug("udp", "unknown protocol")
		return
	}

	plugin := udp.protocols.GetUDP(protocol)
	if plugin == nil {
		logp.Debug("udp", "Ignoring protocol for which we have no module loaded: %s", protocol)
		return
	}

	if len(pkt.Payload) > 0 {
		logp.Debug("udp", "Parsing packet from %v of length %d.",
			pkt.Tuple.String(), len(pkt.Payload))
		plugin.ParseUDP(pkt)
	}
}

// buildPortsMap creates a mapping of port numbers to protocol identifiers. If
// any two UdpProtocolPlugins operate on the same port number then an error
// will be returned.
func buildPortsMap(plugins map[protos.Protocol]protos.UDPPlugin) (map[uint16]protos.Protocol, error) {
	res := map[uint16]protos.Protocol{}

	for proto, protoPlugin := range plugins {
		for _, port := range protoPlugin.GetPorts() {
			oldProto, exists := res[uint16(port)]
			if exists {
				if oldProto == proto {
					continue
				}
				return nil, fmt.Errorf("Duplicate port (%d) exists in %s and %s protocols",
					port, oldProto, proto)
			}
			res[uint16(port)] = proto
		}
	}

	return res, nil
}

// NewUDP creates and returns a new UDP.
func NewUDP(p protos.Protocols) (*UDP, error) {
	portMap, err := buildPortsMap(p.GetAllUDP())
	if err != nil {
		return nil, err
	}

	udp := &UDP{protocols: p, portMap: portMap}
	logp.Debug("udp", "Port map: %v", portMap)

	return udp, nil
}
