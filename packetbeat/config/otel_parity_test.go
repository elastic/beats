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

	ucfgyaml "github.com/elastic/go-ucfg/yaml"

	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

// TestFromStaticOTelParity proves that FromStatic correctly extracts per-stream
// interface and procs settings from the OTel receiver hybrid config format.
// elastic-agent nests these settings inside individual protocol entries rather
// than at the top level; without the fix they are silently dropped.
//
// The corpus fixtures under testdata/ are pre-rendered from the network_traffic
// integration templates using default variable values and represent what
// elastic-agent actually delivers.
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
			cfg, err := config.NewConfigFrom(map[string]any{
				"protocols": []any{streamMap},
			})
			if err != nil {
				t.Fatalf("building config for %s: %v", e.DataStream, err)
			}
			got, err := Config{}.FromStatic(cfg, logptest.NewTestingLogger(t, ""))
			if err != nil {
				t.Fatalf("FromStatic(%s): %v", e.DataStream, err)
			}
			if len(got.Interfaces) == 0 {
				t.Errorf("%s: Interfaces is empty; per-stream interface was dropped", e.DataStream)
			} else if got.Interfaces[0].Device == "" {
				t.Errorf("%s: Interfaces[0].Device is empty; device was not preserved", e.DataStream)
			}
			if e.DataStream == "network_traffic.flow" {
				if got.Flows == nil {
					t.Errorf("%s: Flows is nil; flow entry was not routed", e.DataStream)
				}
			}
		})
	}

	// This case has no corpus equivalent: all corpus entries include a per-stream
	// interface, but when none is present FromStatic must fall back to defaultDevice().
	t.Run("no_stream_interface_fallback", func(t *testing.T) {
		cfg, err := config.NewConfigFrom(`
protocols:
- type: http
  ports: [80]
`)
		if err != nil {
			t.Fatalf("parsing config: %v", err)
		}
		got, err := Config{}.FromStatic(cfg, logptest.NewTestingLogger(t, ""))
		if err != nil {
			t.Fatalf("FromStatic: %v", err)
		}
		if len(got.Interfaces) == 0 {
			t.Error("Interfaces is empty, want defaultDevice() fallback")
		} else if got.Interfaces[0].Device != defaultDevice() {
			t.Errorf("Interfaces[0].Device = %q, want %q", got.Interfaces[0].Device, defaultDevice())
		}
	})
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
