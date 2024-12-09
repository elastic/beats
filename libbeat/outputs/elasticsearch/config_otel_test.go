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

package elasticsearch

import (
	_ "embed"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/confmap/confmaptest"
	"gopkg.in/yaml.v2"

	"github.com/elastic/elastic-agent-libs/config"
)

//go:embed testdata/basic.yml
var beatYAMLCfg string

func TestToOtelConfig(t *testing.T) {

	tests := []struct {
		name      string
		input     string
		expOutput string
		experr    bool
	}{
		{
			name:      "basic elasticsearch input",
			input:     "basic.yml",
			expOutput: "basicop.yml",
			experr:    false,
		},
		// {
		// 	name:      "when cloud id is provided",
		// 	input:     "unsupported-output.yml",
		// 	expOutput: "",
		// 	experr:    true,
		// },
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			beatCfg := config.MustNewConfigFrom(beatYAMLCfg)

			otelCfg, err := ToOTelConfig(beatCfg)
			require.NoError(t, err, "could not convert beat config to otel ES config")

			expectedValue, err := confmaptest.LoadConf(filepath.Join("testdata", test.expOutput))
			require.NoError(t, err)
			want, err := yaml.Marshal(expectedValue.ToStringMap())
			require.NoError(t, err)

			got, err := yaml.Marshal(otelCfg)
			require.NoError(t, err)

			assert.Equal(t, string(want), string(got))

		})
	}

}
