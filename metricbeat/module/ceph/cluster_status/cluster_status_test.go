package cluster_status

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
	assert.NoError(t, err)

	response, err := ioutil.ReadFile(absPath + "/status_sample_response.json")
	assert.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json;")
		w.Write([]byte(response))
	}))
	defer server.Close()

	config := map[string]interface{}{
		"module":     "ceph",
		"metricsets": []string{"cluster_status"},
		"hosts":      []string{server.URL},
	}

	f := mbtest.NewEventsFetcher(t, config)
	events, err := f.Fetch()
	event := events[0]

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event.StringToPrint())

	//check status version number
	assert.EqualValues(t, 813, event["version"])

	//check osd info
	osdmap := event["osd"].(common.MapStr)
	assert.EqualValues(t, false, osdmap["full"])
	assert.EqualValues(t, false, osdmap["nearfull"])
	assert.EqualValues(t, 6, osdmap["osd_count"])
	assert.EqualValues(t, 3, osdmap["up_osd_count"])
	assert.EqualValues(t, 4, osdmap["in_osd_count"])
	assert.EqualValues(t, 240, osdmap["remapped_pg_count"])
	assert.EqualValues(t, 264, osdmap["epoch"])

	//check traffic info
	trafficInfo := event["traffic"].(common.MapStr)
	assert.EqualValues(t, 55667788, trafficInfo["read_bytes"])
	assert.EqualValues(t, 1234, trafficInfo["read_op_per_sec"])
	assert.EqualValues(t, 11996158, trafficInfo["write_bytes"])
	assert.EqualValues(t, 10, trafficInfo["write_op_per_sec"])

	//check misplace info
	misplaceInfo := event["misplace"].(common.MapStr)
	assert.EqualValues(t, 768, misplaceInfo["total"])
	assert.EqualValues(t, 88, misplaceInfo["objects"])
	assert.EqualValues(t, 0.114583, misplaceInfo["pct"])

	//check degraded info
	degradedInfo := event["degraded"].(common.MapStr)
	assert.EqualValues(t, 768, degradedInfo["total"])
	assert.EqualValues(t, 294, degradedInfo["objects"])
	assert.EqualValues(t, 0.382812, degradedInfo["pct"])

	//check pg info
	pgInfo := event["pg"].(common.MapStr)
	assert.EqualValues(t, 1054023794, pgInfo["data_bytes"])
	assert.EqualValues(t, 9965821952, pgInfo["avail_bytes"])
	assert.EqualValues(t, 12838682624, pgInfo["total_bytes"])
	assert.EqualValues(t, 2872860672, pgInfo["used_bytes"])

	//check pg_state info
	pg_stateInfo := events[1]["pg_state"].(common.MapStr)
	assert.EqualValues(t, "active+undersized+degraded", pg_stateInfo["state_name"])
	assert.EqualValues(t, 109, pg_stateInfo["count"])
	assert.EqualValues(t, 813, pg_stateInfo["version"])

	pg_stateInfo = events[2]["pg_state"].(common.MapStr)
	assert.EqualValues(t, "undersized+degraded+peered", pg_stateInfo["state_name"])
	assert.EqualValues(t, 101, pg_stateInfo["count"])
	assert.EqualValues(t, 813, pg_stateInfo["version"])

	pg_stateInfo = events[3]["pg_state"].(common.MapStr)
	assert.EqualValues(t, "active+remapped", pg_stateInfo["state_name"])
	assert.EqualValues(t, 55, pg_stateInfo["count"])
	assert.EqualValues(t, 813, pg_stateInfo["version"])

	pg_stateInfo = events[4]["pg_state"].(common.MapStr)
	assert.EqualValues(t, "active+undersized+degraded+remapped", pg_stateInfo["state_name"])
	assert.EqualValues(t, 55, pg_stateInfo["count"])
	assert.EqualValues(t, 813, pg_stateInfo["version"])
}
