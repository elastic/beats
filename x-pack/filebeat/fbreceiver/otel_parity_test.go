// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fbreceiver

import (
	"embed"
	"fmt"
	"io/fs"
	"strings"
	"testing"

	"go.opentelemetry.io/collector/confmap"

	ucfgyaml "github.com/elastic/go-ucfg/yaml"
)

// TestOTelParityConfig proves that fbreceiver.Config accepts a config
// assembled from the corpus stream fixtures and that type and key input
// fields survive the round-trip. This documents the format that
// elastic-agent produces for the filebeat OTel receiver.
func TestOTelParityConfig(t *testing.T) {
	inputs := assembleInputs(t)
	cfg := unmarshalReceiverConfig(t, inputs)
	if err := cfg.Validate(); err != nil {
		t.Errorf("Config.Validate: %v", err)
	}

	fbMap, ok := cfg.Beatconfig["filebeat"].(map[string]any)
	if !ok {
		t.Fatalf("Beatconfig[\"filebeat\"] is not map[string]any")
	}
	rawInputs, ok := fbMap["inputs"].([]any)
	if !ok {
		t.Fatalf("filebeat.inputs is not []any")
	}

	byType := map[string]map[string]any{}
	for _, raw := range rawInputs {
		inp, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if typ, _ := inp["type"].(string); typ != "" {
			byType[typ] = inp
		}
	}

	httpjson, ok := byType["httpjson"]
	if !ok {
		t.Error("httpjson input not found in assembled config")
	} else {
		// request.* dotted keys in the template are expanded by ucfg into
		// a nested map before reaching the receiver.
		if _, ok := httpjson["request"]; !ok {
			t.Error("httpjson: request block not found")
		}
		if _, ok := httpjson["cursor"]; !ok {
			t.Error("httpjson: cursor not found")
		}
	}

	cel, ok := byType["cel"]
	if !ok {
		t.Error("cel input not found in assembled config")
	} else {
		if _, ok := cel["program"]; !ok {
			t.Error("cel: program not found")
		}
		if _, ok := cel["state"]; !ok {
			t.Error("cel: state not found")
		}
	}

	logfile, ok := byType["auditd-logfile"]
	if !ok {
		t.Error("auditd-logfile input not found in assembled config")
	} else {
		if _, ok := logfile["paths"]; !ok {
			t.Error("auditd-logfile: paths not found")
		}
	}
}

func assembleInputs(t *testing.T) []any {
	t.Helper()
	entries, err := loadCorpus()
	if err != nil {
		t.Fatalf("loading corpus: %v", err)
	}
	inputs := make([]any, 0, len(entries))
	for _, e := range entries {
		ucfg, err := ucfgyaml.NewConfig(e.yaml)
		if err != nil {
			t.Fatalf("parsing %s: %v", e.dataStream, err)
		}
		var m map[string]any
		if err := ucfg.Unpack(&m); err != nil {
			t.Fatalf("unpacking %s: %v", e.dataStream, err)
		}
		inputs = append(inputs, m)
	}
	return inputs
}

func unmarshalReceiverConfig(t *testing.T, inputs []any) *Config {
	t.Helper()
	cfg := &Config{}
	conf := confmap.NewFromStringMap(map[string]any{
		"filebeat": map[string]any{"inputs": inputs},
	})
	if err := cfg.Unmarshal(conf); err != nil {
		t.Fatalf("Config.Unmarshal: %v", err)
	}
	return cfg
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
			dataStream: strings.TrimSuffix(e.Name(), ".yaml"),
			yaml:       data,
		})
	}
	return out, nil
}

type corpusEntry struct {
	dataStream string
	yaml       []byte
}

//go:embed testdata/*.yaml
var testdata embed.FS
