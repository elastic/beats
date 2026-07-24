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

package actions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/otel/otelmap"
	"github.com/elastic/beats/v7/libbeat/processors"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type addFieldsCase struct {
	eventFields mapstr.M
	eventMeta   mapstr.M
	wantFields  mapstr.M
	wantMeta    mapstr.M
	cfg         []string
}

func TestAddFields(t *testing.T) {
	multi := func(strs ...string) []string { return strs }
	single := func(str string) []string { return multi(str) }

	cases := map[string]addFieldsCase{
		"add field": {
			eventFields: mapstr.M{},
			wantFields:  mapstr.M{"fields": mapstr.M{"field": "test"}},
			cfg:         single(`{add_fields: {fields: {field: test}}}`),
		},
		"custom target": {
			eventFields: mapstr.M{},
			wantFields:  mapstr.M{"my": mapstr.M{"field": "test"}},
			cfg:         single(`{add_fields: {target: my, fields: {field: test}}}`),
		},
		"overwrite existing field": {
			eventFields: mapstr.M{"fields": mapstr.M{"field": "old"}},
			wantFields:  mapstr.M{"fields": mapstr.M{"field": "test"}},
			cfg:         single(`{add_fields: {fields: {field: test}}}`),
		},
		"merge with existing meta": {
			eventMeta: mapstr.M{"_id": "unique"},
			wantMeta:  mapstr.M{"_id": "unique", "op_type": "index"},
			cfg:       single(`{add_fields: {target: "@metadata", fields: {op_type: "index"}}}`),
		},
		"merge with existing fields": {
			eventFields: mapstr.M{"fields": mapstr.M{"existing": "a"}},
			wantFields:  mapstr.M{"fields": mapstr.M{"existing": "a", "field": "test"}},
			cfg:         single(`{add_fields: {fields: {field: test}}}`),
		},
		"combine 2 processors": {
			eventFields: mapstr.M{},
			wantFields:  mapstr.M{"fields": mapstr.M{"l1": "a", "l2": "b"}},
			cfg: multi(
				`{add_fields: {fields: {l1: a}}}`,
				`{add_fields: {fields: {l2: b}}}`,
			),
		},
		"different targets": {
			eventFields: mapstr.M{},
			wantFields:  mapstr.M{"a": mapstr.M{"l1": "a"}, "b": mapstr.M{"l2": "b"}},
			cfg: multi(
				`{add_fields: {target: a, fields: {l1: a}}}`,
				`{add_fields: {target: b, fields: {l2: b}}}`,
			),
		},
		"under root": {
			eventFields: mapstr.M{},
			wantFields:  mapstr.M{"a": mapstr.M{"b": "test"}},
			cfg:         single(`{add_fields: {target: "", fields: {a.b: test}}}`),
		},
		"merge under root": {
			eventFields: mapstr.M{"a": mapstr.M{"old": "value"}},
			wantFields:  mapstr.M{"a": mapstr.M{"old": "value", "new": "test"}},
			cfg:         single(`{add_fields: {target: "", fields: {a.new: test}}}`),
		},
		"overwrite existing under root": {
			eventFields: mapstr.M{"a": mapstr.M{"keep": "value", "change": "a"}},
			wantFields:  mapstr.M{"a": mapstr.M{"keep": "value", "change": "b"}},
			cfg:         single(`{add_fields: {target: "", fields: {a.change: b}}}`),
		},
		"add fields to nil event": {
			eventFields: nil,
			wantFields:  mapstr.M{"fields": mapstr.M{"field": "test"}},
			cfg:         single(`{add_fields: {fields: {field: test}}}`),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ps := make([]*processors.Processors, len(tc.cfg))
			for i := range tc.cfg {
				config, err := conf.NewConfigWithYAML([]byte(tc.cfg[i]), "test")
				require.NoError(t, err)
				ps[i], err = processors.New([]*conf.C{config}, logptest.NewTestingLogger(t, ""))
				require.NoError(t, err)
			}

			// Legacy Run path.
			current := &beat.Event{}
			if tc.eventFields != nil {
				current.Fields = tc.eventFields.Clone()
			}
			if tc.eventMeta != nil {
				current.Meta = tc.eventMeta.Clone()
			}
			for i, p := range ps {
				var err error
				current, err = p.Run(current)
				require.NoError(t, err)
				require.NotNilf(t, current, "event dropped by processor %d", i)
			}
			assert.Equal(t, tc.wantFields, current.Fields)
			assert.Equal(t, tc.wantMeta, current.Meta)

			// RunPdata path: assert Run == RunPdata.
			// otelconsumer serializes beat.Event.Meta into the body under "@metadata",
			// so seed it there and include it in the expected output comparison.
			body := pcommon.NewMap()
			if tc.eventFields != nil {
				require.NoError(t, otelmap.FromMapstr(body, tc.eventFields))
			}
			if tc.eventMeta != nil {
				require.NoError(t, otelmap.FromMapstr(body.PutEmptyMap("@metadata"), tc.eventMeta))
			}
			for _, p := range ps {
				for _, proc := range p.List {
					pp, ok := proc.(processors.PdataProcessor)
					require.True(t, ok, "processor %T does not implement PdataProcessor", proc)
					drop, err := pp.RunPdata(body)
					require.NoError(t, err)
					require.False(t, drop)
				}
			}
			// Build the expected output from the legacy result. event.Meta maps to
			// "@metadata" in the pdata body, so merge it in before normalizing.
			expectedFields := current.Fields.Clone()
			if len(current.Meta) > 0 {
				expectedFields["@metadata"] = current.Meta
			}
			legacyNorm := pcommon.NewMap()
			require.NoError(t, otelmap.FromMapstr(legacyNorm, expectedFields))
			wantPdata := otelmap.ToMapstr(legacyNorm)
			gotPdata := otelmap.ToMapstr(body)
			assert.Equal(t, wantPdata, gotPdata)
		})
	}
}
