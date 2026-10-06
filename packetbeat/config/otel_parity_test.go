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
	"embed"
	"fmt"
	"io/fs"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	ucfgyaml "github.com/elastic/go-ucfg/yaml"

	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

// TestFromStaticOTelParity checks that FromStatic, which is used when
// packetbeat runs as an OTel receiver, extracts the same per-stream interface
// and procs settings as NewAgentConfig, which is used by the process runtime.
// elastic-agent nests these settings inside individual protocol entries
// rather than at the top level.
//
// The corpus fixtures under testdata/ are pre-rendered from the network_traffic
// integration templates using default variable values and represent what
// elastic-agent delivers. The Source comment at the top of each fixture names
// the template to re-render it from.
func TestFromStaticOTelParity(t *testing.T) {
	entries, err := loadCorpus()
	if err != nil {
		t.Fatalf("loading corpus: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("corpus is empty")
	}

	for _, e := range entries {
		t.Run(e.DataStream, func(t *testing.T) {
			// Parse the stream fixture, then build an outer config that wraps it
			// in a protocols list — the hybrid format elastic-agent delivers to
			// the packetbeat OTel receiver.
			stream, err := ucfgyaml.NewConfig(e.YAML)
			if err != nil {
				t.Fatalf("parsing corpus entry %s: %v", e.DataStream, err)
			}
			var streamMap map[string]any
			if err := stream.Unpack(&streamMap); err != nil {
				t.Fatalf("unpacking corpus entry %s: %v", e.DataStream, err)
			}

			// The process runtime receives the stream under streams.
			agentCfg, err := config.NewConfigFrom(map[string]any{
				"streams": []any{streamMap},
			})
			if err != nil {
				t.Fatalf("building agent config for %s: %v", e.DataStream, err)
			}
			want, err := NewAgentConfig(agentCfg, logptest.NewTestingLogger(t, ""))
			if err != nil {
				t.Fatalf("NewAgentConfig(%s): %v", e.DataStream, err)
			}

			// The OTel receiver receives it under protocols.
			otelCfg, err := config.NewConfigFrom(map[string]any{
				"protocols": []any{streamMap},
			})
			if err != nil {
				t.Fatalf("building OTel config for %s: %v", e.DataStream, err)
			}
			got, err := seededConfig().FromStatic(otelCfg, logptest.NewTestingLogger(t, ""))
			if err != nil {
				t.Fatalf("FromStatic(%s): %v", e.DataStream, err)
			}

			assert.Equal(t, want.Interfaces, got.Interfaces, "interfaces for %s", e.DataStream)
			assert.Equal(t, want.Procs, got.Procs, "procs for %s", e.DataStream)
			assert.Equal(t, want.Flows != nil, got.Flows != nil, "flows routing for %s", e.DataStream)
		})
	}
}

// TestFromStaticStreamInterfaces checks the handling of per-stream interfaces
// when FromStatic starts from the placeholder interface that packetbeat seeds
// its configuration with.
func TestFromStaticStreamInterfaces(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		// zero starts from a zero Config instead of a seeded one.
		zero bool
		want []InterfaceConfig
		err  string
	}{
		{
			name: "any_device",
			yaml: `
protocols:
- type: http
  ports: [80]
  interface:
    device: any
    bpf_filter: port 80
    internal_networks: [private]
`,
			want: []InterfaceConfig{{Device: "any", BpfFilter: "port 80", InternalNetworks: []string{"private"}}},
		},
		{
			name: "named_device",
			yaml: `
protocols:
- type: http
  ports: [80]
  interface:
    device: eth0
    snaplen: 1500
`,
			want: []InterfaceConfig{{Device: "eth0", Snaplen: 1500}},
		},
		{
			name: "top_level_interfaces_take_precedence",
			yaml: `
interfaces:
- device: lo
protocols:
- type: http
  ports: [80]
  interface:
    device: eth0
`,
			want: []InterfaceConfig{{Device: "lo", Loop: 1}},
		},
		{
			name: "no_stream_interface_keeps_placeholder",
			yaml: `
protocols:
- type: http
  ports: [80]
`,
			want: []InterfaceConfig{{Loop: 1}},
		},
		{
			name: "zero_config_with_stream_interface",
			yaml: `
protocols:
- type: http
  ports: [80]
  interface:
    device: eth0
`,
			zero: true,
			want: []InterfaceConfig{{Device: "eth0"}},
		},
		{
			name: "zero_config_without_interface_has_none",
			yaml: `
protocols:
- type: http
  ports: [80]
`,
			zero: true,
			want: nil,
		},
		{
			name: "duplicate_devices",
			yaml: `
protocols:
- type: http
  ports: [80]
  interface:
    device: eth0
- type: dns
  ports: [53]
  interface:
    device: eth0
`,
			err: "duplicated device configurations: eth0",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg, err := config.NewConfigFrom(test.yaml)
			if err != nil {
				t.Fatalf("parsing config: %v", err)
			}
			start := seededConfig()
			if test.zero {
				start = Config{}
			}
			got, err := start.FromStatic(cfg, logptest.NewTestingLogger(t, ""))
			if test.err != "" {
				assert.ErrorContains(t, err, test.err, "FromStatic error")
				return
			}
			if err != nil {
				t.Fatalf("FromStatic: %v", err)
			}
			assert.Equal(t, test.want, got.Interfaces, "interfaces")
			assert.Nil(t, got.Interface, "singular interface should be cleared")
		})
	}
}

// seededConfig returns a Config shaped like the initial configuration that
// packetbeat passes to FromStatic: a single interface with no device, holding
// the command line settings.
func seededConfig() Config {
	c := Config{Interfaces: []InterfaceConfig{{Loop: 1}}}
	c.Interface = &c.Interfaces[0]
	return c
}

func loadCorpus() ([]corpusEntry, error) {
	dirEntries, err := fs.ReadDir(testdata, "testdata")
	if err != nil {
		return nil, fmt.Errorf("reading corpus: %w", err)
	}
	var out []corpusEntry
	for _, e := range dirEntries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		data, err := fs.ReadFile(testdata, "testdata/"+e.Name())
		if err != nil {
			return nil, fmt.Errorf("reading corpus entry %s: %w", e.Name(), err)
		}
		out = append(out, corpusEntry{
			DataStream: strings.TrimSuffix(e.Name(), ".yaml"),
			YAML:       data,
		})
	}
	return out, nil
}

type corpusEntry struct {
	DataStream string
	YAML       []byte
}

//go:embed testdata/*.yaml
var testdata embed.FS
