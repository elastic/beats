// +build !integration

package node

import (
	"net/http"
	"net/http/httptest"
	"testing"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"

	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"path/filepath"
)

// TestFetchEventContents verifies the contents of the returned event against
// the raw Apache response.
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
	assert.EqualValues(t, 8359174144, event["memoryTotal"])
	assert.EqualValues(t, 4678324224, event["memoryFree"])
	assert.EqualValues(t, 6377, event["mcdMemoryReserved"])
	assert.EqualValues(t, 6377, event["mcdMemoryAllocated"])
	assert.EqualValues(t, 0, event["cmdGet"])
	assert.EqualValues(t, 13563791, event["couchDocsActualDiskSize"])
	assert.EqualValues(t, 9792512, event["couchDocsDataSize"])
	assert.EqualValues(t, 0, event["couchSpatialDataSize"])
	assert.EqualValues(t, 0, event["couchSpatialDiskSize"])
	assert.EqualValues(t, 2805219, event["couchViewsActualDiskSize"])
	assert.EqualValues(t, 2805219, event["couchViewsDataSize"])
	assert.EqualValues(t, 7303, event["currItems"])
	assert.EqualValues(t, 7303, event["currItemsTot"])
	assert.EqualValues(t, 0, event["epBgFetched"])
	assert.EqualValues(t, 0, event["getHits"])
	assert.EqualValues(t, 53962016, event["memUsed"])
	assert.EqualValues(t, 0, event["ops"])
	assert.EqualValues(t, 0, event["vbReplicaCurrItems"])
	assert.EqualValues(t, 29.64705882352941, event["CPUUtilizationRate"])
	assert.EqualValues(t, 4189057024, event["swapTotal"])
	assert.EqualValues(t, 135168, event["swapUsed"])
	assert.EqualValues(t, 8359174144, event["memTotal"])
	assert.EqualValues(t, 4678324224, event["memFree"])
}
