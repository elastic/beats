// +build !integration

package cluster

import (
	"net/http"
	"net/http/httptest"
	"testing"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"

	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"path/filepath"
)

func TestFetchEventContents(t *testing.T) {
	absPath, err := filepath.Abs("./testdata/")
	// response is a raw response from a couchbase
	response, err := ioutil.ReadFile(absPath + "/sample_response.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "appication/json;")
		w.Write([]byte(response))
	}))
	defer server.Close()

	config := map[string]interface{}{
		"module":     "couchbase",
		"metricsets": []string{"cluster"},
		"hosts":      []string{server.URL},
	}

	f := mbtest.NewEventFetcher(t, config)
	event, err := f.Fetch()
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event.StringToPrint())

	assert.EqualValues(t, 300, event["indexMemoryQuota"])
	assert.EqualValues(t, 10, event["maxBucketCount"])
	assert.EqualValues(t, 300, event["memoryQuota"])
	assert.EqualValues(t, 46902679716, event["hdd_free"])
	assert.EqualValues(t, 63381999616, event["hdd_quotaTotal"])
	assert.EqualValues(t, 63381999616, event["hdd_total"])
	assert.EqualValues(t, 16479319900, event["hdd_used"])
	assert.EqualValues(t, 16369010, event["hdd_usedByData"])
	assert.EqualValues(t, 314572800, event["ram_quotaTotal"])
	assert.EqualValues(t, 314572800, event["ram_quotaTotalPerNode"])
	assert.EqualValues(t, 104857600, event["ram_quotaUsed"])
	assert.EqualValues(t, 104857600, event["ram_quotaUsedPerNode"])
	assert.EqualValues(t, 8359174144, event["ram_total"])
	assert.EqualValues(t, 8004751360, event["ram_used"])
	assert.EqualValues(t, 53962016, event["ram_usedByData"])
}
