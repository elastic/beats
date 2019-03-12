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

	"github.com/mitchellh/hashstructure"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"

	// TODO: generate include file for these tests automatically moving forward
	_ "github.com/elastic/beats/metricbeat/module/couchbase/cluster"
	_ "github.com/elastic/beats/metricbeat/module/couchbase/node"
	_ "github.com/elastic/beats/metricbeat/module/kibana/status"
	_ "github.com/elastic/beats/metricbeat/module/kubernetes/apiserver"
	_ "github.com/elastic/beats/metricbeat/module/php_fpm/pool"
	_ "github.com/elastic/beats/metricbeat/module/php_fpm/process"
	_ "github.com/elastic/beats/metricbeat/module/rabbitmq/connection"
	_ "github.com/elastic/beats/metricbeat/module/traefik/health"
)

const (
	expectedExtension = "-expected.json"
)

var (
	// Use `go test -generate` to update files.
	generateFlag = flag.Bool("generate", false, "Write golden files")
)

type Config struct {
	Type   string
	URL    string
	Suffix string
}

func TestAll(t *testing.T) {

	configFiles, _ := filepath.Glob(getModulesPath() + "/*/*/_meta/testdata/config.yml")

	for _, f := range configFiles {
		// get module and metricset name from path
		s := strings.Split(f, string(os.PathSeparator))
		moduleName := s[3]
		metricSetName := s[4]

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

		getTestdataFiles(t, config.URL, moduleName, metricSetName, config.Suffix)
	}
}

func getTestdataFiles(t *testing.T, url, module, metricSet, suffix string) {

	ff, _ := filepath.Glob(getMetricsetPath(module, metricSet) + "/_meta/testdata/*." + suffix)
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
			runTest(t, f, module, metricSet, url, suffix)
		})
	}
}

func runTest(t *testing.T, file string, module, metricSetName, url, suffix string) {

	// starts a server serving the given file under the given url
	s := server(t, file, url)
	defer s.Close()

	metricSet := newMetricSet(t, getConfig(module, metricSetName, s.URL))

	var events []mb.Event
	var errs []error

	switch v := metricSet.(type) {
	case mb.ReportingMetricSetV2:
		metricSet := NewReportingMetricSetV2(t, getConfig(module, metricSetName, s.URL))
		events, errs = ReportingFetchV2(metricSet)
	case mb.ReportingMetricSetV2Error:
		metricSet := NewReportingMetricSetV2Error(t, getConfig(module, metricSetName, s.URL))
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

	output, err := json.MarshalIndent(&data, "", "    ")
	if err != nil {
		t.Fatal(err)
	}

	// Overwrites the golden files if run with -generate
	if *generateFlag {
		if err = ioutil.WriteFile(file+expectedExtension, output, 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Read expected file
	expected, err := ioutil.ReadFile(file + expectedExtension)
	if err != nil {
		t.Fatalf("could not read file: %s", err)
	}

	assert.Equal(t, string(expected), string(output))

	if strings.HasSuffix(file, "docs."+suffix) {
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

// GetConfig returns config for elasticsearch module
func getConfig(module, metricSet, url string) map[string]interface{} {
	return map[string]interface{}{
		"module":     module,
		"metricsets": []string{metricSet},
		"hosts":      []string{url},
	}
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
	return "../../module"
}

func getModulePath(module string) string {
	return getModulesPath() + "/" + module
}

func getMetricsetPath(module, metricSet string) string {
	return getModulePath(module) + "/" + metricSet
}
