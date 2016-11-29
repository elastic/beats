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

// TestFetchEventContents verifies the contents of the returned event against
// the raw Apache response.
func TestFetchEventContents(t *testing.T) {
	absPath, err := filepath.Abs("./testdata/")
	// response is a raw response copied from an Apache web server.
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

	assert.Equal(t, "membase", event["bucketType"])
	assert.Equal(t, "beer-sample", event["name"])
	assert.Equal(t, 104857600, event["quota_RAM"])
	assert.Equal(t, 104857600, event["quota_RawRAM"])
	assert.Equal(t, 12597731, event["stats_DataUsed"])
	assert.Equal(t, 0, event["stats_DiskFetches"])
	assert.Equal(t, 16369008, event["stats_DiskUsed"])
	assert.Equal(t, 7303, event["stats_ItemCount"])
	assert.Equal(t, 53962160, event["stats_MemUsed"])
	assert.Equal(t, 0, event["stats_OpsPerSec"])
	assert.Equal(t, 51.46232604980469, event["stats_QuotaPercUse"])
}
