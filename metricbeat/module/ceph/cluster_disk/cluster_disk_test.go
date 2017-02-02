package cluster_disk

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
	assert.NoError(t, err)

	response, err := ioutil.ReadFile(absPath + "/sample_response.json")
	assert.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "appication/json;")
		w.Write([]byte(response))
	}))
	defer server.Close()

	config := map[string]interface{}{
		"module":     "ceph",
		"metricsets": []string{"cluster_disk"},
		"hosts":      []string{server.URL},
	}

	f := mbtest.NewEventFetcher(t, config)
	event, err := f.Fetch()

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event.StringToPrint())

	statsCluster := event["stats"].(common.MapStr)

	used := statsCluster["used"].(common.MapStr)
	assert.EqualValues(t, 1428520960, used["bytes"])

	total := statsCluster["total"].(common.MapStr)
	assert.EqualValues(t, 6431965184, total["bytes"])

	available := statsCluster["available"].(common.MapStr)
	assert.EqualValues(t, 5003444224, available["bytes"])
}
