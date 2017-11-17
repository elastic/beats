// +build !integration

package cluster

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
)

func TestFetchEventContents(t *testing.T) {
	absPath, err := filepath.Abs("./testdata/")
	// response is a raw response from a couchbase
	response, err := ioutil.ReadFile(absPath + "/sample_response.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json;")
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
	hdd_free := hdd["free"].(common.MapStr)
	assert.EqualValues(t, 46902679716, hdd_free["bytes"])

	hdd_total := hdd["total"].(common.MapStr)
	assert.EqualValues(t, 63381999616, hdd_total["bytes"])

	hdd_used := hdd["used"].(common.MapStr)
	hdd_used_value := hdd_used["value"].(common.MapStr)
	assert.EqualValues(t, 16479319900, hdd_used_value["bytes"])

	hdd_used_by_data := hdd_used["by_data"].(common.MapStr)
	assert.EqualValues(t, 16369010, hdd_used_by_data["bytes"])

	hdd_quota := hdd["quota"].(common.MapStr)
	hdd_quota_total := hdd_quota["total"].(common.MapStr)
	assert.EqualValues(t, 63381999616, hdd_quota_total["bytes"])

	assert.EqualValues(t, 10, event["max_bucket_count"])

	quota := event["quota"].(common.MapStr)
	quota_index_memory := quota["index_memory"].(common.MapStr)
	assert.EqualValues(t, 300, quota_index_memory["mb"])

	quota_memory := quota["memory"].(common.MapStr)
	assert.EqualValues(t, 300, quota_memory["mb"])

	ram := event["ram"].(common.MapStr)

	ram_quota := ram["quota"].(common.MapStr)

	ram_quota_total := ram_quota["total"].(common.MapStr)
	ram_quota_total_value := ram_quota_total["value"].(common.MapStr)
	assert.EqualValues(t, 314572800, ram_quota_total_value["bytes"])

	ram_quota_total_per_node := ram_quota_total["per_node"].(common.MapStr)
	assert.EqualValues(t, 314572800, ram_quota_total_per_node["bytes"])

	ram_quota_used := ram_quota["used"].(common.MapStr)
	ram_quota_used_value := ram_quota_used["value"].(common.MapStr)
	assert.EqualValues(t, 104857600, ram_quota_used_value["bytes"])

	ram_quota_used_per_node := ram_quota_used["per_node"].(common.MapStr)
	assert.EqualValues(t, 104857600, ram_quota_used_per_node["bytes"])

	ram_total := ram["total"].(common.MapStr)
	assert.EqualValues(t, 8359174144, ram_total["bytes"])

	ram_used := ram["used"].(common.MapStr)
	ram_used_value := ram_used["value"].(common.MapStr)
	assert.EqualValues(t, 8004751360, ram_used_value["bytes"])

	ram_used_by_data := ram_used["by_data"].(common.MapStr)
	assert.EqualValues(t, 53962016, ram_used_by_data["bytes"])
}
