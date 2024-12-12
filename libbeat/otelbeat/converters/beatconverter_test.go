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

package converters

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/confmap/confmaptest"
	"gopkg.in/yaml.v2"
)

func TestConverter(t *testing.T) {
	c := converter{}

	tests := []struct {
		name      string
		input     string
		expOutput string
		experr    bool
	}{
		{
			name:      "correct input type",
			input:     "supported.yml",
			expOutput: "oteloutput.yml",
			experr:    false,
		},
		{
			name:      "unsupported output type is configured",
			input:     "unsupported-output.yml",
			expOutput: "",
			experr:    true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input, err := confmaptest.LoadConf(filepath.Join("testdata", test.input))
			require.NoError(t, err, "could not load file")

			err = c.Convert(context.Background(), input)
			if test.experr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				expectedValue, _ := confmaptest.LoadConf(filepath.Join("testdata", test.expOutput))

				// convert expected and returned value to same format
				expectedYAML, err := yaml.Marshal(expectedValue.ToStringMap())
				require.NoError(t, err)

				retYAML, err := yaml.Marshal(input.ToStringMap())
				require.NoError(t, err)

				assert.Equal(t, string(expectedYAML), string(retYAML))
			}
		})
	}

}
