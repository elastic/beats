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

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/processors"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestAddFields(t *testing.T) {
	multi := func(strs ...string) []string { return strs }
	single := func(str string) []string { return multi(str) }

	testProcessors(t, map[string]testCase{
		"add field": {
			eventFields: mapstr.M{},
			wantFields: mapstr.M{
				"fields": mapstr.M{"field": "test"},
			},
			cfg: single(`{add_fields: {fields: {field: test}}}`),
		},
		"custom target": {
			eventFields: mapstr.M{},
			wantFields: mapstr.M{
				"my": mapstr.M{"field": "test"},
			},
			cfg: single(`{add_fields: {target: my, fields: {field: test}}}`),
		},
		"overwrite existing field": {
			eventFields: mapstr.M{
				"fields": mapstr.M{"field": "old"},
			},
			wantFields: mapstr.M{"fields": mapstr.M{"field": "test"}},
			cfg:        single(`{add_fields: {fields: {field: test}}}`),
		},
		"merge with existing meta": {
			eventMeta: mapstr.M{
				"_id": "unique",
			},
			wantMeta: mapstr.M{
				"_id":     "unique",
				"op_type": "index",
			},
			cfg: single(`{add_fields: {target: "@metadata", fields: {op_type: "index"}}}`),
		},
		"merge with existing fields": {
			eventFields: mapstr.M{
				"fields": mapstr.M{"existing": "a"},
			},
			wantFields: mapstr.M{
				"fields": mapstr.M{"existing": "a", "field": "test"},
			},
			cfg: single(`{add_fields: {fields: {field: test}}}`),
		},
		"combine 2 processors": {
			eventFields: mapstr.M{},
			wantFields: mapstr.M{
				"fields": mapstr.M{
					"l1": "a",
					"l2": "b",
				},
			},
			cfg: multi(
				`{add_fields: {fields: {l1: a}}}`,
				`{add_fields: {fields: {l2: b}}}`,
			),
		},
		"different targets": {
			eventFields: mapstr.M{},
			wantFields: mapstr.M{
				"a": mapstr.M{"l1": "a"},
				"b": mapstr.M{"l2": "b"},
			},
			cfg: multi(
				`{add_fields: {target: a, fields: {l1: a}}}`,
				`{add_fields: {target: b, fields: {l2: b}}}`,
			),
		},
		"under root": {
			eventFields: mapstr.M{},
			wantFields: mapstr.M{
				"a": mapstr.M{"b": "test"},
			},
			cfg: single(
				`{add_fields: {target: "", fields: {a.b: test}}}`,
			),
		},
		"merge under root": {
			eventFields: mapstr.M{
				"a": mapstr.M{"old": "value"},
			},
			wantFields: mapstr.M{
				"a": mapstr.M{"old": "value", "new": "test"},
			},
			cfg: single(
				`{add_fields: {target: "", fields: {a.new: test}}}`,
			),
		},
		"overwrite existing under root": {
			eventFields: mapstr.M{
				"a": mapstr.M{"keep": "value", "change": "a"},
			},
			wantFields: mapstr.M{
				"a": mapstr.M{"keep": "value", "change": "b"},
			},
			cfg: single(
				`{add_fields: {target: "", fields: {a.change: b}}}`,
			),
		},
		"add fields to nil event": {
			eventFields: nil,
			wantFields: mapstr.M{
				"fields": mapstr.M{"field": "test"},
			},
			cfg: single(`{add_fields: {fields: {field: test}}}`),
		},
	})
}

