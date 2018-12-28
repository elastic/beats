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

package connz

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
)

func TestEventMapping(t *testing.T) {
	content, err := ioutil.ReadFile("../_meta/test/connzmetrics.json")
	assert.NoError(t, err)

	event := eventMapping(content)

	metrics, _ := event["metrics"]
	d, _ := metrics.(Connz)

	assert.Equal(t, d.ServerID, string("bUAdpRFtMWddIBWw80Yd9D"))
}

func TestFetchEventContent(t *testing.T) {
	absPath, err := filepath.Abs("../_meta/test/")

	response, err := ioutil.ReadFile(absPath + "/connzmetrics.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json;")
		w.Write([]byte(response))
	}))
	defer server.Close()

	config := map[string]interface{}{
		"module":     "nats",
		"metricsets": []string{"connz"},
		"hosts":      []string{server.URL},
	}
	f := mbtest.NewEventFetcher(t, config)
	event, err := f.Fetch()
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event)
}
