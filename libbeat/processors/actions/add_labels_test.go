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

func TestAddLabels(t *testing.T) {
	multi := func(strs ...string) []string { return strs }
	single := func(str string) []string { return multi(str) }

	testProcessors(t, map[string]testCase{
		"add label": {
			eventFields: mapstr.M{},
			wantFields: mapstr.M{
				"labels": mapstr.M{"label": "test"},
			},
			cfg: single(`{add_labels: {labels: {label: test}}}`),
		},
		"add dotted label": {
			eventFields: mapstr.M{},
			wantFields: mapstr.M{
				"labels": mapstr.M{"a.b": "test"},
			},
			cfg: single(`{add_labels: {labels: {a.b: test}}}`),
		},
		"add nested labels": {
			eventFields: mapstr.M{},
			wantFields: mapstr.M{
				"labels": mapstr.M{"a.b": "test", "a.c": "test2"},
			},
			cfg: single(`{add_labels: {labels: {a: {b: test, c: test2}}}}`),
		},
		"merge labels": {
			eventFields: mapstr.M{},
			wantFields: mapstr.M{
				"labels": mapstr.M{"l1": "a", "l2": "b", "lc": "b"},
			},
			cfg: multi(
				`{add_labels.labels: {l1: a, lc: a}}`,
				`{add_labels.labels: {l2: b, lc: b}}`,
			),
		},
		"add array": {
			eventFields: mapstr.M{},
			wantFields: mapstr.M{
				"labels": mapstr.M{
					"array.0":       "foo",
					"array.1":       "bar",
					"array.2.hello": "world",
				},
			},
			cfg: single(`{add_labels: {labels: {array: ["foo", "bar", {"hello": "world"}]}}}`),
		},
	})
}