// TestAddFieldsEquivalence verifies that events processed through multiple
// add_fields processors and a single add_fields_multiple both produce the
// expected output.
func TestAddFieldsEquivalence(t *testing.T) {
	cases := map[string]struct {
		eventFields    mapstr.M
		eventMeta      mapstr.M
		wantFields     mapstr.M
		wantMeta       mapstr.M
		multipleAddCfg []string
		singleAddCfg   string
	}{
		"two fields to default target": {
			eventFields: mapstr.M{},
			wantFields: mapstr.M{
				"fields": mapstr.M{"l1": "a", "l2": "b"},
			},
			multipleAddCfg: []string{
				`{add_fields: {fields: {l1: a}}}`,
				`{add_fields: {fields: {l2: b}}}`,
			},
			singleAddCfg: `{add_fields_multiple: [{fields: {l1: a}}, {fields: {l2: b}}]}`,
		},
		"different targets": {
			eventFields: mapstr.M{},
			wantFields: mapstr.M{
				"a": mapstr.M{"l1": "a"},
				"b": mapstr.M{"l2": "b"},
			},
			multipleAddCfg: []string{
				`{add_fields: {target: a, fields: {l1: a}}}`,
				`{add_fields: {target: b, fields: {l2: b}}}`,
			},
			singleAddCfg: `{add_fields_multiple: [{target: a, fields: {l1: a}}, {target: b, fields: {l2: b}}]}`,
		},
		"many fields to same target": {
			eventFields: mapstr.M{},
			wantFields: mapstr.M{
				"project": mapstr.M{
					"name": "myproject",
					"id":   "574734885120952459",
					"env":  "production",
				},
			},
			multipleAddCfg: []string{
				`{add_fields: {target: project, fields: {name: myproject}}}`,
				`{add_fields: {target: project, fields: {id: "574734885120952459"}}}`,
				`{add_fields: {target: project, fields: {env: production}}}`,
			},
			singleAddCfg: `{add_fields_multiple: [{target: project, fields: {name: myproject, id: "574734885120952459", env: production}}]}`,
		},
		"merge with pre-existing event fields": {
			eventFields: mapstr.M{
				"fields": mapstr.M{"existing": "a"},
			},
			wantFields: mapstr.M{
				"fields": mapstr.M{
					"existing": "a",
					"field1":   "value1",
					"field2":   "value2",
				},
			},
			multipleAddCfg: []string{
				`{add_fields: {fields: {field1: value1}}}`,
				`{add_fields: {fields: {field2: value2}}}`,
			},
			singleAddCfg: `{add_fields_multiple: [{fields: {field1: value1}}, {fields: {field2: value2}}]}`,
		},
		"overwrite existing field": {
			eventFields: mapstr.M{
				"fields": mapstr.M{"field": "old"},
			},
			wantFields: mapstr.M{
				"fields": mapstr.M{
					"field": "new",
					"other": "value",
				},
			},
			multipleAddCfg: []string{
				`{add_fields: {fields: {field: new}}}`,
				`{add_fields: {fields: {other: value}}}`,
			},
			singleAddCfg: `{add_fields_multiple: [{fields: {field: new}}, {fields: {other: value}}]}`,
		},
		"metadata target": {
			eventMeta: mapstr.M{
				"_id": "unique",
			},
			wantMeta: mapstr.M{
				"_id":      "unique",
				"op_type":  "index",
				"pipeline": "my-pipeline",
			},
			multipleAddCfg: []string{
				`{add_fields: {target: "@metadata", fields: {op_type: "index"}}}`,
				`{add_fields: {target: "@metadata", fields: {pipeline: "my-pipeline"}}}`,
			},
			singleAddCfg: `{add_fields_multiple: [{target: "@metadata", fields: {op_type: "index"}}, {target: "@metadata", fields: {pipeline: "my-pipeline"}}]}`,
		},
		"mixed targets and root": {
			eventFields: mapstr.M{},
			wantFields: mapstr.M{
				"root_field": "root_val",
				"nested":     mapstr.M{"key": "val"},
				"fields":     mapstr.M{"default_key": "default_val"},
			},
			multipleAddCfg: []string{
				`{add_fields: {target: "", fields: {root_field: root_val}}}`,
				`{add_fields: {target: nested, fields: {key: val}}}`,
				`{add_fields: {fields: {default_key: default_val}}}`,
			},
			singleAddCfg: `{add_fields_multiple: [{target: "", fields: {root_field: root_val}}, {target: nested, fields: {key: val}}, {fields: {default_key: default_val}}]}`,
		},
		"five fields across three targets": {
			eventFields: mapstr.M{},
			wantFields: mapstr.M{
				"host":  mapstr.M{"name": "server1", "ip": "10.0.0.1"},
				"agent": mapstr.M{"version": "8.0.0", "type": "filebeat"},
				"cloud": mapstr.M{"provider": "aws"},
			},
			multipleAddCfg: []string{
				`{add_fields: {target: host, fields: {name: server1}}}`,
				`{add_fields: {target: host, fields: {ip: "10.0.0.1"}}}`,
				`{add_fields: {target: agent, fields: {version: "8.0.0"}}}`,
				`{add_fields: {target: agent, fields: {type: filebeat}}}`,
				`{add_fields: {target: cloud, fields: {provider: aws}}}`,
			},
			singleAddCfg: `{add_fields_multiple: [{target: host, fields: {name: server1, ip: "10.0.0.1"}}, {target: agent, fields: {version: "8.0.0", type: filebeat}}, {target: cloud, fields: {provider: aws}}]}`,
		},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			// Run through multiple add_fields processors
			multiEvent := &beat.Event{}
			if tc.eventFields != nil {
				multiEvent.Fields = tc.eventFields.Clone()
			}
			if tc.eventMeta != nil {
				multiEvent.Meta = tc.eventMeta.Clone()
			}
			for i, cfgStr := range tc.multipleAddCfg {
				c, err := conf.NewConfigWithYAML([]byte(cfgStr), "test")
				require.NoError(t, err, "config %d", i)
				ps, err := processors.New([]*conf.C{c}, logptest.NewTestingLogger(t, ""))
				require.NoError(t, err, "processor %d", i)
				multiEvent, err = ps.Run(multiEvent)
				require.NoError(t, err, "run %d", i)
				require.NotNil(t, multiEvent, "event dropped at %d", i)
			}

			// Run through single add_fields_multiple processor
			singleEvent := &beat.Event{}
			if tc.eventFields != nil {
				singleEvent.Fields = tc.eventFields.Clone()
			}
			if tc.eventMeta != nil {
				singleEvent.Meta = tc.eventMeta.Clone()
			}
			c, err := conf.NewConfigWithYAML([]byte(tc.singleAddCfg), "test")
			require.NoError(t, err)
			ps, err := processors.New([]*conf.C{c}, logptest.NewTestingLogger(t, ""))
			require.NoError(t, err)
			singleEvent, err = ps.Run(singleEvent)
			require.NoError(t, err)
			require.NotNil(t, singleEvent)

			// Both must match the expected output
			assert.Equal(t, tc.wantFields, multiEvent.Fields, "multiple add_fields: Fields mismatch")
			assert.Equal(t, tc.wantMeta, multiEvent.Meta, "multiple add_fields: Meta mismatch")

			assert.Equal(t, tc.wantFields, singleEvent.Fields, "add_fields_multiple: Fields mismatch")
			assert.Equal(t, tc.wantMeta, singleEvent.Meta, "add_fields_multiple: Meta mismatch")
		})
	}
}
