package osd_df

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
	absPath, err := filepath.Abs("../_meta/testdata/")
	assert.NoError(t, err)

	response, err := ioutil.ReadFile(absPath + "/osd_df_sample_response.json")
	assert.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json;")
		w.Write([]byte(response))
	}))
	defer server.Close()

	config := map[string]interface{}{
		"module":     "ceph",
		"metricsets": []string{"osd_df"},
		"hosts":      []string{server.URL},
	}

	f := mbtest.NewEventsFetcher(t, config)
	events, err := f.Fetch()
	event := events[0]

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event.StringToPrint())

	//check osd0 df info
	nodeInfo := events[0]
	assert.EqualValues(t, 0, nodeInfo["pg_num"])
	assert.EqualValues(t, 52325356, nodeInfo["total.byte"])
	assert.EqualValues(t, 1079496, nodeInfo["used.byte"])
	assert.EqualValues(t, 51245860, nodeInfo["available.byte"])
	assert.EqualValues(t, "hdd", nodeInfo["device_class"])
	assert.EqualValues(t, 0.020630456866839092, nodeInfo["used.pct"])
	assert.EqualValues(t, 0, nodeInfo["id"])
	assert.EqualValues(t, "osd.0", nodeInfo["name"])

	//check osd1 df info
	nodeInfo = events[1]
	assert.EqualValues(t, 0, nodeInfo["pg_num"])
	assert.EqualValues(t, 52325356, nodeInfo["total.byte"])
	assert.EqualValues(t, 1079496, nodeInfo["used.byte"])
	assert.EqualValues(t, 51245860, nodeInfo["available.byte"])
	assert.EqualValues(t, "hdd", nodeInfo["device_class"])
	assert.EqualValues(t, 0.020630456866839092, nodeInfo["used.pct"])
	assert.EqualValues(t, 1, nodeInfo["id"])
	assert.EqualValues(t, "osd.1", nodeInfo["name"])

}
