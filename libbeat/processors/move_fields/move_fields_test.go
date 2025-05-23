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

package move_fields

import (
	"reflect"
	"testing"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestMoveFields(t *testing.T) {
	cases := []struct {
		name         string
		in, expected mapstr.M
		p            *moveFields
	}{
		{
			"move fields to app.rpc exclude 'method'",
			mapstr.M{"app": mapstr.M{"version": 1, "method": "2"}, "other": 3},
			mapstr.M{"app": mapstr.M{"method": "2"}, "rpc": mapstr.M{"version": 1}, "other": 3},
			&moveFields{
				config: moveFieldsConfig{
					From:    "app",
					Fields:  nil,
					To:      "rpc.",
					Exclude: []string{"method"},
				},
				excludeMap: map[string]struct{}{"method": {}},
			},
		},
		{
			"move 'version' to app.rpc",
			mapstr.M{"app": mapstr.M{"version": 1, "method": "2", "other": 3}},
			mapstr.M{"app": mapstr.M{"method": "2", "other": 3}, "rpc": mapstr.M{"version": 1}},
			&moveFields{
				config: moveFieldsConfig{
					From:    "app",
					Fields:  []string{"version"},
					To:      "rpc.",
					Exclude: nil,
				},
				excludeMap: nil,
			},
		},
		{
			"move fields from app object to rpc object",
			mapstr.M{"app": mapstr.M{"version": 1, "method": "2"}, "other": 3},
			mapstr.M{"app": mapstr.M{}, "rpc": mapstr.M{"method": "2", "version": 1}, "other": 3},
			&moveFields{
				config: moveFieldsConfig{
					From:    "app",
					Fields:  nil,
					To:      "rpc.",
					Exclude: nil,
				},
				excludeMap: nil,
			},
		},
		{
			"add prefix to fields in app object",
			mapstr.M{"app": mapstr.M{"version": 1, "method": "2"}, "other": 3},
			mapstr.M{"app": mapstr.M{}, "rpc_method": "2", "rpc_version": 1, "other": 3},
			&moveFields{
				config: moveFieldsConfig{
					From:    "app",
					Fields:  nil,
					To:      "rpc_",
					Exclude: nil,
				},
				excludeMap: nil,
			},
		},
		{
			"add prefix to fields in event root",
			mapstr.M{"app": mapstr.M{"version": 1, "method": "2"}, "other": 3},
			mapstr.M{"my_prefix_app": mapstr.M{"version": 1, "method": "2"}, "my_prefix_other": 3},
			&moveFields{
				config: moveFieldsConfig{
					From:    "",
					Fields:  nil,
					To:      "my_prefix_",
					Exclude: nil,
				},
				excludeMap: nil,
			},
		},
		{
			`move field "other" into app.b object`,
			mapstr.M{"app": mapstr.M{"version": 1, "method": "2"}, "other": 3},
			mapstr.M{"app": mapstr.M{"version": 1, "method": "2", "b": mapstr.M{"my_prefix_other": 3}}},
			&moveFields{
				config: moveFieldsConfig{
					From:    "",
					Fields:  []string{"other"},
					To:      "app.b.my_prefix_",
					Exclude: nil,
				},
				excludeMap: nil,
			},
		},
	}

	for idx, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			evt := &beat.Event{Fields: c.in.Clone()}
			out, err := c.p.Run(evt)
			if err != nil {
				t.Fatal(err)
			}
			except, output := c.expected.String(), out.Fields.String()
			if !reflect.DeepEqual(c.expected, out.Fields) {
				t.Fatalf("move field test case failed, out: %s, except: %s, index: %d\n",
					output, except, idx)
			}
		})
	}
}
