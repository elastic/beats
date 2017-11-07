// +build !integration

package node

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/elastic/beats/libbeat/common"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"

	"github.com/stretchr/testify/assert"
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
		"metricsets": []string{"node"},
		"hosts":      []string{server.URL},
	}

	f := mbtest.NewEventsFetcher(t, config)
	events, err := f.Fetch()
	event := events[0]
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event.StringToPrint())

	assert.EqualValues(t, 0, event["cmd_get"])

	couch := event["couch"].(common.MapStr)

	couch_docs := couch["docs"].(common.MapStr)
	couch_docs_disk_size := couch_docs["disk_size"].(common.MapStr)
	assert.EqualValues(t, 13563791, couch_docs_disk_size["bytes"])

	couch_docs_data_size := couch_docs["data_size"].(common.MapStr)
	assert.EqualValues(t, 9792512, couch_docs_data_size["bytes"])

	couch_spatial := couch["spatial"].(common.MapStr)
	couch_spatial_data_size := couch_spatial["data_size"].(common.MapStr)
	assert.EqualValues(t, 0, couch_spatial_data_size["bytes"])

	couch_spatial_disk_size := couch_spatial["disk_size"].(common.MapStr)
	assert.EqualValues(t, 0, couch_spatial_disk_size["bytes"])

	couch_views := couch["views"].(common.MapStr)
	couch_views_disk_size := couch_views["disk_size"].(common.MapStr)
	assert.EqualValues(t, 2805219, couch_views_disk_size["bytes"])

	couch_views_data_size := couch_views["data_size"].(common.MapStr)
	assert.EqualValues(t, 2805219, couch_views_data_size["bytes"])

	event_cpu_utilization_rate := event["cpu_utilization_rate"].(common.MapStr)
	assert.EqualValues(t, 29.64705882352941, event_cpu_utilization_rate["pct"])

	current_items := event["current_items"].(common.MapStr)
	assert.EqualValues(t, 7303, current_items["value"])
	assert.EqualValues(t, 7303, current_items["total"])

	assert.EqualValues(t, 0, event["ep_bg_fetched"])
	assert.EqualValues(t, 0, event["get_hits"])
	assert.Equal(t, "172.17.0.2:8091", event["hostname"])

	mcd_memory := event["mcd_memory"].(common.MapStr)
	mcd_memory_reserved := mcd_memory["reserved"].(common.MapStr)
	assert.EqualValues(t, 6377, mcd_memory_reserved["bytes"])

	mcd_memory_allocated := mcd_memory["allocated"].(common.MapStr)
	assert.EqualValues(t, 6377, mcd_memory_allocated["bytes"])

	memory := event["memory"].(common.MapStr)
	memory_total := memory["total"].(common.MapStr)
	assert.EqualValues(t, 8359174144, memory_total["bytes"])

	memory_free := memory["free"].(common.MapStr)
	assert.EqualValues(t, 4678324224, memory_free["bytes"])

	memory_used := memory["used"].(common.MapStr)
	assert.EqualValues(t, 53962016, memory_used["bytes"])

	assert.EqualValues(t, 0, event["ops"])

	swap := event["swap"].(common.MapStr)
	swap_total := swap["total"].(common.MapStr)
	assert.EqualValues(t, 4189057024, swap_total["bytes"])

	swap_used := swap["used"].(common.MapStr)
	assert.EqualValues(t, 135168, swap_used["bytes"])

	event_uptime := event["uptime"].(common.MapStr)
	assert.EqualValues(t, 7260, event_uptime["sec"])

	assert.EqualValues(t, 0, event["vb_replica_curr_items"])
}
