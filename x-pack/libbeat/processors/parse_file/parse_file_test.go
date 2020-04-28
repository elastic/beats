// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package parse_file

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
)

func TestParseFile(t *testing.T) {
	type enrichedFields struct {
		path     string
		expected common.MapStr
	}
	tests := []struct {
		name     string
		enriched enrichedFields
		config   map[string]interface{}
		fields   common.MapStr
	}{
		{
			name: "default config",
			enriched: enrichedFields{
				path: "file.pe",
				expected: common.MapStr{
					"imphash":            "ca7337bd1dfa93fd45ff30b369488a37",
					"company":            "Microsoft Corporation",
					"description":        "Windows Calculator",
					"file_version":       "6.1.7600.16385 (win7_rtm.090713-1255)",
					"original_file_name": "CALC.EXE",
					"product":            "Microsoft® Windows® Operating System",
				},
			},
			config: map[string]interface{}{},
			fields: common.MapStr{
				"file.path": "./testdata/calc.exe",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			evt := beat.Event{
				Fields: test.fields,
			}
			p, err := New(common.MustNewConfigFrom(test.config))
			require.NoError(t, err)
			observed, err := p.Run(&evt)
			require.NoError(t, err)
			if test.enriched.path != "" {
				enriched, err := observed.Fields.GetValue(test.enriched.path)
				require.NoError(t, err)
				require.Equal(t, test.enriched.expected, enriched)
			}
		})
	}
}
