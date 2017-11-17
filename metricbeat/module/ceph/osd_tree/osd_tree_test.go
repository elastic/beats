package osd_tree

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

	response, err := ioutil.ReadFile(absPath + "/osd_tree_sample_response.json")
	assert.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json;")
		w.Write([]byte(response))
	}))
	defer server.Close()

	config := map[string]interface{}{
		"module":     "ceph",
		"metricsets": []string{"osd_tree"},
		"hosts":      []string{server.URL},
	}

	f := mbtest.NewEventsFetcher(t, config)
	events, err := f.Fetch()
	event := events[0]

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event.StringToPrint())

	//check root bucket info
	nodeInfo := events[0]
	assert.EqualValues(t, "default", nodeInfo["name"])
	assert.EqualValues(t, "root", nodeInfo["type"])
	assert.EqualValues(t, "-3", nodeInfo["children"])
	assert.EqualValues(t, -1, nodeInfo["id"])
	assert.EqualValues(t, 10, nodeInfo["type_id"])
	assert.EqualValues(t, "", nodeInfo["father"])

	//check host bucket info
	nodeInfo = events[1]
	assert.EqualValues(t, "ceph-mon1", nodeInfo["name"])
	assert.EqualValues(t, "host", nodeInfo["type"])
	assert.EqualValues(t, "1,0", nodeInfo["children"])
	assert.EqualValues(t, -3, nodeInfo["id"])
	assert.EqualValues(t, 1, nodeInfo["type_id"])
	assert.EqualValues(t, "default", nodeInfo["father"])

	//check osd bucket info
	nodeInfo = events[2]
	assert.EqualValues(t, "up", nodeInfo["status"])
	assert.EqualValues(t, "osd.0", nodeInfo["name"])
	assert.EqualValues(t, "osd", nodeInfo["type"])
	assert.EqualValues(t, 1, nodeInfo["primary_affinity"])
	assert.EqualValues(t, true, nodeInfo["exists"])
	assert.EqualValues(t, 0, nodeInfo["id"])
	assert.EqualValues(t, 0, nodeInfo["type_id"])
	assert.EqualValues(t, 0.048691, nodeInfo["crush_weight"])
	assert.EqualValues(t, "hdd", nodeInfo["device_class"])
	assert.EqualValues(t, 1, nodeInfo["reweight"])
	assert.EqualValues(t, "ceph-mon1", nodeInfo["father"])
	assert.EqualValues(t, 2, nodeInfo["depth"])

	nodeInfo = events[3]
	assert.EqualValues(t, "up", nodeInfo["status"])
	assert.EqualValues(t, "osd.1", nodeInfo["name"])
	assert.EqualValues(t, "osd", nodeInfo["type"])
	assert.EqualValues(t, 1, nodeInfo["primary_affinity"])
	assert.EqualValues(t, true, nodeInfo["exists"])
	assert.EqualValues(t, 1, nodeInfo["id"])
	assert.EqualValues(t, 0, nodeInfo["type_id"])
	assert.EqualValues(t, 0.048691, nodeInfo["crush_weight"])
	assert.EqualValues(t, "hdd", nodeInfo["device_class"])
	assert.EqualValues(t, 1, nodeInfo["reweight"])
	assert.EqualValues(t, "ceph-mon1", nodeInfo["father"])
	assert.EqualValues(t, 2, nodeInfo["depth"])

}
