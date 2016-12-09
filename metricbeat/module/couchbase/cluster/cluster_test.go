// +build !integration

package cluster

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"

	"github.com/stretchr/testify/assert"
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

	assert.EqualValues(t, 300, event["quota.index_memory.mb"])
	assert.EqualValues(t, 300, event["quota.memory.mb"])
	assert.EqualValues(t, 10, event["max_bucket_count"])
	assert.EqualValues(t, 46902679716, event["hdd.free.bytes"])
	assert.EqualValues(t, 63381999616, event["hdd.quota_total.bytes"])
	assert.EqualValues(t, 63381999616, event["hdd.total.bytes"])
	assert.EqualValues(t, 16479319900, event["hdd.used.bytes"])
	assert.EqualValues(t, 16369010, event["hdd.used.by_data.bytes"])
	assert.EqualValues(t, 314572800, event["ram.quota.total.bytes"])
	assert.EqualValues(t, 314572800, event["ram.quota.total.per_node.bytes"])
	assert.EqualValues(t, 104857600, event["ram.quota.used.bytes"])
	assert.EqualValues(t, 104857600, event["ram.quota.used.per_node.bytes"])
	assert.EqualValues(t, 8359174144, event["ram.total.bytes"])
	assert.EqualValues(t, 8004751360, event["ram.used.bytes"])
	assert.EqualValues(t, 53962016, event["ram.used.by_data.bytes"])
}
