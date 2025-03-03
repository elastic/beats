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

//go:build !integration

package module_test

import (
	"runtime"
	"testing"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/diagnostics"
	pubtest "github.com/elastic/beats/v7/libbeat/publisher/testing"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/mb/module"
	_ "github.com/elastic/beats/v7/metricbeat/module/system"
	_ "github.com/elastic/beats/v7/metricbeat/module/system/cpu"
	conf "github.com/elastic/elastic-agent-libs/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunner(t *testing.T) {
	pubClient, factory := newPubClientFactory()

	config, err := conf.NewConfigFrom(map[string]interface{}{
		"module":     moduleName,
		"metricsets": []string{reportingFetcherName},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Create a new Wrapper based on the configuration.
	m, err := module.NewWrapper(config, mb.Registry, module.WithMetricSetInfo())
	if err != nil {
		t.Fatal(err)
	}

	// Create the Runner facade.
	runner := module.NewRunner(factory(), m)

	// Start the module and have it publish to a new publisher.Client.
	runner.Start()

	assert.NotNil(t, <-pubClient.Channel)

	// Stop the module. This blocks until all MetricSets in the Module have
	// stopped and the publisher.Client is closed.
	runner.Stop()
}

func TestCPUDiagnostics(t *testing.T) {
	pubClient, factory := newPubClientFactory()

	config, err := conf.NewConfigFrom(map[string]interface{}{
		"module":     "system",
		"metricsets": []string{"cpu"},
	})
	require.NoError(t, err)

	// Create a new Wrapper based on the configuration.
	m, err := module.NewWrapper(config, mb.Registry, module.WithMetricSetInfo())
	if err != nil {
		t.Fatal(err)
	}

	runner := module.NewRunner(factory(), m)

	// First test, run before start. Shouldn't cause panics or other undefined behavior
	diag, ok := runner.(diagnostics.DiagnosticReporter)
	require.True(t, ok)
	diags := diag.Diagnostics()
	// This diagnostic set is only available on linux.
	// On other OSes, the list should be empty
	if runtime.GOOS == "linux" {
		require.NotEmpty(t, diags)
	} else {
		require.Empty(t, diags)
	}

	runner.Start()
	assert.NotNil(t, <-pubClient.Channel)

	diag, ok = runner.(diagnostics.DiagnosticReporter)
	require.True(t, ok)
	diags = diag.Diagnostics()
	if runtime.GOOS == "linux" {
		require.NotEmpty(t, diags)
		res := diags[0].Callback()
		require.NotEmpty(t, res)
	} else {
		require.Empty(t, diags)
	}

	runner.Stop()
	// stop, test again.
	diag, ok = runner.(diagnostics.DiagnosticReporter)
	require.True(t, ok)
	diags = diag.Diagnostics()
	if runtime.GOOS == "linux" {
		require.NotEmpty(t, diags)
	} else {
		require.Empty(t, diags)
	}

}

// newPubClientFactory returns a new ChanClient and a function that returns
// the same Client when invoked. This simulates the return value of
// Publisher.Connect.
func newPubClientFactory() (*pubtest.ChanClient, func() beat.Client) {
	client := pubtest.NewChanClient(10)
	return client, func() beat.Client { return client }
}
