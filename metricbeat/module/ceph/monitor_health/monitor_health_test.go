package monitor_health

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
	absPath, err := filepath.Abs("../_meta/testdata/")

	response, err := ioutil.ReadFile(absPath + "/sample_response.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json;")
		w.Write([]byte(response))
	}))
	defer server.Close()

	config := map[string]interface{}{
		"module":     "ceph",
		"metricsets": []string{"monitor_health"},
		"hosts":      []string{server.URL},
	}

	f := mbtest.NewEventsFetcher(t, config)
	events, err := f.Fetch()
	if err != nil {
		t.Fatal(err)
	}
	event := events[0]

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event.StringToPrint())

	mon := event
	assert.EqualValues(t, "HEALTH_OK", mon["health"])
	assert.EqualValues(t, "ceph", mon["name"])
	assert.EqualValues(t, "2017-01-19 11:34:50.700723 +0000 UTC", mon["last_updated"].(Tick).Time.String())

	available := mon["available"].(common.MapStr)
	assert.EqualValues(t, 4091244, available["kb"])
	assert.EqualValues(t, 65, available["pct"])

	total := mon["total"].(common.MapStr)
	assert.EqualValues(t, 6281216, total["kb"])

	used := mon["used"].(common.MapStr)
	assert.EqualValues(t, 2189972, used["kb"])

	store_stats := mon["store_stats"].(common.MapStr)
	assert.EqualValues(t, "0.000000", store_stats["last_updated"])

	misc := store_stats["misc"].(common.MapStr)
	assert.EqualValues(t, 840, misc["bytes"])

	log := store_stats["log"].(common.MapStr)
	assert.EqualValues(t, 8488103, log["bytes"])

	sst := store_stats["sst"].(common.MapStr)
	assert.EqualValues(t, 0, sst["bytes"])

	total = store_stats["total"].(common.MapStr)
	assert.EqualValues(t, 8488943, total["bytes"])
}
