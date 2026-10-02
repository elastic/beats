// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package config

import (
	"embed"
	"fmt"
	"io/fs"
	"slices"
	"strings"
	"testing"

	ucfgyaml "github.com/elastic/go-ucfg/yaml"
)

// TestOTelParityConfig proves that osquerybeat's InputConfig parser correctly
// consumes each corpus stream fixture, including the osquery block that
// elastic-agent injects into the result stream. This documents the contract
// that the OTel translation in elastic-agent must satisfy.
func TestOTelParityConfig(t *testing.T) {
	inputs := parseInputs(t)

	result := findInput(inputs, DefaultDataset)
	if result == nil {
		t.Errorf("result stream (%s) not found in corpus", DefaultDataset)
	} else {
		if result.Osquery == nil {
			t.Errorf("result stream: Osquery is nil; osquery block was dropped")
		} else if len(result.Osquery.Schedule) == 0 {
			t.Errorf("result stream: Osquery.Schedule is empty; no queries present")
		}
	}

	if findInput(inputs, DefaultActionResponsesDataset) == nil {
		t.Errorf("action responses stream (%s) not found in corpus", DefaultActionResponsesDataset)
	}
	if findInput(inputs, DefaultQueryProfileDataset) == nil {
		t.Errorf("query profile stream (%s) not found in corpus", DefaultQueryProfileDataset)
	}
}

// TestOTelParityOrdering checks that all three osquerybeat corpus streams are
// present and found by dataset regardless of their order in the slice.
func TestOTelParityOrdering(t *testing.T) {
	inputs := parseInputs(t)

	// Reverse the order to simulate a non-standard delivery sequence.
	slices.Reverse(inputs)

	if findInput(inputs, DefaultDataset) == nil {
		t.Errorf("result stream (%s) not found after reordering", DefaultDataset)
	}
	if findInput(inputs, DefaultActionResponsesDataset) == nil {
		t.Errorf("action responses stream (%s) not found after reordering", DefaultActionResponsesDataset)
	}
	if findInput(inputs, DefaultQueryProfileDataset) == nil {
		t.Errorf("query profile stream (%s) not found after reordering", DefaultQueryProfileDataset)
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

// findInput returns the first InputConfig whose Datastream.Dataset equals dataset,
// or nil if none is found.
func findInput(inputs []InputConfig, dataset string) *InputConfig {
	for i := range inputs {
		if inputs[i].Datastream.Dataset == dataset {
			return &inputs[i]
		}
	}
	return nil
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
