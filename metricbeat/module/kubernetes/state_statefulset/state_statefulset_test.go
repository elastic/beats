// +build !integration

package state_statefulset

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
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
		"metricsets": []string{"state_statefulset"},
		"hosts":      []string{server.URL},
	}

	f := mbtest.NewEventsFetcher(t, config)

	events, err := f.Fetch()
	assert.NoError(t, err)

	assert.Equal(t, 3, len(events), "Wrong number of returned events")

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
						testValue(t, event, k, v)
					}
					delete(testCases, eventKey)
				}
			}
		}
	}

	if len(testCases) > 0 {
		t.Errorf("Test reference events not found: %v", testCases)
	}
}

func testValue(t *testing.T, event common.MapStr, field string, expected interface{}) {
	data, err := event.GetValue(field)
	assert.NoError(t, err, "Could not read field "+field)
	assert.EqualValues(t, expected, data, "Wrong value for field "+field)
}

func testCases() map[string]map[string]interface{} {
	return map[string]map[string]interface{}{
		"default@elasticsearch": {
			"_module.namespace": "default",
			"name":              "elasticsearch",

			"created":             1511973651,
			"replicas.observed":   1,
			"replicas.desired":    4,
			"generation.observed": 1,
			"generation.desired":  3,
		},
		"default@mysql": {
			"_module.namespace": "default",
			"name":              "mysql",

			"created":             1511989697,
			"replicas.observed":   2,
			"replicas.desired":    5,
			"generation.observed": 2,
			"generation.desired":  4,
		},
		"custom@mysql": {
			"_module.namespace": "custom",
			"name":              "mysql",

			"created":             1511999697,
			"replicas.observed":   3,
			"replicas.desired":    6,
			"generation.observed": 3,
			"generation.desired":  5,
		},
	}
}
