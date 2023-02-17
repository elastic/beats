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

package sniffer

import (
	"github.com/google/gopacket/layers"

	"github.com/elastic/beats/v7/packetbeat/config"
	"github.com/elastic/beats/v7/packetbeat/decoder"
	"github.com/elastic/beats/v7/packetbeat/flows"
	"github.com/elastic/beats/v7/packetbeat/procs"
	"github.com/elastic/beats/v7/packetbeat/protos"
	"github.com/elastic/beats/v7/packetbeat/protos/icmp"
	"github.com/elastic/beats/v7/packetbeat/protos/tcp"
	"github.com/elastic/beats/v7/packetbeat/protos/udp"
	"github.com/elastic/beats/v7/packetbeat/publish"
)

// Decoders functions return a Decoder able to process the provided network
// link type for use with a Sniffer. The cleanup closure should be called after
// the decoders are no longer needed to clean up resources.
type Decoders func(_ layers.LinkType, device string) (decoders *decoder.Decoder, cleanup func(), err error)

// DecodersFor returns a source of Decoders using the provided configuration
// components. The id string is expected to be the ID of the beat.
func DecodersFor(id string, publisher *publish.TransactionPublisher, protocols *protos.ProtocolsStruct, watcher *procs.ProcessesWatcher, flows *flows.Flows, cfg config.Config) Decoders {
	return func(dl layers.LinkType, device string) (*decoder.Decoder, func(), error) {
		var icmp4 icmp.ICMPv4Processor
		var icmp6 icmp.ICMPv6Processor
		icmpCfg, err := cfg.ICMP()
		if err != nil {
			return nil, nil, err
		}
		if icmpCfg.Enabled() {
			reporter, err := publisher.CreateReporter(icmpCfg)
			if err != nil {
				return nil, nil, err
			}

			icmp, err := icmp.New(false, reporter, watcher, icmpCfg)
			if err != nil {
				return nil, nil, err
			}

			icmp4 = icmp
			icmp6 = icmp
		}

		tcp, err := tcp.NewTCP(protocols, id, device)
		if err != nil {
			return nil, nil, err
		}

		udp, err := udp.NewUDP(protocols, id, device)
		if err != nil {
			return nil, nil, err
		}

		worker, err := decoder.New(flows, dl, icmp4, icmp6, tcp, udp)
		if err != nil {
			return nil, nil, err
		}

		cleanup := func() {
			// Close metric collection.
			tcp.Close()
			udp.Close()
		}

		return worker, cleanup, nil
	}
}
