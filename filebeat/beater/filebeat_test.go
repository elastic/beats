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
	conf "github.com/elastic/beats/v7/pkg/config"
	"github.com/elastic/beats/v7/pkg/logp/logptest"
	"github.com/elastic/beats/v7/pkg/paths"
)

func TestNewModuleLogs(t *testing.T) {
	tests := []struct {
		name string
		opts []Option
		want bool // Whether the module and pipeline lines are logged.
	}{
		{name: "default", want: true},
		{name: "without modules", opts: []Option{WithoutModules()}, want: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// The home directory has no module directory and the output is
			// not Elasticsearch.
			home := t.TempDir()
			logger, logs := logptest.NewTestingLoggerWithObserver(t, "")
			b := &beat.Beat{
				Info: beat.Info{
					Logger: logger,
					Paths:  &paths.Path{Home: home, Config: home, Data: home, Logs: home},
				},
				Config:     &beat.BeatConfig{},
				BeatConfig: conf.NewConfig(),
				InSetupCmd: true,
			}
			_, err := New(nil, tc.opts...)(b, conf.NewConfig())
			require.NoError(t, err, "creating the beater failed")

			gotModules := logs.FilterMessageSnippet("Not loading modules").Len() != 0
			assert.Equal(t, tc.want, gotModules, "unexpected presence of the module directory error")
			gotPipelines := logs.FilterMessageSnippet("unable to load the ingest pipelines for the configured modules").Len() != 0
			assert.Equal(t, tc.want, gotPipelines, "unexpected presence of the pipelines warning")
		})
	}
}
