// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	ucfgyaml "github.com/elastic/go-ucfg/yaml"
)

// TestOTelParityConfig checks that osquerybeat's InputConfig parser consumes
// each fixture stream, including the osquery key that elastic-agent puts on
// the result stream. This documents what the OTel translation in
// elastic-agent must produce.
func TestOTelParityConfig(t *testing.T) {
	inputs := parseInputs(t)

	result := ResultInput(inputs)
	if result.Datastream.Dataset != DefaultDataset {
		t.Fatalf("ResultInput dataset = %q; want %q", result.Datastream.Dataset, DefaultDataset)
	}
	if result.Osquery == nil {
		t.Fatal("result stream: Osquery is nil; osquery key was dropped")
	}
	if len(result.Osquery.Schedule) == 0 {
		t.Error("result stream: Osquery.Schedule is empty; no queries present")
	}

	datasets := make(map[string]bool)
	for _, in := range inputs {
		datasets[in.Datastream.Dataset] = true
	}
	for _, want := range []string{DefaultDataset, DefaultActionResponsesDataset, DefaultQueryProfileDataset} {
		if !datasets[want] {
			t.Errorf("stream with dataset %q not found in fixtures", want)
		}
	}
}

// TestOTelParityOrdering checks that ResultInput finds the result stream, and
// so the osquery key, wherever it is in the delivery order.
func TestOTelParityOrdering(t *testing.T) {
	inputs := parseInputs(t)

	for _, order := range []string{"as read", "reversed"} {
		if order == "reversed" {
			slices.Reverse(inputs)
		}
		result := ResultInput(inputs)
		if result.Datastream.Dataset != DefaultDataset {
			t.Errorf("%s: ResultInput dataset = %q; want %q", order, result.Datastream.Dataset, DefaultDataset)
			continue
		}
		if result.Osquery == nil || len(result.Osquery.Schedule) == 0 {
			t.Errorf("%s: result stream has no osquery schedule", order)
		}
	}
}

// parseInputs loads corpus entries and parses each as an InputConfig.
// It returns the slice in the same order as the corpus.
func parseInputs(t *testing.T) []InputConfig {
	t.Helper()
	entries, err := loadCorpus()
	if err != nil {
		t.Fatalf("loading corpus: %v", err)
	}
	inputs := make([]InputConfig, 0, len(entries))
	for _, e := range entries {
		cfg, err := ucfgyaml.NewConfig(e.yaml)
		if err != nil {
			t.Fatalf("parsing corpus entry %s: %v", e.dataStream, err)
		}
		var ic InputConfig
		if err := cfg.Unpack(&ic); err != nil {
			t.Fatalf("unpacking corpus entry %s: %v", e.dataStream, err)
		}
		inputs = append(inputs, ic)
	}
	return inputs
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
