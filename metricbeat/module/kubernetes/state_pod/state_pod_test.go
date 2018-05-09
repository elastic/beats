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

	assert.Equal(t, 9, len(events), "Wrong number of returned events")

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
		t.Errorf("Test reference events not found: %v\n\n got: %v", testCases, events)
	}
}

func testValue(eventKey string, t *testing.T, event common.MapStr, field string, expected interface{}) {
	data, err := event.GetValue(field)
	assert.NoError(t, err, eventKey+": Could not read field "+field)
	assert.EqualValues(t, expected, data, eventKey+": Wrong value for field "+field)
}

// Test cases built to match 3 examples in 'module/kubernetes/_meta/test/kube-state-metrics'.
// In particular, test same named pods in different namespaces
func testCases() map[string]map[string]interface{} {
	return map[string]map[string]interface{}{
		"default@jumpy-owl-redis-3481028193-s78x9": {
			"_namespace":        "pod",
			"_module.namespace": "default",
			"_module.node.name": "minikube",
			"name":              "jumpy-owl-redis-3481028193-s78x9",

			"host_ip": "192.168.99.100",
			"ip":      "172.17.0.4",

			"status.phase":     "succeeded",
			"status.ready":     "false",
			"status.scheduled": "true",
		},
		"test@jumpy-owl-redis-3481028193-s78x9": {
			"_namespace":        "pod",
			"_module.namespace": "test",
			"_module.node.name": "minikube-test",
			"name":              "jumpy-owl-redis-3481028193-s78x9",

			"host_ip": "192.168.99.200",
			"ip":      "172.17.0.5",

			"status.phase":     "running",
			"status.ready":     "true",
			"status.scheduled": "false",
		},
		"jenkins@wise-lynx-jenkins-1616735317-svn6k": {
			"_namespace":        "pod",
			"_module.namespace": "jenkins",
			"_module.node.name": "minikube",
			"name":              "wise-lynx-jenkins-1616735317-svn6k",

			"host_ip": "192.168.99.100",
			"ip":      "172.17.0.7",

			"status.phase":     "running",
			"status.ready":     "true",
			"status.scheduled": "true",
		},
	}
}
