// +build !integration

package cluster

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"

	"github.com/elastic/beats/libbeat/common"
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

	hdd := event["hdd"].(common.MapStr)
	assert.EqualValues(t, 46902679716, hdd["free.bytes"])
	assert.EqualValues(t, 63381999616, hdd["total.bytes"])

	hdd_used := hdd["used"].(common.MapStr)
	assert.EqualValues(t, 16479319900, hdd_used["value.bytes"])
	assert.EqualValues(t, 16369010, hdd_used["by_data.bytes"])

	hdd_quota := hdd["quota"].(common.MapStr)
	assert.EqualValues(t, 63381999616, hdd_quota["total.bytes"])

	assert.EqualValues(t, 10, event["max_bucket_count"])

	quota := event["quota"].(common.MapStr)
	assert.EqualValues(t, 300, quota["index_memory.mb"])
	assert.EqualValues(t, 300, quota["memory.mb"])

	ram := event["ram"].(common.MapStr)

	ram_quota := ram["quota"].(common.MapStr)

	ram_quota_total := ram_quota["total"].(common.MapStr)
	assert.EqualValues(t, 314572800, ram_quota_total["value.bytes"])
	assert.EqualValues(t, 314572800, ram_quota_total["per_node.bytes"])

	ram_quota_used := ram_quota["used"].(common.MapStr)
	assert.EqualValues(t, 104857600, ram_quota_used["value.bytes"])
	assert.EqualValues(t, 104857600, ram_quota_used["per_node.bytes"])

	assert.EqualValues(t, 8359174144, ram["total.bytes"])

	ram_used := ram["used"].(common.MapStr)
	assert.EqualValues(t, 8004751360, ram_used["value.bytes"])
	assert.EqualValues(t, 53962016, ram_used["by_data.bytes"])
}
