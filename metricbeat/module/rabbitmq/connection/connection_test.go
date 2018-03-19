package connection

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

	response, err := ioutil.ReadFile(absPath + "/connection_sample_response.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json;")
		w.Write([]byte(response))
	}))
	defer server.Close()

	config := map[string]interface{}{
		"module":     "rabbitmq",
		"metricsets": []string{"connection"},
		"hosts":      []string{server.URL},
	}

	f := mbtest.NewEventsFetcher(t, config)
	events, err := f.Fetch()
	event := events[0]
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event.StringToPrint())

	assert.EqualValues(t, "[::1]:60938 -> [::1]:5672", event["name"])
	assert.EqualValues(t, "/", event["vhost"])
	assert.EqualValues(t, "guest", event["user"])
	assert.EqualValues(t, "nodename", event["node"])
	assert.EqualValues(t, 8, event["channels"])
	assert.EqualValues(t, 65535, event["channel_max"])
	assert.EqualValues(t, 131072, event["frame_max"])
	assert.EqualValues(t, "network", event["type"])

	packetCount := event["packet_count"].(common.MapStr)
	assert.EqualValues(t, 376, packetCount["sent"])
	assert.EqualValues(t, 376, packetCount["received"])
	assert.EqualValues(t, 0, packetCount["pending"])

	octetCount := event["octet_count"].(common.MapStr)
	assert.EqualValues(t, 3840, octetCount["sent"])
	assert.EqualValues(t, 3764, octetCount["received"])

	assert.EqualValues(t, "::1", event["host"])
	assert.EqualValues(t, 5672, event["port"])

	peer := event["peer"].(common.MapStr)
	assert.EqualValues(t, "::1", peer["host"])
	assert.EqualValues(t, 60938, peer["port"])
}
