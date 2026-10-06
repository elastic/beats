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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	ucfgyaml "github.com/elastic/go-ucfg/yaml"

	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

// TestFromStaticOTelParity checks that FromStatic, which is used when
// packetbeat runs as an OTel receiver, reads the interface and procs keys of a
// protocols entry in the same way that NewAgentConfig reads them from a
// streams entry.
//
// The fixtures under testdata/ are written by hand, modelled on the output of
// the elastic/integrations templates named in their Source comments. They are
// not generated, and their values are not the template defaults. Fleet adds
// the data_stream key when it builds the agent policy. elastic-agent adds an
// index key and an add_agent_metadata processor for a receiver. The fixtures
// include data_stream but have neither index nor add_agent_metadata.
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
			// Parse the stream fixture, then wrap it in a protocols list, as
			// elastic-agent does when it delivers a stream to the packetbeat
			// OTel receiver. The interface and procs keys stay inside the entry.
			stream, err := ucfgyaml.NewConfig(e.YAML)
			if err != nil {
				t.Fatalf("parsing corpus entry %s: %v", e.DataStream, err)
			}
			var streamMap map[string]any
			if err := stream.Unpack(&streamMap); err != nil {
				t.Fatalf("unpacking corpus entry %s: %v", e.DataStream, err)
			}

			// NewAgentConfig reads the stream from a streams list.
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

			// FromStatic reads the same stream from a protocols list.
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

// TestFromStaticStreamInterfaces checks how FromStatic combines the interface
// keys of protocols entries with the interfaces it starts with. Unless a case
// sets zero, that is the placeholder interface from seededConfig.
func TestFromStaticStreamInterfaces(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		zero bool // zero starts from a zero Config instead of seededConfig.
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

// seededConfig returns a Config like the one initialConfig in the beater
// package passes to FromStatic: one interface with no device, with Loop set to
// the default of the -l flag, and Interface pointing at it.
func seededConfig() Config {
	c := Config{Interfaces: []InterfaceConfig{{Loop: 1}}}
	c.Interface = &c.Interfaces[0]
	return c
}

func loadCorpus() ([]corpusEntry, error) {
	dirEntries, err := os.ReadDir("testdata")
	if err != nil {
		return nil, fmt.Errorf("reading corpus: %w", err)
	}
	var out []corpusEntry
	for _, e := range dirEntries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		data, err := os.ReadFile(filepath.Join("testdata", e.Name()))
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
