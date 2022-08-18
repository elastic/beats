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

package config

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/elastic/elastic-agent-libs/config"
)

var fromStaticTests = []struct {
	name   string
	cli    Config
	config string
	want   Config
}{
	{
		name: "single_interface",
		config: `
interfaces.device: default_route

interfaces.dumpfile: dwnp

interfaces.poll_default_route: 1m

interfaces.internal_networks:
  - private

protocols:
- type: icmp
  enabled: true

- type: amqp
  ports: [5672]
`,
		want: Config{
			Interfaces: []InterfacesConfig{{
				Device:           "default_route",
				PollDefaultRoute: time.Minute,
				InternalNetworks: []string{"private"},
				Dumpfile:         "dwnp",
			}},
			Protocols: map[string]*config.C{},
			ProtocolsList: []*config.C{
				config.MustNewConfigFrom(map[string]interface{}{
					"enabled": true,
					"type":    "icmp",
				}),
				config.MustNewConfigFrom(map[string]interface{}{
					"type": "amqp",
					"ports": []int{
						5672,
					},
				}),
			},
		},
	},
	{
		name: "single_interface_cli",
		cli:  cliOptions("input.pcap", 0, true, true, "dump"),
		config: `
interfaces.device: default_route

interfaces.poll_default_route: 1m

interfaces.internal_networks:
  - private

protocols:
- type: icmp
  enabled: true

- type: amqp
  ports: [5672]
`,
		want: Config{
			Interfaces: []InterfacesConfig{{
				Device:           "default_route",
				File:             "input.pcap",
				TopSpeed:         true,
				OneAtATime:       true,
				PollDefaultRoute: time.Minute,
				InternalNetworks: []string{"private"},
				Dumpfile:         "dump",
			}},
			Protocols: map[string]*config.C{},
			ProtocolsList: []*config.C{
				config.MustNewConfigFrom(map[string]interface{}{
					"enabled": true,
					"type":    "icmp",
				}),
				config.MustNewConfigFrom(map[string]interface{}{
					"type": "amqp",
					"ports": []int{
						5672,
					},
				}),
			},
		},
	},
	{
		name: "empty_cli",
		cli:  cliOptions("input.pcap", 0, true, true, "dump"),
		config: `
protocols:
- type: icmp
  enabled: true

- type: amqp
  ports: [5672]
`,
		want: Config{
			Interfaces: []InterfacesConfig{{
				File:       "input.pcap",
				TopSpeed:   true,
				OneAtATime: true,
				Dumpfile:   "dump",
			}},
			Protocols: map[string]*config.C{},
			ProtocolsList: []*config.C{
				config.MustNewConfigFrom(map[string]interface{}{
					"enabled": true,
					"type":    "icmp",
				}),
				config.MustNewConfigFrom(map[string]interface{}{
					"type": "amqp",
					"ports": []int{
						5672,
					},
				}),
			},
		},
	},
	{
		name: "single_interface_array",
		config: `
interfaces:
- device: default_route
  dumpfile: dwnp
  poll_default_route: 1m
  internal_networks:
  - private

protocols:
- type: icmp
  enabled: true

- type: amqp
  ports: [5672]
`,
		want: Config{
			Interfaces: []InterfacesConfig{{
				Device:           "default_route",
				PollDefaultRoute: time.Minute,
				InternalNetworks: []string{"private"},
				Dumpfile:         "dwnp",
			}},
			Protocols: map[string]*config.C{},
			ProtocolsList: []*config.C{
				config.MustNewConfigFrom(map[string]interface{}{
					"enabled": true,
					"type":    "icmp",
				}),
				config.MustNewConfigFrom(map[string]interface{}{
					"type": "amqp",
					"ports": []int{
						5672,
					},
				}),
			},
		},
	},
	{
		name: "single_interface_array_cli",
		cli:  cliOptions("input.pcap", 0, true, true, "dump"),
		config: `
interfaces:
- device: default_route
  poll_default_route: 1m
  internal_networks:
  - private

protocols:
- type: icmp
  enabled: true

- type: amqp
  ports: [5672]
`,
		want: Config{
			Interfaces: []InterfacesConfig{{
				Device:           "default_route",
				File:             "input.pcap",
				TopSpeed:         true,
				OneAtATime:       true,
				PollDefaultRoute: time.Minute,
				InternalNetworks: []string{"private"},
				Dumpfile:         "dump",
			}},
			Protocols: map[string]*config.C{},
			ProtocolsList: []*config.C{
				config.MustNewConfigFrom(map[string]interface{}{
					"enabled": true,
					"type":    "icmp",
				}),
				config.MustNewConfigFrom(map[string]interface{}{
					"type": "amqp",
					"ports": []int{
						5672,
					},
				}),
			},
		},
	},
	{
		name: "multiple_interface",
		config: `
interfaces:
- device: en0
  bpf_filter: foo
  internal_networks:
  - private

- device: en1
  bpf_filter: bar
  internal_networks:
  - private

protocols:
- type: icmp
  enabled: true

- type: amqp
  ports: [5672]
`,
		want: Config{
			Interfaces: []InterfacesConfig{
				{
					Device:           "en0",
					BpfFilter:        "foo",
					InternalNetworks: []string{"private"},
				},
				{
					Device:           "en1",
					BpfFilter:        "bar",
					InternalNetworks: []string{"private"},
				},
			},
			Protocols: map[string]*config.C{},
			ProtocolsList: []*config.C{
				config.MustNewConfigFrom(map[string]interface{}{
					"enabled": true,
					"type":    "icmp",
				}),
				config.MustNewConfigFrom(map[string]interface{}{
					"type": "amqp",
					"ports": []int{
						5672,
					},
				}),
			},
		},
	},
	{
		name: "multiple_interface_cli",
		cli:  cliOptions("input.pcap", 0, true, true, "dump"),
		config: `
interfaces:
- device: en0
  bpf_filter: foo
  internal_networks:
  - private

- device: en1
  bpf_filter: bar
  internal_networks:
  - private

protocols:
- type: icmp
  enabled: true

- type: amqp
  ports: [5672]
`,
		want: Config{
			Interfaces: []InterfacesConfig{
				{
					Device:           "en0",
					File:             "input.pcap",
					BpfFilter:        "foo",
					TopSpeed:         true,
					OneAtATime:       true,
					InternalNetworks: []string{"private"},
					Dumpfile:         "dump",
				},
				{
					Device:           "en1",
					BpfFilter:        "bar",
					InternalNetworks: []string{"private"},
				},
			},
			Protocols: map[string]*config.C{},
			ProtocolsList: []*config.C{
				config.MustNewConfigFrom(map[string]interface{}{
					"enabled": true,
					"type":    "icmp",
				}),
				config.MustNewConfigFrom(map[string]interface{}{
					"type": "amqp",
					"ports": []int{
						5672,
					},
				}),
			},
		},
	},
	{
		name: "single_interface_cli_clobber",
		cli:  cliOptions("input.pcap", 0, true, true, "dump"),
		config: `
interfaces.device: default_route
interfaces.dumpfile: dwnp

interfaces.poll_default_route: 1m

interfaces.internal_networks:
  - private

protocols:
- type: icmp
  enabled: true

- type: amqp
  ports: [5672]
`,
		want: Config{
			Interfaces: []InterfacesConfig{{
				Device:           "default_route",
				File:             "input.pcap",
				TopSpeed:         true,
				OneAtATime:       true,
				PollDefaultRoute: time.Minute,
				InternalNetworks: []string{"private"},
				Dumpfile:         "dwnp",
			}},
			Protocols: map[string]*config.C{},
			ProtocolsList: []*config.C{
				config.MustNewConfigFrom(map[string]interface{}{
					"enabled": true,
					"type":    "icmp",
				}),
				config.MustNewConfigFrom(map[string]interface{}{
					"type": "amqp",
					"ports": []int{
						5672,
					},
				}),
			},
		},
	},
	{
		name: "single_interface_array_cli_clobber",
		cli:  cliOptions("input.pcap", 0, true, true, "dump"),
		config: `
interfaces:
- device: default_route
  dumpfile: dwnp
  poll_default_route: 1m
  internal_networks:
  - private

protocols:
- type: icmp
  enabled: true

- type: amqp
  ports: [5672]
`,
		want: Config{
			Interfaces: []InterfacesConfig{{
				Device:           "default_route",
				File:             "input.pcap",
				TopSpeed:         true,
				OneAtATime:       true,
				PollDefaultRoute: time.Minute,
				InternalNetworks: []string{"private"},
				Dumpfile:         "dwnp",
			}},
			Protocols: map[string]*config.C{},
			ProtocolsList: []*config.C{
				config.MustNewConfigFrom(map[string]interface{}{
					"enabled": true,
					"type":    "icmp",
				}),
				config.MustNewConfigFrom(map[string]interface{}{
					"type": "amqp",
					"ports": []int{
						5672,
					},
				}),
			},
		},
	},
}

