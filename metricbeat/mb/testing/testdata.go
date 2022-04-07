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

package testing

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/pkg/errors"

	"github.com/mitchellh/hashstructure"
	"gopkg.in/yaml.v2"

	"github.com/elastic/beats/v8/libbeat/asset"
	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/mapping"
	"github.com/elastic/beats/v8/metricbeat/mb"
	"github.com/elastic/beats/v8/metricbeat/mb/testing/flags"

	_ "github.com/elastic/beats/v8/metricbeat/include/fields"
)

const (
	expectedExtension = "-expected.json"
	applicationJson   = "application/json"
)

// DataConfig is the configuration for testdata tests
//
// For example for an http service that mimics the apache status page the following
// configuration could be used:
// ```
// type: http
// url: "/server-status?auto="
// suffix: plain
// omit_documented_fields_check:
//  - "apache.status.hostname"
// remove_fields_from_comparison:
// - "apache.status.hostname"
// module:
//   namespace: test
// ```
// A test will be run for each file with the `plain` extension in the same directory
// where a file with this configuration is placed.
type DataConfig struct {
	// Path is the directory containing this configuration
	Path string

	// WritePath is the path where to write the generated files
	WritePath string

	// The type of the test to run, usually `http`.
	Type string

	// URL of the endpoint that must be tested depending on each module
	URL string

	// ContentType of the data being returned by server
	ContentType string `yaml:"content_type"`

	// Suffix is the extension of the source file with the input contents. Defaults to `json`, `plain` is also a common use.
	Suffix string

	// Module is a map of specific configs that will be appended to a module configuration prior initializing it.
	// For example, the following config in yaml:
	//   module:
	//     namespace: test
	//     foo: bar
	//
	// Will produce the following module config:
	//   - module: http
	//     metricsets:
	//       - json
	//     period: 10s
	//     hosts: ["localhost:80"]
	//     path: "/"
	//     namespace: "test"
	//     foo: bar
	//
	// (notice last two lines)
	Module map[string]interface{} `yaml:"module"`

	// OmitDocumentedFieldsCheck is a list of fields that must be omitted from the function that checks if the field
	// is contained in {metricset}/_meta/fields.yml
	OmitDocumentedFieldsCheck []string `yaml:"omit_documented_fields_check"`

	// RemoveFieldsForComparison
	RemoveFieldsForComparison []string `yaml:"remove_fields_from_comparison"`
}

func defaultDataConfig() DataConfig {
	return DataConfig{
		Path:        ".",
		WritePath:   ".",
		Suffix:      "json",
		ContentType: applicationJson,
	}
}

// ReadDataConfig reads the testdataconfig from a path
func ReadDataConfig(t *testing.T, f string) DataConfig {
	t.Helper()
	config := defaultDataConfig()
	config.Path = filepath.Dir(f)
	config.WritePath = filepath.Dir(config.Path)
	configFile, err := ioutil.ReadFile(f)
	if err != nil {
		t.Fatalf("failed to read '%s': %v", f, err)
	}
	err = yaml.Unmarshal(configFile, &config)
	if err != nil {
		t.Fatalf("failed to parse test configuration file '%s': %v", f, err)
	}
	return config
}

// TestDataConfig is a convenience helper function to read the testdata config
// from the usual path
func TestDataConfig(t *testing.T) DataConfig {
	t.Helper()
	return ReadDataConfig(t, "_meta/testdata/config.yml")
}

// TestDataFiles run tests with config from the usual path (`_meta/testdata`)
func TestDataFiles(t *testing.T, module, metricSet string) {
	t.Helper()
	config := TestDataConfig(t)
	TestDataFilesWithConfig(t, module, metricSet, config)
}

// TestDataFilesWithConfig run tests for a testdata config
func TestDataFilesWithConfig(t *testing.T, module, metricSet string, config DataConfig) {
	t.Helper()
	ff, err := filepath.Glob(filepath.Join(config.Path, "*."+config.Suffix))
	if err != nil {
		t.Fatal(err)
	}
	if len(ff) == 0 {
		t.Fatalf("test path with config but without data files: %s", config.Path)
	}

	var files []string
	for _, f := range ff {
		// Exclude all the expected files
		if strings.HasSuffix(f, expectedExtension) {
			continue
		}
		files = append(files, f)
	}

	for _, f := range files {
		t.Run(filepath.Base(f), func(t *testing.T) {
			runTest(t, f, module, metricSet, config)
		})
	}
}

