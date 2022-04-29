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

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/processors"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type testCase struct {
	eventFields mapstr.M
	eventMeta   mapstr.M
	wantFields  mapstr.M
	wantMeta    mapstr.M
	cfg         []string
}

func testProcessors(t *testing.T, cases map[string]testCase) {
	for name, test := range cases {
		test := test
		t.Run(name, func(t *testing.T) {
			ps := make([]*processors.Processors, len(test.cfg))
			for i := range test.cfg {
				config, err := conf.NewConfigWithYAML([]byte(test.cfg[i]), "test")
				if err != nil {
					t.Fatalf("Failed to create config(%v): %+v", i, err)
				}

				ps[i], err = processors.New([]*conf.C{config})
				if err != nil {
					t.Fatalf("Failed to create add_tags processor(%v): %+v", i, err)
				}
			}

			current := &beat.Event{}
			if test.eventFields != nil {
				current.Fields = test.eventFields.Clone()
			}
			if test.eventMeta != nil {
				current.Meta = test.eventMeta.Clone()
			}
			for i, processor := range ps {
				var err error
				current, err = processor.Run(current)
				if err != nil {
					t.Fatal(err)
				}
				if current == nil {
					t.Fatalf("Event dropped(%v)", i)
				}
			}

			assert.Equal(t, test.wantFields, current.Fields)
			assert.Equal(t, test.wantMeta, current.Meta)
		})
	}
}
