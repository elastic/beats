package queue

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

	response, err := ioutil.ReadFile(absPath + "/queue_sample_response.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json;")
		w.Write([]byte(response))
	}))
	defer server.Close()

	config := map[string]interface{}{
		"module":     "rabbitmq",
		"metricsets": []string{"queue"},
		"hosts":      []string{server.URL},
	}

	f := mbtest.NewEventsFetcher(t, config)
	events, err := f.Fetch()
	event := events[0]
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event.StringToPrint())

	assert.EqualValues(t, "queuenamehere", event["name"])
	assert.EqualValues(t, "/", event["vhost"])
	assert.EqualValues(t, true, event["durable"])
	assert.EqualValues(t, false, event["auto_delete"])
	assert.EqualValues(t, false, event["exclusive"])
	assert.EqualValues(t, "running", event["state"])
	assert.EqualValues(t, "rabbit@localhost", event["node"])

	arguments := event["arguments"].(common.MapStr)
	assert.EqualValues(t, 9, arguments["max_priority"])

	consumers := event["consumers"].(common.MapStr)
	utilisation := consumers["utilisation"].(common.MapStr)
	assert.EqualValues(t, 3, consumers["count"])
	assert.EqualValues(t, 0.7, utilisation["pct"])

	memory := event["memory"].(common.MapStr)
	assert.EqualValues(t, 232720, memory["bytes"])

	messages := event["messages"].(common.MapStr)
	total := messages["total"].(common.MapStr)
	ready := messages["ready"].(common.MapStr)
	unacknowledged := messages["unacknowledged"].(common.MapStr)
	persistent := messages["persistent"].(common.MapStr)
	assert.EqualValues(t, 74, total["count"])
	assert.EqualValues(t, 71, ready["count"])
	assert.EqualValues(t, 3, unacknowledged["count"])
	assert.EqualValues(t, 73, persistent["count"])

	totalDetails := total["details"].(common.MapStr)
	assert.EqualValues(t, 2.2, totalDetails["rate"])

	readyDetails := ready["details"].(common.MapStr)
	assert.EqualValues(t, 0, readyDetails["rate"])

	unacknowledgedDetails := unacknowledged["details"].(common.MapStr)
	assert.EqualValues(t, 0.5, unacknowledgedDetails["rate"])

	disk := event["disk"].(common.MapStr)
	reads := disk["reads"].(common.MapStr)
	writes := disk["writes"].(common.MapStr)
	assert.EqualValues(t, 212, reads["count"])
	assert.EqualValues(t, 121, writes["count"])
}