// TestMetricsetFieldsDocumented checks metricset fields are documented from metricsets that cannot run `TestDataFiles` test which contains this check
func TestMetricsetFieldsDocumented(t *testing.T, metricSet mb.MetricSet, events []mb.Event) {
	var data []common.MapStr
	for _, e := range events {
		beatEvent := StandardizeEvent(metricSet, e, mb.AddMetricSetInfo)
		data = append(data, beatEvent.Fields)
	}

	if err := checkDocumented(data, nil); err != nil {
		t.Errorf("%v: check if fields are documented in `metricbeat/module/%s/%s/_meta/fields.yml` "+
			"file or run 'make update' on Metricbeat folder to update fields in `metricbeat/fields.yml`",
			err, metricSet.Module().Name(), metricSet.Name())
	}

}

func runTest(t *testing.T, file string, module, metricSetName string, config DataConfig) {
	// starts a server serving the given file under the given url
	s := server(t, file, config.URL, config.ContentType)
	defer s.Close()

	moduleConfig := getConfig(module, metricSetName, s.URL, config)
	metricSet := NewMetricSet(t, moduleConfig)

	var events []mb.Event
	var errs []error

	switch v := metricSet.(type) {
	case mb.ReportingMetricSetV2:
		metricSet := NewReportingMetricSetV2(t, moduleConfig)
		events, errs = ReportingFetchV2(metricSet)
	case mb.ReportingMetricSetV2Error:
		metricSet := NewReportingMetricSetV2Error(t, moduleConfig)
		events, errs = ReportingFetchV2Error(metricSet)
	default:
		t.Fatalf("unknown type: %T", v)
	}

	// Gather errors to build also error events
	for _, e := range errs {
		// TODO: for errors strip out and standardise the URL error as it would create a different diff every time
		events = append(events, mb.Event{Error: e})
	}

	var data []common.MapStr

	for _, e := range events {
		beatEvent := StandardizeEvent(metricSet, e, mb.AddMetricSetInfo)
		// Overwrite service.address as the port changes every time
		beatEvent.Fields.Put("service.address", "127.0.0.1:55555")
		data = append(data, beatEvent.Fields)
	}

	// Sorting the events is necessary as events are not necessarily sent in the same order
	sort.SliceStable(data, func(i, j int) bool {
		h1, _ := hashstructure.Hash(data[i], nil)
		h2, _ := hashstructure.Hash(data[j], nil)
		return h1 < h2
	})

	if err := checkDocumented(data, config.OmitDocumentedFieldsCheck); err != nil {
		t.Errorf("%v: check if fields are documented in `metricbeat/module/%s/%s/_meta/fields.yml` "+
			"file or run 'make update' on Metricbeat folder to update fields in `metricbeat/fields.yml`",
			err, module, metricSetName)
	}

	// Overwrites the golden files if run with -generate
	if *flags.DataFlag {
		outputIndented, err := json.MarshalIndent(&data, "", "    ")
		if err != nil {
			t.Fatal(err)
		}
		if err = ioutil.WriteFile(file+expectedExtension, outputIndented, 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Read expected file
	expected, err := ioutil.ReadFile(file + expectedExtension)
	if err != nil {
		t.Fatalf("could not read file: %s", err)
	}

	expectedMap := []common.MapStr{}
	if err := json.Unmarshal(expected, &expectedMap); err != nil {
		t.Fatal(err)
	}

	for _, fieldToRemove := range config.RemoveFieldsForComparison {
		for eventIndex := range data {
			if err := data[eventIndex].Delete(fieldToRemove); err != nil {
				t.Fatal(err)
			}
		}

		for eventIndex := range expectedMap {
			if err := expectedMap[eventIndex].Delete(fieldToRemove); err != nil {
				t.Fatal(err)
			}
		}
	}

	for _, event := range data {
		// ensure the event is in expected list
		found := -1
		for i, expectedEvent := range expectedMap {
			if event.String() == expectedEvent.String() {
				found = i
				break
			}
		}
		if found > -1 {
			expectedMap = append(expectedMap[:found], expectedMap[found+1:]...)
		} else {
			t.Errorf("Event was not expected: %+v", event)
		}
	}

	if len(expectedMap) > 0 {
		t.Error("Some events were missing:")
		for _, e := range expectedMap {
			t.Error(e)
		}
	}

	// If there was some error, fail before trying to write anything.
	if t.Failed() {
		t.FailNow()
	}

	if strings.HasSuffix(file, "docs."+config.Suffix) {
		writeDataJSON(t, data[0], filepath.Join(config.WritePath, "data.json"))
	}
}

func writeDataJSON(t *testing.T, data common.MapStr, path string) {
	// Add hardcoded timestamp
	data.Put("@timestamp", "2019-03-01T08:05:34.853Z")
	output, err := json.MarshalIndent(&data, "", "    ")
	if err = ioutil.WriteFile(path, output, 0644); err != nil {
		t.Fatal(err)
	}
}

// checkDocumented checks that all fields which show up in the events are documented
func checkDocumented(data []common.MapStr, omitFields []string) error {
	fieldsData, err := asset.GetFields("metricbeat")
	if err != nil {
		return err
	}

	fields, err := mapping.LoadFields(fieldsData)
	if err != nil {
		return err
	}
	documentedFields := fields.GetKeys()
	keys := map[string]interface{}{}

	for _, k := range documentedFields {
		keys[k] = struct{}{}
	}

	for _, d := range data {
		flat := d.Flatten()
		if err := documentedFieldCheck(flat, keys, omitFields); err != nil {
			return err
		}
	}

	return nil
}

func documentedFieldCheck(foundKeys common.MapStr, knownKeys map[string]interface{}, omitFields []string) error {
	// Sort all found keys to guarantee consistent validation messages
	sortedFoundKeys := make([]string, 0, len(foundKeys))
	for k := range foundKeys {
		sortedFoundKeys = append(sortedFoundKeys, k)
	}
	sort.Strings(sortedFoundKeys)

	for k := range sortedFoundKeys {
		foundKey := sortedFoundKeys[k]
		if _, ok := knownKeys[foundKey]; !ok {
			for _, omitField := range omitFields {
				if omitDocumentedField(foundKey, omitField) {
					return nil
				}
			}
			// If a field is defined as object it can also have a * somewhere
			// So this checks if such a key with the * exists by testing with it
			splits := strings.Split(foundKey, ".")
			found := false
			for pos := 1; pos < len(splits)-1; pos++ {
				key := strings.Join(splits[0:pos], ".") + ".*." + strings.Join(splits[pos+1:len(splits)], ".")
				if _, ok := knownKeys[key]; ok {
					found = true
					break
				}
			}
			if found {
				continue
			}
			// case `status_codes.*`:
			prefix := strings.Join(splits[0:len(splits)-1], ".")
			if _, ok := knownKeys[prefix+".*"]; ok {
				continue
			}
			// should cover scenarios as status_codes.*.*` and `azure.compute_vm_scaleset.*.*`
			if len(splits) > 2 {
				prefix = strings.Join(splits[0:len(splits)-2], ".")
				if _, ok := knownKeys[prefix+".*.*"]; ok {
					continue
				}
			}

			// case `aws.*.metrics.*.*`:
			if len(splits) == 5 {
				if _, ok := knownKeys[splits[0]+".*."+splits[2]+".*.*"]; ok {
					continue
				}
			}

			return errors.Errorf("field missing '%s'", foundKey)
		}
	}

	return nil
}

// omitDocumentedField returns true if 'field' is exactly like 'omitField' or if 'field' equals the prefix of 'omitField'
// if the latter contains a dot.wildcard ".*". For example:
// field: hello, 						  	omitField: world 					false
// field: hello, 						  	omitField: hello 					true
// field: elasticsearch.stats 			  	omitField: elasticsearch.stats 		true
// field: elasticsearch.stats.hello.world 	omitField: elasticsearch.* 			true
// field: elasticsearch.stats.hello.world 	omitField: * 						true
func omitDocumentedField(field, omitField string) bool {
	if strings.Contains(omitField, "*") {
		// Omit every key prefixed with chars before "*"
		prefixedField := strings.Trim(omitField, ".*")
		if strings.Contains(field, prefixedField) {
			return true
		}
	} else {
		// Omit only if key matches exactly
		if field == omitField {
			return true
		}
	}

	return false
}

// getConfig returns config for elasticsearch module
func getConfig(module, metricSet, url string, config DataConfig) map[string]interface{} {
	moduleConfig := map[string]interface{}{
		"module":     module,
		"metricsets": []string{metricSet},
		"hosts":      []string{url},
	}

	for k, v := range config.Module {
		moduleConfig[k] = v
	}

	return moduleConfig
}

// server starts a server with a mock output
func server(t *testing.T, path string, url string, contentType string) *httptest.Server {

	body, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatalf("could not read file: %s", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := ""
		v := r.URL.Query()
		if len(v) > 0 {
			query += "?" + v.Encode()
		}

		if r.URL.Path+query == url {
			w.Header().Set("Content-Type", contentType)
			w.WriteHeader(200)
			w.Write(body)
		} else {
			w.WriteHeader(404)
		}
	}))
	return server
}
