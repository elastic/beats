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

package beater

import (
	"testing"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/management"
	"github.com/elastic/elastic-agent-libs/config"
)

var canInstallNpcapTests = []struct {
	name    string
	cfg     string
	managed bool
	want    any
}{
	{
		name: `packetbeat_never_install`,
		cfg: `
interfaces.device: default_route
interfaces.poll_default_route: 1m
interfaces.internal_networks:
  - private

npcap:
  never_install: true

protocols:
- type: icmp
  enabled: true
`,
		managed: false,
		want:    false,
	},
	{
		name: `packetbeat_can_install`,
		cfg: `
interfaces.device: default_route
interfaces.poll_default_route: 1m
interfaces.internal_networks:
  - private

protocols:
- type: icmp
  enabled: true
`,
		managed: false,
		want:    true,
	},
	{
		name: `fleet_never_install_single`,
		cfg: `
type: packet
data_stream:
  namespace: default
processors:
  - add_fields:
      target: 'elastic_agent'
      fields:
        id: agent-id
        version: 8.0.0
        snapshot: false
streams:
  - type: icmp
    interface:
      device: default_route
    procs:
      enabled: true
    data_stream:
      dataset: packet.icmp
      type: logs
    npcap:
      never_install: true
`,
		managed: true,
		want:    false,
	},
	{
		name: `fleet_can_install_single`,
		cfg: `
type: packet
data_stream:
  namespace: default
processors:
  - add_fields:
      target: 'elastic_agent'
      fields:
        id: agent-id
        version: 8.0.0
        snapshot: false
streams:
  - type: icmp
    interface:
      device: default_route
    procs:
      enabled: true
    data_stream:
      dataset: packet.icmp
      type: logs
`,
		managed: true,
		want:    true,
	},
	{
		name: `fleet_never_install_multi`,
		cfg: `
type: packet
data_stream:
  namespace: default
processors:
  - add_fields:
      target: 'elastic_agent'
      fields:
        id: agent-id
        version: 8.0.0
        snapshot: false
streams:
  - type: http
    interface:
      device: en2
      snaplen: 1514
      type: af_packet
      buffer_size_mb: 100
    procs:
      enabled: true
      monitored:
        - process: curl
          cmdline_grep: curl
    data_stream:
      dataset: packet.http
      type: logs
  - type: icmp
    interface:
      device: default_route
    procs:
      enabled: true
    data_stream:
      dataset: packet.icmp
      type: logs
    npcap:
      never_install: true
`,
		managed: true,
		want:    false,
	},
	{
		name: `fleet_can_install_multi`,
		cfg: `
type: packet
data_stream:
  namespace: default
processors:
  - add_fields:
      target: 'elastic_agent'
      fields:
        id: agent-id
        version: 8.0.0
        snapshot: false
streams:
  - type: http
    interface:
      device: en2
      snaplen: 1514
      type: af_packet
      buffer_size_mb: 100
    procs:
      enabled: true
      monitored:
        - process: curl
          cmdline_grep: curl
    data_stream:
      dataset: packet.http
      type: logs
  - type: icmp
    interface:
      device: default_route
    procs:
      enabled: true
    data_stream:
      dataset: packet.icmp
      type: logs
`,
		managed: true,
		want:    true,
	},
}

func TestCanInstallNpcap(t *testing.T) {
	for _, test := range canInstallNpcapTests {
		t.Run(test.name, func(t *testing.T) {
			cfg, err := config.NewConfigFrom(test.cfg)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			b := &beat.Beat{
				BeatConfig: cfg,
				Manager:    boolManager{managed: test.managed},
			}
			got, err := canInstallNpcap(b)
			if err != nil {
				t.Errorf("unexpected error from canInstallNpcap: %v", err)
			}
			if got != test.want {
				t.Errorf("unexpected result from canInstallNpcap: got=%t want=%t", got, test.want)
			}
		})
	}
}

type boolManager struct {
	managed bool

	// For interface satisfaction.
	management.Manager
}

func (m boolManager) Enabled() bool { return m.managed }
