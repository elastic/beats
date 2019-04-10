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

package leader

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"

	"github.com/stretchr/testify/assert"

	"testing"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
)

func TestEventMapping(t *testing.T) {
	content, err := ioutil.ReadFile("../_meta/test/leaderstats.json")
	assert.NoError(t, err)

	event := eventMapping(content)

	assert.Equal(t, event["leader"], string("924e2e83e93f2560"))
}

func TestFetchEventContent(t *testing.T) {
	absPath, err := filepath.Abs("../_meta/test/")
	assert.NoError(t, err)

	response, err := ioutil.ReadFile(absPath + "/leaderstats.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json;")
		w.Write([]byte(response))
	}))
	defer server.Close()

	config := map[string]interface{}{
		"module":     "etcd",
		"metricsets": []string{"leader"},
		"hosts":      []string{server.URL},
	}

	f := mbtest.NewReportingMetricSetV2Error(t, config)
	events, errs := mbtest.ReportingFetchV2Error(f)
	if len(errs) > 0 {
		t.Fatalf("Expected 0 error, had %d. %v\n", len(errs), errs)
	}
	assert.NotEmpty(t, events)

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), events[0])
}
