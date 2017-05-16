// +build !integration

package state_deployment

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/elastic/beats/libbeat/common"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"

	"github.com/stretchr/testify/assert"
)

const testFile = "../_meta/test/kube-state-metrics"

func TestEventMapping(t *testing.T) {

	file, err := os.Open(testFile)
	assert.NoError(t, err, "cannot open test file "+testFile)

	body, err := ioutil.ReadAll(file)
	assert.NoError(t, err, "cannot read test file "+testFile)

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "text/plain; charset=ISO-8859-1")
		w.Write([]byte(body))
	}))

	server.Start()
	defer server.Close()

	config := map[string]interface{}{
		"module":     "kubernetes",
		"metricsets": []string{"state_deployment"},
		"hosts":      []string{server.URL},
	}

	f := mbtest.NewEventsFetcher(t, config)

	events, err := f.Fetch()
	assert.NoError(t, err)

	assert.Equal(t, 4, len(events), "Wrong number of returned events")

	testCases := map[string]interface{}{
		"_module.namespace": "default",

		"name":   "jumpy-owl-redis",
		"paused": false,

		"replicas.available":   0,
		"replicas.desired":     1,
		"replicas.unavailable": 1,
		"replicas.updated":     1,
	}

	for _, event := range events {
		name, err := event.GetValue("name")
		if err == nil && name == "jumpy-owl-redis" {
			for k, v := range testCases {
				testValue(t, event, k, v)
			}
			return
		}
	}

	t.Error("Test reference event not found")
}

func testValue(t *testing.T, event common.MapStr, field string, expected interface{}) {
	data, err := event.GetValue(field)
	assert.NoError(t, err, "Could not read field "+field)
	assert.EqualValues(t, expected, data, "Wrong value for field "+field)
}
