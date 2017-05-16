// +build !integration

package state_pod

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
		"metricsets": []string{"state_pod"},
		"hosts":      []string{server.URL},
	}

	f := mbtest.NewEventsFetcher(t, config)

	events, err := f.Fetch()
	assert.NoError(t, err)

	assert.Equal(t, 8, len(events), "Wrong number of returned events")

	testCases := map[string]interface{}{
		"_module.namespace": "default",
		"_module.node.name": "minikube",
		"name":              "jumpy-owl-redis-3481028193-s78x9",

		"host_ip": "192.168.99.100",
		"ip":      "172.17.0.4",

		"status.phase":     "running",
		"status.ready":     "false",
		"status.scheduled": "true",
	}

	for _, event := range events {
		name, err := event.GetValue("name")
		if err == nil && name == "jumpy-owl-redis-3481028193-s78x9" {
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
