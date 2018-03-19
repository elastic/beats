// +build !integration

package state_node

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
		"metricsets": []string{"state_node"},
		"hosts":      []string{server.URL},
	}

	f := mbtest.NewEventsFetcher(t, config)

	events, err := f.Fetch()
	assert.NoError(t, err)

	assert.Equal(t, 2, len(events), "Wrong number of returned events")

	testCases := testCases()
	for _, event := range events {
		name, err := event.GetValue("name")
		if err == nil {
			eventKey := name.(string)
			oneTestCase, oneTestCaseFound := testCases[eventKey]
			if oneTestCaseFound {
				for k, v := range oneTestCase {
					testValue(eventKey, t, event, k, v)
				}
				delete(testCases, eventKey)
			}
		}
	}

	if len(testCases) > 0 {
		t.Errorf("Test reference events not found: %v, \n\ngot: %v", testCases, events)
	}
}

func testValue(eventKey string, t *testing.T, event common.MapStr, field string, expected interface{}) {
	data, err := event.GetValue(field)
	assert.NoError(t, err, eventKey+": Could not read field "+field)
	assert.EqualValues(t, expected, data, eventKey+": Wrong value for field "+field)
}

func testCases() map[string]map[string]interface{} {
	return map[string]map[string]interface{}{
		"minikube": {
			"_namespace": "node",
			"name":       "minikube",

			"status.ready":         "true",
			"status.unschedulable": false,

			"cpu.allocatable.cores": 2,
			"cpu.capacity.cores":    2,

			"memory.allocatable.bytes": 2097786880,
			"memory.capacity.bytes":    2097786880,

			"pod.allocatable.total": 110,
			"pod.capacity.total":    110,
		},
		"minikube-test": {
			"_namespace": "node",
			"name":       "minikube-test",

			"status.ready":         "true",
			"status.unschedulable": true,

			"cpu.allocatable.cores": 3,
			"cpu.capacity.cores":    4,

			"memory.allocatable.bytes": 3097786880,
			"memory.capacity.bytes":    4097786880,

			"pod.allocatable.total": 210,
			"pod.capacity.total":    310,
		},
	}
}
