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

package module

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/metricbeat/mb"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/paths"
)

const (
	pathAwareModuleName    = "path-aware"
	pathAwareMetricsetName = "test"
)

type pathAwareMetricSet struct {
	mb.BaseMetricSet
	received []*paths.Path
}

func (m *pathAwareMetricSet) Fetch(mb.ReporterV2) error {
	return nil
}

func (m *pathAwareMetricSet) SetPaths(p *paths.Path) error {
	m.received = append(m.received, p)
	return nil
}

func newPathAwareMetricSetFactory(msP **pathAwareMetricSet) mb.MetricSetFactory {
	return func(base mb.BaseMetricSet) (mb.MetricSet, error) {
		ms := &pathAwareMetricSet{BaseMetricSet: base}
		*msP = ms
		return ms, nil
	}
}

type (
	testClient   struct{}
	testPipeline struct{}
)

func (*testClient) Close() error                                        { return nil }
func (*testClient) Publish(beat.Event)                                  {}
func (*testClient) PublishAll([]beat.Event)                             {}
func (testPipeline) Connect() (beat.Client, error)                      { return &testClient{}, nil }
func (testPipeline) ConnectWith(beat.ClientConfig) (beat.Client, error) { return &testClient{}, nil }

func TestFactorySetsMetricSetPath(t *testing.T) {
	tests := []struct {
		name  string
		paths *paths.Path
	}{
		{
			name:  "with custom paths",
			paths: &paths.Path{Home: "/tmp/home", Config: "/tmp/config", Data: "/tmp/data", Logs: "/tmp/logs"},
		},
		{
			name:  "with nil paths",
			paths: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ms *pathAwareMetricSet

			reg := mb.NewRegister()
			err := reg.AddMetricSet(pathAwareModuleName, pathAwareMetricsetName, newPathAwareMetricSetFactory(&ms))
			require.NoError(t, err)
			cfg, err := conf.NewConfigFrom(map[string]any{
				"module":     pathAwareModuleName,
				"metricsets": []string{pathAwareMetricsetName},
				"hosts":      []string{"example.net"},
			})
			require.NoError(t, err)

			factory := NewFactory(beat.Info{
				Beat:   "metricbeat",
				Logger: logp.NewNopLogger(),
			}, beat.NewMonitoring(), tt.paths, reg)

			assert.Nil(t, ms, "Not called yet")
			runner, err := factory.Create(&testPipeline{}, cfg)
			require.NoError(t, err)
			require.NotNil(t, runner)
			t.Cleanup(runner.Stop)

			require.Len(t, ms.received, 1)
			assert.Same(t, tt.paths, ms.received[0])
		})
	}
}
