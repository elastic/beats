// +build !integration

package beater_test

import (
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/publisher"
	pubtest "github.com/elastic/beats/libbeat/publisher/testing"
	metricbeat "github.com/elastic/beats/metricbeat/beater"
	"github.com/elastic/beats/metricbeat/mb"

	"github.com/stretchr/testify/assert"
)

func TestModuleRunner(t *testing.T) {
	pubClient, factory := newPubClientFactory()

	config, err := common.NewConfigFrom(map[string]interface{}{
		"module":     moduleName,
		"metricsets": []string{metricSetName},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Create a new ModuleWrapper based on the configuration.
	module, err := metricbeat.NewModuleWrapper(config, mb.Registry)
	if err != nil {
		t.Fatal(err)
	}

	// Create the ModuleRunner facade.
	runner := metricbeat.NewModuleRunner(factory, module)

	// Start the module and have it publish to a new publisher.Client.
	runner.Start()

	assert.NotNil(t, <-pubClient.Channel)

	// Stop the module. This blocks until all MetricSets in the Module have
	// stopped and the publisher.Client is closed.
	runner.Stop()
}

// newPubClientFactory returns a new ChanClient and a function that returns
// the same Client when invoked. This simulates the return value of
// Publisher.Connect.
func newPubClientFactory() (*pubtest.ChanClient, func() publisher.Client) {
	client := pubtest.NewChanClient(10)
	return client, func() publisher.Client { return client }
}
