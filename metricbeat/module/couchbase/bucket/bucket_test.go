// +build !integration

package bucket

import (
	"net/http"
	"net/http/httptest"
	"testing"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"

	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"path/filepath"
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
		"metricsets": []string{"bucket"},
		"hosts":      []string{server.URL},
	}

	f := mbtest.NewEventsFetcher(t, config)
	events, err := f.Fetch()
	event := events[0]
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event.StringToPrint())

	assert.EqualValues(t, "membase", event["type"])
	assert.EqualValues(t, "beer-sample", event["name"])
	assert.EqualValues(t, 104857600, event["quota.ram"])
	assert.EqualValues(t, 104857600, event["quota.raw_ram"])
	assert.EqualValues(t, 51.46232604980469, event["quota.use.pct"])
	assert.EqualValues(t, 12597731, event["data_used"])
	assert.EqualValues(t, 0, event["disk.fetches"])
	assert.EqualValues(t, 16369008, event["disk.used"])
	assert.EqualValues(t, 7303, event["item_count"])
	assert.EqualValues(t, 53962160, event["mem_used"])
	assert.EqualValues(t, 0, event["ops_per_sec"])
}
