// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package module

import (
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/pmezard/go-difflib/difflib"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/mapping"
	"github.com/elastic/beats/v8/libbeat/processors/script/javascript"
	"github.com/elastic/beats/v8/winlogbeat/checkpoint"
	"github.com/elastic/beats/v8/winlogbeat/eventlog"

	// Register javascript modules.
	_ "github.com/elastic/beats/v8/libbeat/processors/script/javascript/module"
)

var update = flag.Bool("update", false, "update golden files")

// Option configures the test behavior.
type Option func(*params)

type params struct {
	ignoreFields []string
}

// WithFieldFilter filters the specified fields from the event prior to
// creating the golden file.
func WithFieldFilter(filter []string) Option {
	return func(p *params) {
		p.ignoreFields = filter
	}
}

// TestPipeline tests the given pipeline by reading events from the .evtx files
// and processing them with the pipeline. Then it compares the results against
// a saved golden file. Use -update to regenerate the golden files.
func TestPipeline(t *testing.T, evtx string, pipeline string, opts ...Option) {
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
			testPipeline(t, f, pipeline, &p)
		})
	}
}

func testPipeline(t testing.TB, evtx string, pipeline string, p *params) {
	t.Helper()

	path, err := filepath.Abs(evtx)
	if err != nil {
		t.Fatal(err)
	}

	// Open evtx file.
	log, err := eventlog.New(common.MustNewConfigFrom(common.MapStr{
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

	// Load javascript processor.
	processor, err := javascript.New(common.MustNewConfigFrom(common.MapStr{
		"file": pipeline,
	}))
	if err != nil {
		t.Fatal(err)
	}

	// Read and process events.
	var events []common.MapStr
	for stop := false; !stop; {
		records, err := log.Read()
		if err == io.EOF {
			stop = true
		} else if err != nil {
			t.Fatal(err)
		}

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

			evt, err := processor.Run(&record)
			if err != nil {
				t.Fatalf("%v while processing event:\n%v", err, record.Fields.StringToPrint())
			}

			// Copy the timestamp to the beat.Event.Fields because this is what
			// we write to the golden data for testing purposes. In the normal
			// Beats output this the handled by the encoder (go-structform).
			if !evt.Timestamp.IsZero() {
				evt.Fields["@timestamp"] = evt.Timestamp.UTC()
			}

			events = append(events, filterEvent(evt.Fields, p.ignoreFields))
		}
	}

	if *update {
		writeGolden(t, path, events)
		return
	}

	expected := readGolden(t, path)
	if !assert.Len(t, events, len(expected)) {
		return
	}
	for i, e := range events {
		assertEqual(t, expected[i], normalize(t, e))
	}
}

// assertEqual asserts that the two objects are deeply equal. If not it will
// error the test and output a diff of the two objects' JSON representation.
func assertEqual(t testing.TB, expected, actual interface{}) bool {
	t.Helper()

	if reflect.DeepEqual(expected, actual) {
		return true
	}

	expJSON, _ := json.MarshalIndent(expected, "", "  ")
	actJSON, _ := json.MarshalIndent(actual, "", "  ")

	diff, _ := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:        difflib.SplitLines(string(expJSON)),
		B:        difflib.SplitLines(string(actJSON)),
		FromFile: "Expected",
		ToFile:   "Actual",
		Context:  1,
	})
	t.Errorf("Expected and actual are different:\n%s", diff)
	return false
}

func writeGolden(t testing.TB, source string, events []common.MapStr) {
	data, err := json.MarshalIndent(events, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll("testdata", 0o755); err != nil {
		t.Fatal(err)
	}

	outPath := filepath.Join("testdata", filepath.Base(source)+".golden.json")
	if err := ioutil.WriteFile(outPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func readGolden(t testing.TB, source string) []common.MapStr {
	inPath := filepath.Join("testdata", filepath.Base(source)+".golden.json")

	data, err := ioutil.ReadFile(inPath)
	if err != nil {
		t.Fatal(err)
	}

	var events []common.MapStr
	if err = json.Unmarshal(data, &events); err != nil {
		t.Fatal(err)
	}

	for _, e := range events {
		lowercaseGUIDs(e)
	}
	return events
}

func normalize(t testing.TB, m common.MapStr) common.MapStr {
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}

	var out common.MapStr
	if err = json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}

	// Lowercase the GUIDs in case tests are run Windows < 2019.
	return lowercaseGUIDs(out)
}

func filterEvent(m common.MapStr, ignores []string) common.MapStr {
	for _, f := range ignores {
		m.Delete(f)
	}
	return m
}

var uppercaseGUIDRegex = regexp.MustCompile(`^{[0-9A-F]{8}-[0-9A-F]{4}-[0-9A-F]{4}-[0-9A-F]{4}-[0-9A-F]{12}}$`)

// lowercaseGUIDs finds string fields that look like GUIDs and converts the hex
// from uppercase to lowercase. Prior to Windows 2019, GUIDs used uppercase hex
// (contrary to RFC 4122).
func lowercaseGUIDs(m common.MapStr) common.MapStr {
	for k, v := range m.Flatten() {
		str, ok := v.(string)
		if !ok {
			continue
		}
		if uppercaseGUIDRegex.MatchString(str) {
			m.Put(k, strings.ToLower(str))
		}
	}
	return m
}

var (
	loadDocumentedFieldsOnce sync.Once
	documentedFields         []string
)

// assertFieldsAreDocumented validates that all fields contained in the event
// are documented in a fields.yml file.
func assertFieldsAreDocumented(t testing.TB, m common.MapStr) {
	t.Helper()

	loadDocumentedFieldsOnce.Do(func() {
		fieldsYml, err := mapping.LoadFieldsYaml("../../../build/fields/fields.all.yml")
		if err != nil {
			t.Fatal("Failed to load generated fields.yml data. Try running 'mage update'.", err)
		}
		documentedFields = fieldsYml.GetKeys()
	})

	for eventFieldName := range m.Flatten() {
		found := false
		for _, documentedFieldName := range documentedFields {
			if strings.HasPrefix(eventFieldName, documentedFieldName) {
				found = true
				break
			}
		}
		if !found {
			assert.Fail(t, "Field not documented", "Key '%v' found in event is not documented.", eventFieldName)
		}
	}
}
