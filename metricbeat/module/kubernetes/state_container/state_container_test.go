// +build !integration

package state_container

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
		"metricsets": []string{"state_container"},
		"hosts":      []string{server.URL},
	}

	f := mbtest.NewEventsFetcher(t, config)

	events, err := f.Fetch()
	assert.NoError(t, err)

	assert.Equal(t, 11, len(events), "Wrong number of returned events")

	testCases := testCases()
	for _, event := range events {
		name, err := event.GetValue("name")
		if err == nil {
			namespace, err := event.GetValue("_module.namespace")
			if err != nil {
				continue
			}
			pod, err := event.GetValue("_module.pod.name")
			if err != nil {
				continue
			}
			eventKey := namespace.(string) + "@" + pod.(string) + "@" + name.(string)
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

// Test cases built to match 3 examples in 'module/kubernetes/_meta/test/kube-state-metrics'.
// In particular, test same named containers  in different namespaces
func testCases() map[string]map[string]interface{} {
	return map[string]map[string]interface{}{
		"kube-system@kube-dns-v20-5g5cb@kubedns": {
			"_namespace":        "container",
			"_module.namespace": "kube-system",
			"_module.node.name": "minikube",
			"_module.pod.name":  "kube-dns-v20-5g5cb",
			"name":              "kubedns",
			"id":                "docker://fa3d83f648de42492b38fa3e8501d109376f391c50f2bd210c895c8477ae4b62",

			"image":           "gcr.io/google_containers/kubedns-amd64:1.9",
			"status.phase":    "running",
			"status.ready":    true,
			"status.restarts": 2,

			"memory.limit.bytes":    178257920,
			"memory.request.bytes":  73400320,
			"cpu.request.cores":     0.1,
			"cpu.request.nanocores": 1e+08,
		},
		"test@kube-dns-v20-5g5cb-test@kubedns": {
			"_namespace":        "container",
			"_module.namespace": "test",
			"_module.node.name": "minikube-test",
			"_module.pod.name":  "kube-dns-v20-5g5cb-test",
			"name":              "kubedns",
			"id":                "docker://fa3d83f648de42492b38fa3e8501d109376f391c50f2bd210c895c8477ae4b62-test",

			"image":           "gcr.io/google_containers/kubedns-amd64:1.9-test",
			"status.phase":    "terminated",
			"status.ready":    false,
			"status.restarts": 3,

			"memory.limit.bytes":    278257920,
			"memory.request.bytes":  83400320,
			"cpu.request.cores":     0.2,
			"cpu.request.nanocores": 2e+08,
		},
		"kube-system@kube-dns-v20-5g5cb@healthz": {
			"_namespace":        "container",
			"_module.namespace": "kube-system",
			"_module.node.name": "minikube",
			"_module.pod.name":  "kube-dns-v20-5g5cb",
			"name":              "healthz",
			"id":                "docker://52fa55e051dc5b68e44c027588685b7edd85aaa03b07f7216d399249ff4fc821",

			"image":           "gcr.io/google_containers/exechealthz-amd64:1.2",
			"status.phase":    "running",
			"status.ready":    true,
			"status.restarts": 2,

			"memory.limit.bytes":    52428800,
			"memory.request.bytes":  52428800,
			"cpu.request.cores":     0.01,
			"cpu.request.nanocores": 1e+07,
		},
	}
}
