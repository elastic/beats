package exchange

import (
	"testing"

	"github.com/elastic/beats/libbeat/common"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/beats/metricbeat/module/rabbitmq/mtest"

	"github.com/stretchr/testify/assert"
)

func TestFetchEventContents(t *testing.T) {
	server := mtest.Server(t, mtest.DefaultServerConfig)
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
