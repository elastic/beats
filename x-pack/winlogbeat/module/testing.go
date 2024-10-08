// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package module

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"

	devtools "github.com/elastic/beats/v7/dev-tools/mage"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/esleg/eslegclient"
	"github.com/elastic/beats/v7/libbeat/mapping"
	"github.com/elastic/beats/v7/libbeat/version"
	"github.com/elastic/beats/v7/winlogbeat/module"
	"github.com/elastic/beats/v7/x-pack/winlogbeat/module/wintest"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/transport/httpcommon"
)

var update = flag.Bool("update", false, "update golden files")

// Option configures the test behavior.
type Option func(*params)

type params struct {
	ignoreFields []string
}

// WithFieldFilter filters the specified fields from the event prior to
// comparison of values, but retains them in the written golden files.
func WithFieldFilter(filter []string) Option {
	return func(p *params) {
		p.ignoreFields = filter
	}
}

// TestIngestPipeline tests the partial pipeline by reading events from the .json files
// and processing them through the ingest pipeline. Then it compares the results against
// a saved golden file. Use -update to regenerate the golden files.
func TestIngestPipeline(t *testing.T, pipeline, json string, opts ...Option) {
	var p params
	for _, o := range opts {
		o(&p)
	}
	testIngestPipeline(t, pipeline, json, &p)
}

func testIngestPipeline(t *testing.T, pipeline, pattern string, p *params) {
	const (
		host        = "http://localhost:9200"
		user        = "admin"
		pass        = "testing"
		indexPrefix = "winlogbeat-test"
	)

	paths, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("failed to expand glob pattern %q", pattern)
	}
	if len(paths) == 0 {
		t.Fatal("glob", pattern, "didn't match any files")
	}

	done, _, err := wintest.Docker(".", "test", testing.Verbose())
	if err != nil {
		t.Fatal(err)
	}
	if *wintest.KeepRunning {
		fmt.Fprintln(os.Stdout, "Use this to manually cleanup containers: docker-compose", "-p", devtools.DockerComposeProjectName(), "rm", "--stop", "--force")
	}
	t.Cleanup(func() {
		stop := !*wintest.KeepRunning
		err = done(stop)
		if err != nil {
			t.Errorf("unexpected error during cleanup: %v", err)
		}
	})

	// Currently we are using mixed API because beats is using the old ES API package,
	// while SimulatePipeline is using the official v8 client package.
	conn, err := eslegclient.NewConnection(eslegclient.ConnectionSettings{
		URL:              host,
		Username:         user,
		Password:         pass,
		CompressionLevel: 3,
		Transport:        httpcommon.HTTPTransportSettings{Timeout: time.Minute},
	})
	if err != nil {
		t.Fatalf("unexpected error making connection: %v", err)
	}
	defer conn.Close()

	err = conn.Connect()
	if err != nil {
		t.Fatalf("unexpected error making connection: %v", err)
	}

	info := beat.Info{
		IndexPrefix: indexPrefix,
		Version:     version.GetDefaultVersion(),
	}
	loaded, err := module.UploadPipelines(info, conn, true)
	if err != nil {
		t.Errorf("unexpected error uploading pipelines: %v", err)
	}
	wantPipelines := []string{
		"powershell",
		"powershell_operational",
		"routing",
		"security",
		"sysmon",
	}
	if len(loaded) != len(wantPipelines) {
		t.Fatalf("unexpected number of loaded pipelines: got:%d want:%d", len(loaded), len(wantPipelines))
	}
	want := regexp.MustCompile(`^` + indexPrefix + `-.*-(` + strings.Join(wantPipelines, "|") + `)$`)
	pipelines := make(map[string]string)
	for _, p := range loaded {
		m := want.FindAllStringSubmatch(p, -1)
		pipelines[m[0][1]] = p
	}
	_, ok := pipelines[pipeline]
	if !ok {
		t.Fatalf("failed to upload %q", pipeline)
	}

	cases, err := wintest.SimulatePipeline(host, user, pass, pipelines[pipeline], paths)
	if err != nil {
		t.Fatalf("unexpected error running simulate: %v", err)
	}
	for _, k := range cases {
		name := filepath.Base(k.Path)
		t.Run(name, func(t *testing.T) {
			if k.Err != nil {
				t.Errorf("unexpected error: %v", k.Err)
			}

			var events []mapstr.M
			for i, p := range k.Processed {
				err = wintest.ErrorMessage(p)
				if err != nil {
					t.Errorf("unexpected ingest error for event %d: %v", i, err)
				}

				var event mapstr.M
				err = json.Unmarshal(p, &event)
				if err != nil {
					t.Fatalf("failed to unmarshal event into mapstr: %v", err)
				}

				// Validate fields in event against fields.yml.
				assertFieldsAreDocumented(t, event)

				event.Delete("event.created")
				event.Delete("log.file")

				// Enrichment based on user.identifier varies based on the host
				// where this is execute so remove it.
				if userType, _ := event.GetValue("winlog.user.type"); userType != "Well Known Group" {
					event.Delete("winlog.user.type")
					event.Delete("winlog.user.name")
					event.Delete("winlog.user.domain")
				}

				events = append(events, event)
			}

			path, err := filepath.Abs(k.Path)
			if err != nil {
				t.Fatal(err)
			}
			path = strings.TrimSuffix(path, ".evtx.golden.json")

			if *update {
				writeGolden(t, path, "testdata/ingest", events)
				return
			}

			expected := readGolden(t, path, "testdata/ingest")
			if !assert.Len(t, events, len(expected)) {
				return
			}
			for i, e := range events {
				assertEqual(t, filterEvent(expected[i], p.ignoreFields), normalize(t, filterEvent(e, p.ignoreFields)))
			}
		})
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

	t.Errorf("Expected and actual are different:\n%s",
		cmp.Diff(string(expJSON), string(actJSON)))
	return false
}

func writeGolden(t testing.TB, source, dir string, events []mapstr.M) {
	data, err := json.MarshalIndent(events, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}

	outPath := filepath.Join(dir, filepath.Base(source)+".golden.json")
	if err := ioutil.WriteFile(outPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func readGolden(t testing.TB, source, dir string) []mapstr.M {
	inPath := filepath.Join(dir, filepath.Base(source)+".golden.json")

	data, err := ioutil.ReadFile(inPath)
	if err != nil {
		t.Fatal(err)
	}

	var events []mapstr.M
	if err = json.Unmarshal(data, &events); err != nil {
		t.Fatal(err)
	}

	for _, e := range events {
		lowercaseGUIDs(e)
	}
	return events
}

func normalize(t testing.TB, m mapstr.M) mapstr.M {
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}

	var out mapstr.M
	if err = json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}

	// Lowercase the GUIDs in case tests are run Windows < 2019.
	return lowercaseGUIDs(out)
}

func filterEvent(m mapstr.M, ignores []string) mapstr.M {
	for _, f := range ignores {
		m.Delete(f)
	}
	return m
}

var uppercaseGUIDRegex = regexp.MustCompile(`^{[0-9A-F]{8}-[0-9A-F]{4}-[0-9A-F]{4}-[0-9A-F]{4}-[0-9A-F]{12}}$`)

// lowercaseGUIDs finds string fields that look like GUIDs and converts the hex
// from uppercase to lowercase. Prior to Windows 2019, GUIDs used uppercase hex
// (contrary to RFC 4122).
func lowercaseGUIDs(m mapstr.M) mapstr.M {
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
func assertFieldsAreDocumented(t testing.TB, m mapstr.M) {
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
