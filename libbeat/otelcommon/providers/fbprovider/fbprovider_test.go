// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fbprovider

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/confmap/confmaptest"
	"gopkg.in/yaml.v2"
)

func TestFileBeatProvider(t *testing.T) {
	p := provider{}

	tests := []struct {
		name      string
		input     string
		expOutput string
		experr    bool
	}{
		{ // change oteloutput.yml as more configurations are supported - so it gets covered by this test
			name:      "correct input type",
			input:     "supported.yml",
			expOutput: "oteloutput.yml",
			experr:    false,
		},
		{ // change testdata/invalid.yml as and when more output configurations are supported
			name:      "unsupported beat output configured",
			input:     "invalid.yml",
			expOutput: "",
			experr:    true,
		},
		// unsupported elasticsearch config provided
		// {
		// 	name:   "unsupported beat output configured",
		// 	input:  "invalid.yml",
		// 	experr: true,
		// },
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ret, err := p.Retrieve(context.Background(), "fb:"+filepath.Join("testdata", test.input), nil)
			if test.experr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				retValue, err := ret.AsRaw()
				require.NoError(t, err)
				expectedValue, _ := confmaptest.LoadConf(filepath.Join("testdata", test.expOutput))
				expMap := expectedValue.ToStringMap()

				// convert both expected and actual output to same format
				expectedYAML, err := yaml.Marshal(expMap)
				require.NoError(t, err)

				retYAML, err := yaml.Marshal(retValue)
				require.NoError(t, err)

				assert.Equal(t, string(expectedYAML), string(retYAML))
				assert.NoError(t, p.Shutdown(context.Background()))
			}
		})
	}

}
