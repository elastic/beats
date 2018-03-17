package exchange

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

	response, err := ioutil.ReadFile(absPath + "/exchange_sample_response.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json;")
		w.Write([]byte(response))
	}))
	defer server.Close()

	config := map[string]interface{}{
		"module":     "rabbitmq",
		"metricsets": []string{"exchange"},
		"hosts":      []string{server.URL},
	}

	f := mbtest.NewEventsFetcher(t, config)
	events, err := f.Fetch()
	event := events[0]
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event.StringToPrint())

	assert.EqualValues(t, "exchange.name", event["name"])
	assert.EqualValues(t, "/", event["vhost"])
	assert.EqualValues(t, true, event["durable"])
	assert.EqualValues(t, false, event["auto_delete"])
	assert.EqualValues(t, false, event["internal"])

	messages := event["messages"].(common.MapStr)
	publish := messages["publish"].(common.MapStr)
	publishIn := messages["publish_in"].(common.MapStr)
	publishOut := messages["publish_out"].(common.MapStr)
	ack := messages["ack"].(common.MapStr)
	deliverGet := messages["deliver_get"].(common.MapStr)
	confirm := messages["confirm"].(common.MapStr)
	returnUnroutable := messages["return_unroutable"].(common.MapStr)
	redeliver := messages["redeliver"].(common.MapStr)

	assert.EqualValues(t, 123, publish["count"])
	assert.EqualValues(t, 100, publishIn["count"])
	assert.EqualValues(t, 99, publishOut["count"])
	assert.EqualValues(t, 60, ack["count"])
	assert.EqualValues(t, 50, deliverGet["count"])
	assert.EqualValues(t, 120, confirm["count"])
	assert.EqualValues(t, 40, returnUnroutable["count"])
	assert.EqualValues(t, 30, redeliver["count"])

	publishDetails := publish["details"].(common.MapStr)
	assert.EqualValues(t, 0.1, publishDetails["rate"])

	publishInDetails := publishIn["details"].(common.MapStr)
	assert.EqualValues(t, 0.5, publishInDetails["rate"])

	publishOutDetails := publishOut["details"].(common.MapStr)
	assert.EqualValues(t, 0.9, publishOutDetails["rate"])

	ackDetails := ack["details"].(common.MapStr)
	assert.EqualValues(t, 12.5, ackDetails["rate"])

	deliverGetDetails := deliverGet["details"].(common.MapStr)
	assert.EqualValues(t, 43.21, deliverGetDetails["rate"])

	confirmDetails := confirm["details"].(common.MapStr)
	assert.EqualValues(t, 98.63, confirmDetails["rate"])

	returnUnroutableDetails := returnUnroutable["details"].(common.MapStr)
	assert.EqualValues(t, 123, returnUnroutableDetails["rate"])

	redeliverDetails := redeliver["details"].(common.MapStr)
	assert.EqualValues(t, 0, redeliverDetails["rate"])

}
