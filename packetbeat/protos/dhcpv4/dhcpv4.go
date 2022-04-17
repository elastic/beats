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

	"github.com/menderesk/beats/v7/libbeat/beat"
	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/ecs"
	"github.com/menderesk/beats/v7/libbeat/logp"
	"github.com/menderesk/beats/v7/libbeat/monitoring"
	"github.com/menderesk/beats/v7/packetbeat/pb"
	"github.com/menderesk/beats/v7/packetbeat/procs"
	"github.com/menderesk/beats/v7/packetbeat/protos"
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
	watcher procs.ProcessesWatcher,
	cfg *common.Config,
) (protos.Plugin, error) {
	return newPlugin(testMode, results, watcher, cfg)
}

func newPlugin(testMode bool, results protos.Reporter, watcher procs.ProcessesWatcher, cfg *common.Config) (*dhcpv4Plugin, error) {
	config := defaultConfig

	if !testMode {
		if err := cfg.Unpack(&config); err != nil {
			return nil, err
		}
	}

	return &dhcpv4Plugin{
		dhcpv4Config: config,
		report:       results,
		watcher:      watcher,
		log:          logp.NewLogger("dhcpv4"),
	}, nil
}

type dhcpv4Plugin struct {
	dhcpv4Config
	report  protos.Reporter
	watcher procs.ProcessesWatcher
	log     *logp.Logger
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
		p.log.Warnw("Dropping packet: failed parsing DHCP data", "error", err)
		return nil
	}

	evt, pbf := pb.NewBeatEvent(pkt.Ts)

	// source/destination (note: this protocol does not produce a bi-flow.)
	src, dst := common.MakeEndpointPair(pkt.Tuple.BaseTuple, nil)
	pbf.SetSource(&src)
	pbf.SetDestination(&dst)
	pbf.Source.Bytes = int64(len(pkt.Payload))

	if v4.Opcode() == dhcpv4.OpcodeBootReply {
		// Reverse
		client, server := ecs.Client(*pbf.Destination), ecs.Server(*pbf.Source)
		pbf.Client = &client
		pbf.Server = &server
	} else {
		client, server := ecs.Client(*pbf.Source), ecs.Server(*pbf.Destination)
		pbf.Client = &client
		pbf.Server = &server
	}

	pbf.Event.Start = pkt.Ts
	pbf.Event.Dataset = "dhcpv4"
	pbf.Network.Transport = "udp"
	pbf.Network.Protocol = pbf.Event.Dataset

	fields := evt.Fields
	fields["type"] = pbf.Event.Dataset
	fields["status"] = "OK"

	dhcpData := common.MapStr{
		"op_code":        strings.ToLower(v4.OpcodeToString()),
		"hardware_type":  v4.HwTypeToString(),
		"hops":           v4.HopCount(), // Set to non-zero by relays.
		"transaction_id": fmt.Sprintf("0x%08x", v4.TransactionID()),
		"seconds":        v4.NumSeconds(),
		"flags":          strings.ToLower(v4.FlagsToString()),
		"client_mac":     v4.ClientHwAddrToString(),
	}
	fields["dhcpv4"] = dhcpData

	if !v4.ClientIPAddr().IsUnspecified() {
		dhcpData.Put("client_ip", v4.ClientIPAddr().String())
		pbf.AddIP(v4.ClientIPAddr().String())
	}
	if !v4.YourIPAddr().IsUnspecified() {
		dhcpData.Put("assigned_ip", v4.YourIPAddr().String())
		pbf.AddIP(v4.YourIPAddr().String())
	}
	if !v4.GatewayIPAddr().IsUnspecified() {
		dhcpData.Put("relay_ip", v4.GatewayIPAddr().String())
		pbf.AddIP(v4.GatewayIPAddr().String())
	}
	if serverName := v4.ServerHostNameToString(); serverName != "" {
		dhcpData.Put("server_name", serverName)
	}
	if fileName := v4.BootFileNameToString(); fileName != "" {
		dhcpData.Put("boot_file_name", fileName)
	}

	if opts, err := optionsToMap(v4.StrippedOptions()); err != nil {
		p.log.Warnw("Failed converting DHCP options to map",
			"dhcpv4", v4, "error", err)
	} else if len(opts) > 0 {
		dhcpData.Put("option", opts)
	}

	return &evt
}
