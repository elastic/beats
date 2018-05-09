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

	messagesExpected := common.MapStr{
		"publish_in": common.MapStr{
			"count":   int64(100),
			"details": common.MapStr{"rate": float64(0.5)},
		},
		"publish_out": common.MapStr{
			"count":   int64(99),
			"details": common.MapStr{"rate": float64(0.9)},
		},
	}

	assert.Equal(t, "exchange.name", event["name"])
	assert.Equal(t, "guest", event["user"])
	assert.Equal(t, "/", event["vhost"])
	assert.Equal(t, true, event["durable"])
	assert.Equal(t, false, event["auto_delete"])
	assert.Equal(t, false, event["internal"])
	assert.Equal(t, messagesExpected, event["messages"])
}
