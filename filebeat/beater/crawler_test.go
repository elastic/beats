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

package beater

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

func TestCrawlerLoggingWithoutDynamicConfig(t *testing.T) {
	// Create a mock input factory
	mockFactory := &mockRunnerFactory{}

	// Create input configs
	inputConfigs := []*conf.C{
		conf.MustNewConfigFrom(map[string]interface{}{
			"type":    "log",
			"enabled": true,
			"paths":   []string{"/var/log/*.log"},
		}),
	}

	// Create crawler
	crawler, err := newCrawler(mockFactory, mockFactory, inputConfigs, make(chan struct{}), false, logp.NewLogger("test"))
	require.NoError(t, err)

	// Start crawler without dynamic config
	disabledConfig := conf.MustNewConfigFrom(map[string]interface{}{
		"enabled": false,
	})

	// Create a mock pipeline
	pipeline := &mockPipelineConnector{}

	err = crawler.Start(pipeline, disabledConfig, disabledConfig)
	require.NoError(t, err)

	// Verify that c.inputs has entries after starting
	assert.Equal(t, 1, len(crawler.inputs))
}

func TestCrawlerLoggingWithDynamicConfig(t *testing.T) {
	// Create a mock input factory
	mockFactory := &mockRunnerFactory{}

	// Create empty input configs (simulating managed mode)
	inputConfigs := []*conf.C{}

	// Create crawler
	crawler, err := newCrawler(mockFactory, mockFactory, inputConfigs, make(chan struct{}), false, logp.NewLogger("test"))
	require.NoError(t, err)

	// Start crawler with dynamic config enabled
	dynamicConfig := conf.MustNewConfigFrom(map[string]interface{}{
		"enabled": true,
		"path":    "/tmp/config.d/*.yml",
		"reload": map[string]interface{}{
			"enabled": true,
			"period":  "10s",
		},
	})

	disabledConfig := conf.MustNewConfigFrom(map[string]interface{}{
		"enabled": false,
	})

	// Create a mock pipeline
	pipeline := &mockPipelineConnector{}

	err = crawler.Start(pipeline, dynamicConfig, disabledConfig)
	require.NoError(t, err)

	// Verify that c.inputs is empty initially (no static inputs)
	assert.Equal(t, 0, len(crawler.inputs))

	// Verify that inputReloader is set up
	assert.NotNil(t, crawler.inputReloader)

	// Stop the crawler
	crawler.Stop()
}

// Mock implementations for testing

type mockRunnerFactory struct{}

func (m *mockRunnerFactory) Create(pipeline beat.PipelineConnector, config *conf.C) (cfgfile.Runner, error) {
	return &mockRunner{}, nil
}

func (m *mockRunnerFactory) CheckConfig(config *conf.C) error {
	return nil
}

type mockRunner struct{}

func (m *mockRunner) Start() {}
func (m *mockRunner) Stop()  {}
func (m *mockRunner) String() string {
	return "mock runner"
}

type mockPipelineConnector struct{}

func (m *mockPipelineConnector) Connect() (beat.Client, error) {
	return nil, nil
}

func (m *mockPipelineConnector) ConnectWith(beat.ClientConfig) (beat.Client, error) {
	return nil, nil
}