func TestFromStatic(t *testing.T) {
	for _, test := range fromStaticTests {
		t.Run(test.name, func(t *testing.T) {
			cfg, err := config.NewConfigFrom(test.config)
			if err != nil {
				t.Fatalf("failed to construct config.C: %v", err)
			}
			got := test.cli
			got, err = got.FromStatic(cfg)
			if err != nil {
				t.Fatalf("failed to construct config.C: %v", err)
			}
			if !cmp.Equal(got, test.want, cmp.Comparer(configC)) {
				t.Errorf("unexpected result: got:- want:+\n%s", cmp.Diff(got, test.want, cmp.Comparer(configC)))
			}
		})
	}
}

// keep this in sync with packetbeat/beater.initialConfig()
func cliOptions(file string, loop int, topSpeed, step bool, dump string) Config {
	c := Config{
		Interfaces: []InterfacesConfig{{
			File:       file,
			Loop:       loop,
			TopSpeed:   topSpeed,
			OneAtATime: step,
			Dumpfile:   dump,
		}},
	}
	c.Interface = &c.Interfaces[0]
	return c
}

func configC(a, b *config.C) bool {
	var ma, mb map[string]interface{}
	err := a.Unpack(&ma)
	if err != nil {
		panic(err)
	}
	err = b.Unpack(&mb)
	if err != nil {
		panic(err)
	}
	return cmp.Equal(ma, mb)
}
