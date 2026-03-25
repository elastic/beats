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
