// +build integration

package libbeat

// This tests use filebeat for the libbeat part
import (
	"testing"

	"github.com/elastic/beats/libbeat/common"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/beats/metricbeat/module/beats/filebeat"
	"github.com/stretchr/testify/assert"
)

func TestFetch(t *testing.T) {
	f := mbtest.NewEventFetcher(t, getConfig())
	event, err := f.Fetch()
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event)

	// Check event fields
	openFiles := event["publisher"].(common.MapStr)["events"].(common.MapStr)["published"].(int64)
	assert.True(t, openFiles >= 0)
}

func TestData(t *testing.T) {
	f := mbtest.NewEventFetcher(t, getConfig())
	err := mbtest.WriteEvent(f, t)
	if err != nil {
		t.Fatal("write", err)
	}
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "beats",
		"metricsets": []string{"libbeat"},
		"hosts":      []string{filebeat.GetEnvHost() + ":" + filebeat.GetEnvPort()},
	}
}
