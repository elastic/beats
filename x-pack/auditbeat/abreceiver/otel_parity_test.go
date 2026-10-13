// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package abreceiver

import (
	"embed"
	"fmt"
	"io/fs"
	"strings"
	"testing"

	"go.opentelemetry.io/collector/confmap"

	ucfgyaml "github.com/elastic/go-ucfg/yaml"
)

// TestOTelParityConfig proves that abreceiver.Config accepts a config
// assembled from the corpus stream fixtures and that key module fields
// survive the round-trip. This documents the format that elastic-agent
// must produce for the auditbeat OTel receiver once translation support
// is added. Currently elastic-agent has no auditbeat translation case;
// see getDefaultDatastreamTypeForComponent in otelconfig.go.
func TestOTelParityConfig(t *testing.T) {
	modules := assembleModules(t)
	cfg := unmarshalReceiverConfig(t, modules)
	if err := cfg.Validate(); err != nil {
		t.Errorf("Config.Validate: %v", err)
	}

	abMap, ok := cfg.Beatconfig["auditbeat"].(map[string]any)
	if !ok {
		t.Fatalf("Beatconfig[\"auditbeat\"] is not map[string]any")
	}
	rawModules, ok := abMap["modules"].([]any)
	if !ok {
		t.Fatalf("auditbeat.modules is not []any")
	}

	byName := map[string]map[string]any{}
	for _, m := range rawModules {
		mod, ok := m.(map[string]any)
		if !ok {
			continue
		}
		if name, _ := mod["module"].(string); name != "" {
			byName[name] = mod
		}
	}

	auditd, ok := byName["auditd"]
	if !ok {
		t.Error("auditd module not found in assembled config")
	} else {
		if _, ok := auditd["backlog_limit"]; !ok {
			t.Error("auditd: backlog_limit not found")
		}
		if _, ok := auditd["failure_mode"]; !ok {
			t.Error("auditd: failure_mode not found")
		}
	}

	fi, ok := byName["file_integrity"]
	if !ok {
		t.Error("file_integrity module not found in assembled config")
	} else {
		if _, ok := fi["paths"]; !ok {
			t.Error("file_integrity: paths not found")
		}
	}

	sys, ok := byName["system"]
	if !ok {
		t.Error("system module not found in assembled config")
	} else {
		if _, ok := sys["datasets"]; !ok {
			t.Error("system: datasets not found")
		}
	}
}

func assembleModules(t *testing.T) []any {
	t.Helper()
	entries, err := loadCorpus()
	if err != nil {
		t.Fatalf("loading corpus: %v", err)
	}
	modules := make([]any, 0, len(entries))
	for _, e := range entries {
		ucfg, err := ucfgyaml.NewConfig(e.yaml)
		if err != nil {
			t.Fatalf("parsing %s: %v", e.dataStream, err)
		}
		var m map[string]any
		if err := ucfg.Unpack(&m); err != nil {
			t.Fatalf("unpacking %s: %v", e.dataStream, err)
		}
		modules = append(modules, m)
	}
	return modules
}

func unmarshalReceiverConfig(t *testing.T, modules []any) *Config {
	t.Helper()
	cfg := &Config{}
	conf := confmap.NewFromStringMap(map[string]any{
		"auditbeat": map[string]any{"modules": modules},
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
