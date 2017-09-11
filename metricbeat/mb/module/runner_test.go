// +build !integration

package module_test

import (
	"testing"

	"github.com/elastic/beats/libbeat/common"
	pubtest "github.com/elastic/beats/libbeat/publisher/testing"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/module"

	"github.com/stretchr/testify/assert"
)

func TestRunner(t *testing.T) {
	client := pubtest.NewChanClient(10)

	config, err := common.NewConfigFrom(map[string]interface{}{
		"module":     moduleName,
		"metricsets": []string{eventFetcherName},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Create a new Wrapper based on the configuration.
	m, err := module.NewWrapper(0, config, mb.Registry)
	if err != nil {
		t.Fatal(err)
	}

	// Create the Runner facade.
	connector, err := module.NewConnector(pubtest.NewTestPipeline(client), common.NewConfig())
	if err != nil {
		t.Fatal(err)
	}

	runner := module.NewRunner(connector, m)

	// Start the module and have it publish to a new publisher.Client.
	runner.Start()

	assert.NotNil(t, <-client.Channel)

	// Stop the module. This blocks until all MetricSets in the Module have
	// stopped and the publisher.Client is closed.
	runner.Stop()
}
