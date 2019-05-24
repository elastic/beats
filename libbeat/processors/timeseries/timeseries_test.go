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

package timeseries

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/mapping"
)

var (
	truthy = true
	falsy  = false

	fields = mapping.Fields{
		mapping.Field{Name: "context.first", Type: "long", Description: "a dimension", Dimension: &truthy},
		mapping.Field{
			Name: "context",
			Type: "group",
			Fields: mapping.Fields{
				mapping.Field{Name: "second", Type: "keyword", Dimension: &truthy},
			},
		},
		mapping.Field{
			Name: "context",
			Type: "group",
			Fields: mapping.Fields{
				mapping.Field{Name: "third", Dimension: &truthy},
			},
		},
		mapping.Field{Name: "type-less"},
		mapping.Field{Name: "not-a-dimension", Type: "long"},
		mapping.Field{Name: "dimension-by-default", Type: "keyword"},
		mapping.Field{Name: "overwritten-field1", Type: "long", Dimension: &truthy},
		mapping.Field{Name: "overwritten-field1", Overwrite: true, Type: "long", Dimension: &falsy},
		mapping.Field{Name: "overwritten-field2", Overwrite: true, Type: "long"},
		mapping.Field{Name: "overwritten-field2", Type: "keyword", Dimension: &truthy},
		mapping.Field{
			Name: "nested-obj",
			Type: "object",
			Fields: mapping.Fields{
				mapping.Field{
					Name:       "object-of-keywords",
					Type:       "object",
					ObjectType: "keyword",
				},
				mapping.Field{
					Name:       "wildcard-object-of-keywords.*",
					Type:       "object",
					ObjectType: "keyword",
				},
				// todo: not supported
				mapping.Field{
					Name: "obj-type-params",
					ObjectTypeParams: []mapping.ObjectTypeCfg{
						{ObjectType: "keyword"},
						{ObjectType: "boolean"},
					},
					Type: "object",
				},
				mapping.Field{Name: "not-a-dimension", Type: "long"},
			},
		},
		mapping.Field{
			Name:       "obj1",
			Type:       "object",
			ObjectType: "keyword",
		},
		mapping.Field{
			Name:      "obj1-but-not-a-child-of-obj1",
			Dimension: &falsy,
		},
	}
)

func TestTimesSeriesIsDimension(t *testing.T) {
	processor := NewTimeSeriesProcessor(fields)

	tsProcessor := processor.(*timeseriesProcessor)
	for _, test := range []struct {
		isDim bool
		field string
	}{
		{true, "context.first"},
		{true, "context.second"},
		{false, "type-less"},
		{true, "context.third"},
		{false, "not-a-dimension"},
		{true, "dimension-by-default"},
		{false, "overwritten-field1"},
		{false, "overwritten-field2"},
		{true, "nested-obj.object-of-keywords.third-level"},
		{true, "nested-obj.wildcard-object-of-keywords.third-level"},
		{false, "nested-obj.second-level"},
		{true, "obj1.key1"},
		{false, "obj1-but-not-a-child-of-obj1.key1"},
	} {
		assert.Equal(t, test.isDim, tsProcessor.isDimension(test.field), test.field)
	}

}

func TestTimesSeriesHashes(t *testing.T) {
	timeseriesProcessor := NewTimeSeriesProcessor(fields)

	for _, test := range []struct {
		name     string
		in       common.MapStr
		expected common.MapStr
	}{
		{
			name: "simple fields",
			in: common.MapStr{
				"context": common.MapStr{
					"first":  1,
					"second": "word2",
					"third":  "word3",
				},
			},
			expected: common.MapStr{
				"context": common.MapStr{
					"first":  1,
					"second": "word2",
					"third":  "word3",
				},
				"timeseries": common.MapStr{"instance": uint64(10259802856000774733)},
			},
		},
		{
			name: "simple field - with one ignored field",
			in: common.MapStr{
				"context": common.MapStr{
					"first":  1,
					"second": "word2",
					"third":  "word3",
				},
				"not-a-dimension": 1000,
			},
			expected: common.MapStr{
				"context": common.MapStr{
					"first":  1,
					"second": "word2",
					"third":  "word3",
				},
				"not-a-dimension": 1000,
				"timeseries":      common.MapStr{"instance": uint64(10259802856000774733)}, // same as above
			},
		},
		{
			name: "simple fields and one ignored and one by default",
			in: common.MapStr{
				"context": common.MapStr{
					"first":  1,
					"second": "word2",
					"third":  "word3",
				},
				"not-a-dimension":      1000,
				"dimension-by-default": "dimension1",
			},
			expected: common.MapStr{
				"context": common.MapStr{
					"first":  1,
					"second": "word2",
					"third":  "word3",
				},
				"not-a-dimension":      1000,
				"dimension-by-default": "dimension1",
				"timeseries":           common.MapStr{"instance": uint64(17933311421196639387)},
			},
		},
	} {

		event := beat.Event{
			TimeSeries: true,
			Fields:     test.in,
		}
		t.Run(test.name, func(t *testing.T) {
			out, err := timeseriesProcessor.Run(&event)

			assert.NoError(t, err)
			assert.Equal(t, test.expected, out.Fields)
		})
	}
}
