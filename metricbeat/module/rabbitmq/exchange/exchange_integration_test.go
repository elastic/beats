// +build integration

package exchange

import (
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/beats/metricbeat/module/rabbitmq/mtest"
)

func TestData(t *testing.T) {
	compose.EnsureUp(t, "rabbitmq")

	f := mbtest.NewEventsFetcher(t, getConfig())
	err := mbtest.WriteEventsCond(f, t, func(e common.MapStr) bool {
		hasIn, _ := e.HasKey("messages.publish_in")
		hasOut, _ := e.HasKey("messages.publish_out")
		return hasIn && hasOut
	})
	if err != nil {
		t.Fatal("write", err)
	}
}

func getConfig() map[string]interface{} {
	config := mtest.GetIntegrationConfig()
	config["metricsets"] = []string{"exchange"}
	return config
}
