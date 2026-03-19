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

func TestAddFieldsMultiple(t *testing.T) {
	single := func(str string) []string { return []string{str} }

	testProcessors(t, map[string]testCase{
		"single entry with multiple fields": {
			eventFields: mapstr.M{},
			wantFields: mapstr.M{
				"fields": mapstr.M{
					"field1": "value1",
					"field2": "value2",
					"field3": "value3",
				},
			},
			cfg: single(`{add_fields_multiple: [{fields: {field1: value1, field2: value2, field3: value3}}]}`),
		},
		"single entry with custom target": {
			eventFields: mapstr.M{},
			wantFields: mapstr.M{
				"project": mapstr.M{
					"name": "myproject",
					"id":   "574734885120952459",
					"env":  "production",
				},
			},
			cfg: single(`{add_fields_multiple: [{target: project, fields: {name: myproject, id: "574734885120952459", env: production}}]}`),
		},
		"multiple entries with different targets": {
			eventFields: mapstr.M{},
			wantFields: mapstr.M{
				"a": mapstr.M{"l1": "a"},
				"b": mapstr.M{"l2": "b"},
			},
			cfg: single(`{add_fields_multiple: [{target: a, fields: {l1: a}}, {target: b, fields: {l2: b}}]}`),
		},
		"merge with existing fields": {
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
			cfg: single(`{add_fields_multiple: [{fields: {field1: value1, field2: value2}}]}`),
		},
		"multiple entries to same default target": {
			eventFields: mapstr.M{},
			wantFields: mapstr.M{
				"fields": mapstr.M{
					"l1": "a",
					"l2": "b",
				},
			},
			cfg: single(`{add_fields_multiple: [{fields: {l1: a}}, {fields: {l2: b}}]}`),
		},
		"metadata target": {
			eventMeta: mapstr.M{
				"_id": "unique",
			},
			wantMeta: mapstr.M{
				"_id":     "unique",
				"op_type": "index",
			},
			cfg: single(`{add_fields_multiple: [{target: "@metadata", fields: {op_type: "index"}}]}`),
		},
	})
}
