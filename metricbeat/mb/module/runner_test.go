// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

// +build !integration

package module_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	pubtest "github.com/elastic/beats/libbeat/publisher/testing"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/module"
)

func TestRunnerForStaticModule(t *testing.T) {
	pubClient, factory := newPubClientFactoryForStaticModule()

	config, err := common.NewConfigFrom(map[string]interface{}{
		"module":     moduleName,
		"metricsets": []string{eventFetcherName},
	})
	require.NoError(t, err)

	// Create a new Wrapper based on the configuration.
	m, err := module.NewWrapper(config, mb.Registry, module.WithMetricSetInfo())
	require.NoError(t, err)

	// Create the Runner facade.
	runner := module.NewRunnerForStaticModule(factory(), m)

	// Start the module and have it publish to a new publisher.Client.
	runner.Start()

	assert.NotNil(t, <-pubClient.Channel)

	// Stop the module. This blocks until all MetricSets in the Module have
	// stopped and the publisher.Client is closed.
	runner.Stop()
}

// newPubClientFactoryForStaticModule returns a new ChanClient and a function that returns
// the same Client when invoked. This simulates the return value of
// Publisher.Connect.
func newPubClientFactoryForStaticModule() (*pubtest.ChanClient, func() beat.Client) {
	client := pubtest.NewChanClient(10)
	return client, func() beat.Client { return client }
}

func TestRunner(t *testing.T) {
	pubClients, factory := newPubClientFactory()

	config, err := common.NewConfigFrom(map[string]interface{}{
		"module":     moduleName,
		"metricsets": []string{eventFetcherName, reportingFetcherName},
	})
	require.NoError(t, err)

	// Create a new Wrapper based on the configuration.
	m, err := module.NewWrapper(config, mb.Registry, module.WithMetricSetInfo())
	require.NoError(t, err)

	// Create the Runner facade.
	runner := module.NewRunner(factory(), m)

	// Start the module and have it publish to a new publisher.Client.
	runner.Start()

	assert.NotNil(t, <-pubClients[0].Channel)
	assert.NotNil(t, <-pubClients[1].Channel)

	// Stop the module. This blocks until all MetricSets in the Module have
	// stopped and the publisher.Client is closed.
	runner.Stop()
}

// newPubClientFactory returns new ChanClients and a function that returns
// the same Clients when invoked. This simulates the return value of
// Publisher.Connect.
func newPubClientFactory() ([]*pubtest.ChanClient, func() map[string]beat.Client) {
	firstClient := pubtest.NewChanClient(10)
	secondClient := pubtest.NewChanClient(10)
	return []*pubtest.ChanClient{firstClient, secondClient}, func() map[string]beat.Client {
		return map[string]beat.Client{
			strings.ToLower(eventFetcherName):     firstClient,
			strings.ToLower(reportingFetcherName): secondClient,
		}
	}
}
