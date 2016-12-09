// +build !integration

package node

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

	assert.Equal(t, "172.17.0.2:8091", event["hostname"])
	assert.Equal(t, "7260", event["uptime"])
	assert.EqualValues(t, 8359174144, event["mem.total.bytes"])
	assert.EqualValues(t, 4678324224, event["mem.free.bytes"])
	assert.EqualValues(t, 6377, event["mcd_memory.reserved.bytes"])
	assert.EqualValues(t, 6377, event["mcd_memory.allocated.bytes"])
	assert.EqualValues(t, 0, event["cmd_get"])
	assert.EqualValues(t, 13563791, event["couch_docs_actual_disk_size.bytes"])
	assert.EqualValues(t, 9792512, event["couch_docs_data_size.bytes"])
	assert.EqualValues(t, 0, event["couch_spatial_data_size.bytes"])
	assert.EqualValues(t, 0, event["couch_spatial_disk_size.bytes"])
	assert.EqualValues(t, 2805219, event["couch_views_actual_disk_size.bytes"])
	assert.EqualValues(t, 2805219, event["couch_views_data_size.bytes"])
	assert.EqualValues(t, 7303, event["curr_items"])
	assert.EqualValues(t, 7303, event["curr_items_tot"])
	assert.EqualValues(t, 0, event["ep_bg_fetched"])
	assert.EqualValues(t, 0, event["get_hits"])
	assert.EqualValues(t, 53962016, event["mem.used.bytes"])
	assert.EqualValues(t, 0, event["ops"])
	assert.EqualValues(t, 0, event["vb_replica_curr_items"])
	assert.EqualValues(t, 29.64705882352941, event["cpu_utilization_rate.pct"])
	assert.EqualValues(t, 4189057024, event["swap.total.bytes"])
	assert.EqualValues(t, 135168, event["swap.used.bytes"])
}
