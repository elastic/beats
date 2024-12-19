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
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/confmap/confmaptest"
	"gopkg.in/yaml.v2"

	"github.com/elastic/elastic-agent-libs/config"
)

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
			expOutput: "basic-op.yml",
			experr:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rawConf, err := loadBeatConf(filepath.Join("testdata", test.input))
			require.NoError(t, err)
			beatCfg := config.MustNewConfigFrom(rawConf)

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

func loadBeatConf(fileName string) (map[string]any, error) {
	// Clean the path before using it.
	content, err := os.ReadFile(filepath.Clean(fileName))
	if err != nil {
		return nil, fmt.Errorf("unable to read the file %v: %w", fileName, err)
	}

	var rawConf map[string]any
	if err = yaml.Unmarshal(content, &rawConf); err != nil {
		return nil, err
	}

	return rawConf, nil
}
