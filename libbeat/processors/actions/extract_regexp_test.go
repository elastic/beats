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

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

func TestExtractRegexp(t *testing.T) {
	var tests = []struct {
		Value, Field, Regexp, Prefix, ResultField string
		Result                                    common.MapStr
		Error                                     bool
	}{
		{
			Value:       "/opt/data/alloc/1234/alloc/logs/test.stderr.0",
			Field:       "log.file.path",
			Regexp:      `.*/(?P<allocation>.+)/alloc/logs/(?P<task>.+)\.(?P<stream>std.+)\.[0-9]+`,
			Prefix:      "result.",
			ResultField: "result",
			Result:      common.MapStr{"allocation": "1234", "task": "test", "stream": "stderr"},
		},
		{
			Value:       "/var/log/messages-1",
			Field:       "log.file.path",
			Regexp:      `.*/(?P<filename>.+)-[0-9]+`,
			Prefix:      "result.",
			ResultField: "result",
			Result:      common.MapStr{"filename": "messages"},
		},
	}

	for _, test := range tests {
		var testConfig, _ = common.NewConfigFrom(map[string]interface{}{
			"regexp": test.Regexp,
			"field":  test.Field,
			"prefix": test.Prefix,
		})

		// Configure input to
		input := common.MapStr{
			test.Field: test.Value,
		}

		event, err := runExtractRegexp(t, testConfig, input)
		if test.Error {
			assert.NotNil(t, err)
		} else {
			assert.Nil(t, err)
			result, err := event.Fields.GetValue(test.ResultField)
			if err != nil {
				t.Fatalf("could not get result field: %s", err)
			}
			assert.Equal(t, result, test.Result)
		}

		// Event must be present, even on error
		assert.NotNil(t, event)
	}
}

func runExtractRegexp(t *testing.T, config *common.Config, input common.MapStr) (*beat.Event, error) {
	logp.TestingSetup()

	p, err := NewExtractRegexp(config)
	if err != nil {
		t.Fatalf("error initializing extract_regexp: %s", err)
	}

	return p.Run(&beat.Event{Fields: input})
}
