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

package dhcpv4

import (
	"fmt"
	"strings"

	"github.com/insomniacslk/dhcp/dhcpv4"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/monitoring"
	"github.com/elastic/beats/packetbeat/protos"
)

var (
	metricTotalPackets  = monitoring.NewUint(nil, "dhcpv4.total_packets")
	metricParseFailures = monitoring.NewUint(nil, "dhcpv4.parse_failures")
)

func init() {
	protos.Register("dhcpv4", New)
}

// New constructs a new dhcpv4 protocol plugin.
func New(
	testMode bool,
	results protos.Reporter,
	cfg *common.Config,
) (protos.Plugin, error) {
	return newPlugin(testMode, results, cfg)
}

func newPlugin(testMode bool, results protos.Reporter, cfg *common.Config) (*dhcpv4Plugin, error) {
	config := defaultConfig

	if !testMode {
		if err := cfg.Unpack(&config); err != nil {
			return nil, err
		}
	}

	return &dhcpv4Plugin{
		dhcpv4Config: config,
		report:       results,
		log:          logp.NewLogger("dhcpv4"),
	}, nil
}

type dhcpv4Plugin struct {
	dhcpv4Config
	report protos.Reporter
	log    *logp.Logger
}

func (p *dhcpv4Plugin) GetPorts() []int {
	return p.dhcpv4Config.Ports
}

func (p *dhcpv4Plugin) ParseUDP(pkt *protos.Packet) {
	if event := p.parseDHCPv4(pkt); event != nil {
		p.report(*event)
	}
}

func (p *dhcpv4Plugin) parseDHCPv4(pkt *protos.Packet) *beat.Event {
	metricTotalPackets.Inc()

	v4, err := dhcpv4.FromBytes(pkt.Payload)
	if err != nil {
		metricParseFailures.Inc()
		p.log.Warnw("dropping packet: failed parsing DHCP data", "error", err)
		return nil
	}

	dhcpData := common.MapStr{
		"op_code":        strings.ToLower(v4.OpcodeToString()),
		"hardware_type":  v4.HwTypeToString(),
		"hops":           v4.HopCount(), // Set to non-zero by relays.
		"transaction_id": fmt.Sprintf("0x%08x", v4.TransactionID()),
		"seconds":        v4.NumSeconds(),
		"flags":          strings.ToLower(v4.FlagsToString()),
		"client_mac":     v4.ClientHwAddrToString(),
	}

	if !v4.ClientIPAddr().IsUnspecified() {
		dhcpData.Put("client_ip", v4.ClientIPAddr().String())
	}
	if !v4.YourIPAddr().IsUnspecified() {
		dhcpData.Put("assigned_ip", v4.YourIPAddr().String())
	}
	if !v4.GatewayIPAddr().IsUnspecified() {
		dhcpData.Put("relay_ip", v4.GatewayIPAddr().String())
	}
	if serverName := v4.ServerHostNameToString(); serverName != "" {
		dhcpData.Put("server_name", serverName)
	}
	if fileName := v4.BootFileNameToString(); fileName != "" {
		dhcpData.Put("boot_file_name", fileName)
	}

	if opts, err := optionsToMap(v4.StrippedOptions()); err != nil {
		p.log.Warnw("failed converting DHCP options to map",
			"dhcpv4", v4, "error", err)
	} else if len(opts) > 0 {
		dhcpData.Put("option", opts)
	}

	event := &beat.Event{
		Timestamp: pkt.Ts,
		Fields: common.MapStr{
			"transport": "udp",
			"type":      "dhcpv4",
			"status":    "OK",
			"dhcpv4":    dhcpData,
		},
	}

	if v4.Opcode() == dhcpv4.OpcodeBootReply {
		event.PutValue("ip", pkt.Tuple.SrcIP.String())
		event.PutValue("port", pkt.Tuple.SrcPort)
		event.PutValue("client_ip", pkt.Tuple.DstIP.String())
		event.PutValue("client_port", pkt.Tuple.DstPort)
		event.PutValue("bytes_out", len(pkt.Payload))
	} else {
		event.PutValue("ip", pkt.Tuple.DstIP.String())
		event.PutValue("port", pkt.Tuple.DstPort)
		event.PutValue("client_ip", pkt.Tuple.SrcIP.String())
		event.PutValue("client_port", pkt.Tuple.SrcPort)
		event.PutValue("bytes_in", len(pkt.Payload))
	}

	return event
}
