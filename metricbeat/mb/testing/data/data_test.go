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

package data

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/pkg/errors"

	"github.com/mitchellh/hashstructure"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"

	"github.com/elastic/beats/libbeat/asset"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/mapping"
	"github.com/elastic/beats/metricbeat/mb"
	mbtesting "github.com/elastic/beats/metricbeat/mb/testing"

	_ "github.com/elastic/beats/metricbeat/include"
	_ "github.com/elastic/beats/metricbeat/include/fields"
)

const (
	expectedExtension = "-expected.json"
)

var (
	// Use `go test -generate` to update files.
	generateFlag = flag.Bool("generate", false, "Write golden files")
	moduleFlag   = flag.String("module", "", "Choose a module to test")
)

type Config struct {
	// The type of the test to run, usually `http`.
	Type string

	// URL of the endpoint that must be tested depending on each module
	URL string

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

func TestAll(t *testing.T) {

	configFiles, _ := filepath.Glob(getModulesPath() + "/*/*/_meta/testdata/config.yml")

	for _, f := range configFiles {
		// get module and metricset name from path
		s := strings.Split(f, string(os.PathSeparator))
		moduleName := s[4]
		metricSetName := s[5]

		if *moduleFlag != "" {
			if *moduleFlag != moduleName {
				continue
			}
		}

		configFile, err := ioutil.ReadFile(f)
		if err != nil {
			log.Printf("yamlFile.Get err   #%v ", err)
		}
		var config Config
		err = yaml.Unmarshal(configFile, &config)
		if err != nil {
			log.Fatalf("Unmarshal: %v", err)
		}

		if config.Suffix == "" {
			config.Suffix = "json"
		}

		getTestdataFiles(t, moduleName, metricSetName, config)
	}
}

func getTestdataFiles(t *testing.T, module, metricSet string, config Config) {
	ff, err := filepath.Glob(getMetricsetPath(module, metricSet) + "/_meta/testdata/*." + config.Suffix)
	if err != nil {
		t.Fatal(err)
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
		t.Run(f, func(t *testing.T) {
			runTest(t, f, module, metricSet, config)
		})
	}
}

func runTest(t *testing.T, file string, module, metricSetName string, config Config) {

	// starts a server serving the given file under the given url
	s := server(t, file, config.URL)
	defer s.Close()

	moduleConfig := getConfig(module, metricSetName, s.URL, config)
	metricSet := mbtesting.NewMetricSet(t, moduleConfig)

	var events []mb.Event
	var errs []error

	switch v := metricSet.(type) {
	case mb.ReportingMetricSetV2:
		metricSet := mbtesting.NewReportingMetricSetV2(t, moduleConfig)
		events, errs = mbtesting.ReportingFetchV2(metricSet)
	case mb.ReportingMetricSetV2Error:
		metricSet := mbtesting.NewReportingMetricSetV2Error(t, moduleConfig)
		events, errs = mbtesting.ReportingFetchV2Error(metricSet)
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
		beatEvent := mbtesting.StandardizeEvent(metricSet, e, mb.AddMetricSetInfo)
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

	if err := checkDocumented(t, data, config.OmitDocumentedFieldsCheck); err != nil {
		t.Errorf("'%v' check if fields are documented in `metricbeat/{module}/{metricset}/_meta/fields.yml` "+
			"file or run 'make update' on Metricbeat folder to update root `metricbeat/fields.yml` in ", err)
	}

	// Overwrites the golden files if run with -generate
	if *generateFlag {
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

	output, err := json.Marshal(&data)
	if err != nil {
		t.Fatal(err)
	}

	expectedJSON, err := json.Marshal(&expectedMap)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, string(expectedJSON), string(output))

	if strings.HasSuffix(file, "docs."+config.Suffix) {
		writeDataJSON(t, data[0], module, metricSetName)
	}
}

func writeDataJSON(t *testing.T, data common.MapStr, module, metricSet string) {
	// Add hardcoded timestamp
	data.Put("@timestamp", "2019-03-01T08:05:34.853Z")
	output, err := json.MarshalIndent(&data, "", "    ")
	if err = ioutil.WriteFile(getMetricsetPath(module, metricSet)+"/_meta/data.json", output, 0644); err != nil {
		t.Fatal(err)
	}
}

// checkDocumented checks that all fields which show up in the events are documented
func checkDocumented(t *testing.T, data []common.MapStr, omitFields []string) error {
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
	for foundKey := range foundKeys {
		if _, ok := knownKeys[foundKey]; !ok {
			for _, omitField := range omitFields {
				if omitDocumentedField(foundKey, omitField) {
					return nil
				}
			}
			// If a field is defined as object it can also be defined as `status_codes.*`
			// So this checks if such a key with the * exists by removing the last part.
			splits := strings.Split(foundKey, ".")
			prefix := strings.Join(splits[0:len(splits)-1], ".")
			if _, ok := knownKeys[prefix+".*"]; ok {
				continue
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

func TestOmitDocumentedField(t *testing.T) {
	tts := []struct {
		a, b   string
		result bool
	}{
		{a: "hello", b: "world", result: false},
		{a: "hello", b: "hello", result: true},
		{a: "elasticsearch.stats", b: "elasticsearch.stats", result: true},
		{a: "elasticsearch.stats.hello.world", b: "elasticsearch.*", result: true},
		{a: "elasticsearch.stats.hello.world", b: "*", result: true},
	}

	for _, tt := range tts {
		result := omitDocumentedField(tt.a, tt.b)
		assert.Equal(t, tt.result, result)
	}
}

// GetConfig returns config for elasticsearch module
func getConfig(module, metricSet, url string, config Config) map[string]interface{} {
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
func server(t *testing.T, path string, url string) *httptest.Server {

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
			w.Header().Set("Content-Type", "application/json;")
			w.WriteHeader(200)
			w.Write(body)
		} else {
			w.WriteHeader(404)
		}
	}))
	return server
}

func getModulesPath() string {
	return "../../../module"
}

func getModulePath(module string) string {
	return getModulesPath() + "/" + module
}

func getMetricsetPath(module, metricSet string) string {
	return getModulePath(module) + "/" + metricSet
}
