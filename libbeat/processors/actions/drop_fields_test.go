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

	"github.com/elastic/beats/v7/libbeat/common/match"
	config2 "github.com/elastic/elastic-agent-libs/config"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestDropFieldRun(t *testing.T) {
	event := &beat.Event{
		Fields: mapstr.M{
			"field": "value",
		},
		Meta: mapstr.M{
			"meta_field": "value",
		},
	}

	t.Run("supports a normal field", func(t *testing.T) {
		p := dropFields{
			Fields: []string{"field"},
		}

		newEvent, err := p.Run(event)
		assert.NoError(t, err)
		assert.Equal(t, mapstr.M{}, newEvent.Fields)
		assert.Equal(t, event.Meta, newEvent.Meta)
	})

	t.Run("supports a metadata field", func(t *testing.T) {
		p := dropFields{
			Fields: []string{"@metadata.meta_field"},
		}

		newEvent, err := p.Run(event)
		assert.NoError(t, err)
		assert.Equal(t, mapstr.M{}, newEvent.Meta)
		assert.Equal(t, event.Fields, newEvent.Fields)
	})

	t.Run("supports a regexp field", func(t *testing.T) {
		event = &beat.Event{
			Fields: mapstr.M{
				"field_1": mapstr.M{
					"subfield_1": "sf_1_value",
					"subfield_2": mapstr.M{
						"subfield_2_1": "sf_2_1_value",
						"subfield_2_2": "sf_2_2_value",
					},
					"subfield_3": mapstr.M{
						"subfield_3_1": "sf_3_1_value",
						"subfield_3_2": "sf_3_2_value",
					},
				},
				"field_2": "f_2_value",
			},
		}

		p := dropFields{
			RegexpFields: []match.Matcher{match.MustCompile("field_2$"), match.MustCompile("field_1\\.(.*)\\.subfield_2_1"), match.MustCompile("field_1\\.subfield_3(.*)")},
			Fields:       []string{},
		}

		newEvent, err := p.Run(event)
		assert.NoError(t, err)
		assert.Equal(t, mapstr.M{
			"field_1": mapstr.M{
				"subfield_1": "sf_1_value",
			},
		}, newEvent.Fields)
	})
}

func TestNewDropFields(t *testing.T) {
	t.Run("detects regexp fields and assign to RegexpFields property", func(t *testing.T) {
		c := config2.MustNewConfigFrom(map[string]interface{}{
			"fields": []string{"/field_.*1/", "/second/", "third"},
		})

		procInt, err := newDropFields(c)
		processor := procInt.(*dropFields)

		assert.NoError(t, err)
		assert.Equal(t, []string{"third"}, processor.Fields)
		assert.Equal(t, "<substring 'second'>", processor.RegexpFields[0].String())
		assert.Equal(t, "field_(?-s:.)*1", processor.RegexpFields[1].String())
	})

	t.Run("returns error when regexp field is badly written", func(t *testing.T) {
		c := config2.MustNewConfigFrom(map[string]interface{}{
			"fields": []string{"/[//"},
		})

		_, err := newDropFields(c)

		assert.Equal(t, "wrong configuration in drop_fields. error parsing regexp: missing closing ]: `[/`", err.Error())
	})
}
