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

	assert.Equal(t, 5, len(events), "Wrong number of returned events")

	testCases := testCases()
	for _, event := range events {
		name, err := event.GetValue("name")
		if err == nil {
			namespace, err := event.GetValue("_module.namespace")
			if err == nil {
				eventKey := namespace.(string) + "@" + name.(string)
				oneTestCase, oneTestCaseFound := testCases[eventKey]
				if oneTestCaseFound {
					for k, v := range oneTestCase {
						testValue(eventKey, t, event, k, v)
					}
					delete(testCases, eventKey)
				}
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

// Test cases built to match 3 examples in 'module/kubernetes/_meta/test/kube-state-metrics'.
// In particular, test same named deployments in different namespaces
func testCases() map[string]map[string]interface{} {
	return map[string]map[string]interface{}{
		"default@jumpy-owl-redis": {
			"_namespace":        "deployment",
			"_module.namespace": "default",

			"name":   "jumpy-owl-redis",
			"paused": false,

			"replicas.available":   0,
			"replicas.desired":     1,
			"replicas.unavailable": 1,
			"replicas.updated":     1,
		},
		"test@jumpy-owl-redis": {
			"_namespace":        "deployment",
			"_module.namespace": "test",

			"name":   "jumpy-owl-redis",
			"paused": true,

			"replicas.available":   6,
			"replicas.desired":     2,
			"replicas.unavailable": 7,
			"replicas.updated":     8,
		},
		"kube-system@tiller-deploy": {
			"_namespace":        "deployment",
			"_module.namespace": "kube-system",

			"name":   "tiller-deploy",
			"paused": false,

			"replicas.available":   1,
			"replicas.desired":     1,
			"replicas.unavailable": 0,
			"replicas.updated":     1,
		},
	}
}
