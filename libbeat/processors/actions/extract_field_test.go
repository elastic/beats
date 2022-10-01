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
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

func TestCommonPaths(t *testing.T) {
	var tests = []struct {
		Value, Field, Separator, Target, Result string
		Index                                   int
		Error                                   bool
	}{
		// Common docker case
		{
			Value:     "/var/lib/docker/containers/f1510836197d7c34da22cf796dba5640f87c04de5c95cf0adc11b85f1e1c1528/f1510836197d7c34da22cf796dba5640f87c04de5c95cf0adc11b85f1e1c1528-json.log",
			Field:     "source",
			Separator: "/",
			Target:    "docker.container.id",
			Index:     4,
			Result:    "f1510836197d7c34da22cf796dba5640f87c04de5c95cf0adc11b85f1e1c1528",
		},
		{
			Value:     "/var/lib/foo/bar",
			Field:     "other_field",
			Separator: "/",
			Target:    "destination",
			Index:     3,
			Result:    "bar",
		},
		{
			Value:     "-var-lib-foo-bar",
			Field:     "source",
			Separator: "-",
			Target:    "destination",
			Index:     2,
			Result:    "foo",
		},
		{
			Value:     "*var*lib*foo*bar",
			Field:     "source",
			Separator: "*",
			Target:    "destination",
			Index:     0,
			Result:    "var",
		},
		{
			Value:     "/var/lib/foo/bar",
			Field:     "source",
			Separator: "*",
			Target:    "destination",
			Index:     10, // out of range
			Result:    "var",
			Error:     true,
		},
	}

	for _, test := range tests {
		var testConfig, _ = common.NewConfigFrom(map[string]interface{}{
			"field":     test.Field,
			"separator": test.Separator,
			"index":     test.Index,
			"target":    test.Target,
		})

		// Configure input to
		input := common.MapStr{
			test.Field: test.Value,
		}

		event, err := runExtractField(t, testConfig, input)
		if test.Error {
			assert.Error(t, err)
		} else {

			assert.NoError(t, err)
			result, err := event.Fields.GetValue(test.Target)
			if err != nil {
				t.Fatalf("could not get target field: %s", err)
			}
			assert.Equal(t, result.(string), test.Result)
		}

		// Event must be present, even on error
		assert.NotNil(t, event)
	}

	t.Run("supports a metadata field", func(t *testing.T) {
		var config, _ = common.NewConfigFrom(map[string]interface{}{
			"field":     "field",
			"separator": "/",
			"index":     3,
			"target":    "@metadata.field",
		})

		event := &beat.Event{
			Meta: common.MapStr{},
			Fields: common.MapStr{
				"field": "/var/lib/foo/bar",
			},
		}

		expectedFields := common.MapStr{
			"field": "/var/lib/foo/bar",
		}
		expectedMeta := common.MapStr{
			"field": "bar",
		}

		p, err := NewExtractField(config)
		if err != nil {
			t.Fatalf("error initializing extract_field: %s", err)
		}

		newEvent, err := p.Run(event)
		assert.NoError(t, err)
		assert.Equal(t, expectedFields, newEvent.Fields)
		assert.Equal(t, expectedMeta, newEvent.Meta)
	})
}

func runExtractField(t *testing.T, config *common.Config, input common.MapStr) (*beat.Event, error) {
	logp.TestingSetup()

	p, err := NewExtractField(config)
	if err != nil {
		t.Fatalf("error initializing extract_field: %s", err)
	}

	return p.Run(&beat.Event{Fields: input})
}
