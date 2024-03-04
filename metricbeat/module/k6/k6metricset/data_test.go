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

package k6metricset

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestFetchEventContents(t *testing.T) {
	response, err := ioutil.ReadFile("./_meta/testdata/k6metrics.json")
	assert.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json;")
		w.Write([]byte(response))
	}))
	defer server.Close()

	config := map[string]interface{}{
		"module":     "k6",
		"metricsets": []string{"k6metricset"},
		"hosts":      []string{server.URL},
	}

	f := mbtest.NewReportingMetricSetV2Error(t, config)
	events, errs := mbtest.ReportingFetchV2Error(f)
	if len(errs) > 0 {
		t.Fatalf("Expected 0 error, had %d. %v\n", len(errs), errs)
	}
	assert.NotEmpty(t, events)
	event := events[0].MetricSetFields

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event.StringToPrint())

}

func TestEventMapping(t *testing.T) {
	content, err := ioutil.ReadFile("./_meta/testdata/k6metrics.json")
	assert.NoError(t, err)

	event, _ := eventMapping(content)
	if err != nil {
		t.Fatal(err)
	}

	expected := mapstr.M{
		"data": mapstr.M{
			"metrics": mapstr.M{
				"vus": mapstr.M{
					"value": 1,
				},
				"http_req_duration": mapstr.M{
					"avg":   386.793313,
					"max":   430.822131,
					"med":   386.793313,
					"p(90)": 422.01636740000004,
					"p(95)": 426.4192492,
				},
				"http_req_tls_handshaking": mapstr.M{
					"avg":   61.844635,
					"max":   62.025681,
					"med":   61.844635,
					"p(90)": 61.9894718,
					"p(95)": 62.0075764,
				},
				"vus_max": mapstr.M{
					"value": 1,
				},
				"http_req_receiving": mapstr.M{
					"avg":   7.3451695,
					"max":   13.807923,
					"med":   7.345169500000001,
					"p(90)": 12.515372300000001,
					"p(95)": 13.16164765,
				},
				"http_req_sending": mapstr.M{
					"avg":   1.2124685,
					"max":   2.309611,
					"med":   1.2124685,
					"p(90)": 2.0901824999999996,
					"p(95)": 2.1998967499999997,
				},
				"http_req_connecting": mapstr.M{
					"avg":   19.264117499999998,
					"max":   19.343144,
					"med":   19.264117499999998,
					"p(90)": 19.3273387,
					"p(95)": 19.33524135,
				},
				"http_reqs": mapstr.M{
					"count": 2,
					"rate":  0.07031716558239108,
				},
				"http_req_waiting": mapstr.M{
					"avg":   378.235675,
					"max":   416.898882,
					"med":   378.235675,
					"p(90)": 409.16624060000004,
					"p(95)": 413.0325613,
				},
			},
		},
	}

	assert.Equal(t, expected, event)

}

func TestEventMapping_InvalidJSON(t *testing.T) {
	// Generate invalid JSON data
	invalidJSON := []byte(`{"data": { "metrics": [1, 2, 3] } }`)

	// call eventMapping and wait for an error
	_, err := eventMapping(invalidJSON)

	// Check if the error is an expected error type
	if err == nil {
		t.Errorf("Error expected, but no error received.")
	} else if err.Error() != "JSON unmarshall fail: json: cannot unmarshal object into Go struct field Data.data of type []k6metricset.Metric" {
		t.Errorf("The expected error text was not received. Expected: 'json: cannot unmarshal object into Go struct field Data.data of type []k6metricset.Metric', Received: %v", err)
	}

}
