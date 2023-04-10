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
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/rcrowley/go-metrics"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/monitoring/inputmon"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-libs/monitoring/adapter"

	"github.com/elastic/beats/v7/packetbeat/flows"
	"github.com/elastic/beats/v7/packetbeat/protos"
)

type Processor interface {
	Process(id *flows.FlowID, pkt *protos.Packet)
}

type UDP struct {
	protocols protos.Protocols
	portMap   map[uint16]protos.Protocol

	metrics *inputMetrics
}

// NewUDP creates and returns a new UDP.
func NewUDP(p protos.Protocols, id, device string) (*UDP, error) {
	portMap, err := buildPortsMap(p.GetAllUDP())
	if err != nil {
		return nil, err
	}

	udp := &UDP{
		protocols: p,
		portMap:   portMap,
		metrics:   newInputMetrics(id, device, portMap),
	}
	logp.Debug("udp", "Port map: %v", portMap)

	return udp, nil
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
				return nil, fmt.Errorf("duplicate port (%d) exists in %s and %s protocols",
					port, oldProto, proto)
			}
			res[uint16(port)] = proto
		}
	}

	return res, nil
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
		udp.metrics.log(pkt)
	}
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

func (udp *UDP) Close() {
	if udp.metrics == nil {
		return
	}
	udp.metrics.close()
}

// inputMetrics handles the input's metric reporting.
type inputMetrics struct {
	unregister func()

	lastPacket time.Time

	device         *monitoring.String // name of the device being monitored
	packets        *monitoring.Uint   // number of packets processed
	bytes          *monitoring.Uint   // number of bytes processed
	arrivalPeriod  metrics.Sample     // histogram of the elapsed time between packet arrivals
	processingTime metrics.Sample     // histogram of the elapsed time between packet receipt and publication
}

// newInputMetrics returns an input metric for the UDP processor. If id or
// device is empty a nil inputMetric is returned.
func newInputMetrics(id, device string, ports map[uint16]protos.Protocol) *inputMetrics {
	if id == "" || device == "" {
		// An empty id signals to not record metrics,
		// while an empty device means we are reading
		// from a pcap file and no metrics are needed.
		return nil
	}
	devID := fmt.Sprintf("%s-udp%s::%s", id, portList(ports), device)
	reg, unreg := inputmon.NewInputRegistry("udp", devID, nil)
	out := &inputMetrics{
		unregister:     unreg,
		device:         monitoring.NewString(reg, "device"),
		packets:        monitoring.NewUint(reg, "received_events_total"),
		bytes:          monitoring.NewUint(reg, "received_bytes_total"),
		arrivalPeriod:  metrics.NewUniformSample(1024),
		processingTime: metrics.NewUniformSample(1024),
	}
	_ = adapter.NewGoMetrics(reg, "arrival_period", adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.arrivalPeriod))
	_ = adapter.NewGoMetrics(reg, "processing_time", adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.processingTime))

	out.device.Set(device)

	return out
}

// portList returns a dash-separated list of port numbers sorted ascending. A leading
// dash is prepended to the list if it is not empty.
func portList(m map[uint16]protos.Protocol) string {
	if len(m) == 0 {
		return ""
	}
	ports := make([]int, 0, len(m))
	for p := range m {
		ports = append(ports, int(p))
	}
	sort.Ints(ports)
	s := make([]string, len(ports)+1)
	for i, p := range ports {
		s[i+1] = strconv.FormatInt(int64(p), 10)
	}
	return strings.Join(s, "-")
}

// log logs metric for the given packet.
func (m *inputMetrics) log(pkt *protos.Packet) {
	if m == nil {
		return
	}
	m.processingTime.Update(time.Since(pkt.Ts).Nanoseconds())
	m.packets.Add(1)
	m.bytes.Add(uint64(len(pkt.Payload)))
	if !m.lastPacket.IsZero() {
		m.arrivalPeriod.Update(pkt.Ts.Sub(m.lastPacket).Nanoseconds())
	}
	m.lastPacket = pkt.Ts
}

func (m *inputMetrics) close() {
	if m == nil {
		return
	}
	m.unregister()
}
