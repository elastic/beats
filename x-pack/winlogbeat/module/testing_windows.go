// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package module

import (
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/winlogbeat/checkpoint"
	"github.com/elastic/beats/v7/winlogbeat/eventlog"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/go-sysinfo/providers/windows"
)

// TestCollectionPipeline tests the partial pipeline by reading events from the .evtx files
// and processing them with a basic enrichment. Then it compares the results against
// a saved golden file. Use -update to regenerate the golden files.
func TestCollectionPipeline(t *testing.T, evtx string, opts ...Option) {
	// FIXME: We cannot generate golden files on Windows 2022.
	if *update {
		os, err := windows.OperatingSystem()
		if err != nil {
			t.Fatalf("failed to get operating system info: %v", err)
		}
		if strings.Contains(os.Name, "2022") {
			t.Fatal("cannot generate golden files on Windows 2022: see note in powershell/test/powershell_windows_test.go")
		}
	}

	files, err := filepath.Glob(evtx)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("glob", evtx, "didn't match any files")
	}

	var p params
	for _, o := range opts {
		o(&p)
	}

	for _, f := range files {
		t.Run(filepath.Base(f), func(t *testing.T) {
			testCollectionPipeline(t, f, &p)
		})
	}
}

func testCollectionPipeline(t testing.TB, evtx string, p *params) {
	t.Helper()

	path, err := filepath.Abs(evtx)
	if err != nil {
		t.Fatal(err)
	}

	// Open evtx file.
	log, err := eventlog.New(config.MustNewConfigFrom(mapstr.M{
		"name":           path,
		"api":            "wineventlog",
		"no_more_events": "stop",
	}))
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()

	if err = log.Open(checkpoint.EventLogState{}); err != nil {
		t.Fatal(err)
	}

	// Read and process events.
	var events []mapstr.M
	for stop := false; !stop; {
		records, err := log.Read()
		if err == io.EOF { //nolint:errorlint // io.EOF should never be wrapped.
			stop = true
		} else if err != nil {
			t.Fatal(err)
		}

		//nolint:errcheck // All the errors returned here are from beat.Event queries and may be ignored.
		for _, r := range records {
			record := r.ToEvent()

			// Validate fields in event against fields.yml.
			assertFieldsAreDocumented(t, record.Fields)

			record.Delete("event.created")
			record.Delete("log.file")

			// Enrichment based on user.identifier varies based on the host
			// where this is execute so remove it.
			if userType, _ := record.GetValue("winlog.user.type"); userType != "Well Known Group" {
				record.Delete("winlog.user.type")
				record.Delete("winlog.user.name")
				record.Delete("winlog.user.domain")
			}

			// Copy the timestamp to the beat.Event.Fields because this is what
			// we write to the golden data for testing purposes. In the normal
			// Beats output this the handled by the encoder (go-structform).
			evt := &record
			if !evt.Timestamp.IsZero() {
				evt.Fields["@timestamp"] = evt.Timestamp.UTC()
			}

			events = append(events, evt.Fields)
		}
	}

	if *update {
		writeGolden(t, path, "testdata/collection", events)
		return
	}

	expected := readGolden(t, path, "testdata/collection")
	if !assert.Len(t, events, len(expected)) {
		return
	}
	for i, e := range events {
		assertEqual(t, filterEvent(expected[i], p.ignoreFields), normalize(t, filterEvent(e, p.ignoreFields)))
	}
}
