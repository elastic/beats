package df

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

	response, err := ioutil.ReadFile(absPath + "/sample_response.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "appication/json;")
		w.Write([]byte(response))
	}))
	defer server.Close()

	config := map[string]interface{}{
		"module":     "ceph",
		"metricsets": []string{"df"},
		"hosts":      []string{server.URL},
	}

	f := mbtest.NewEventsFetcher(t, config)
	events, err := f.Fetch()
	event := events[0]
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event.StringToPrint())

	statsCluster := event["stats"].(common.MapStr)

	used := statsCluster["used"].(common.MapStr)
	assert.EqualValues(t, 1428520960, used["bytes"])

	total := statsCluster["total"].(common.MapStr)
	assert.EqualValues(t, 6431965184, total["bytes"])

	available := statsCluster["available"].(common.MapStr)
	assert.EqualValues(t, 5003444224, available["bytes"])

	event = events[1]
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event.StringToPrint())

	pool := event["pool"].(common.MapStr)
	assert.EqualValues(t, "rbd", pool["name"])
	assert.EqualValues(t, 0, pool["id"])

	stats := pool["stats"].(common.MapStr)

	used = stats["used"].(common.MapStr)
	assert.EqualValues(t, 0, used["bytes"])
	assert.EqualValues(t, 0, used["kb"])

	available = stats["available"].(common.MapStr)
	assert.EqualValues(t, 5003444224, available["bytes"])

}
