package timeseries

import (
	"testing"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	truthy = true
	falsy  = false

	fields = common.Fields{
		common.Field{Name: "context.first", Type: "long", Description: "a dimension", Dimension: &truthy},
		common.Field{
			Name: "context",
			Type: "group",
			Fields: common.Fields{
				common.Field{Name: "second", Type: "keyword", Dimension: &truthy},
			},
		},
		common.Field{
			Name: "context",
			Type: "group",
			Fields: common.Fields{
				common.Field{Name: "third", Dimension: &truthy},
			},
		},
		common.Field{Name: "type-less"},
		common.Field{Name: "not-a-dimension", Type: "long"},
		common.Field{Name: "dimension-by-default", Type: "keyword"},
		common.Field{Name: "overwritten-field1", Type: "long", Dimension: &truthy},
		common.Field{Name: "overwritten-field1", Overwrite: true, Type: "long", Dimension: &falsy},
		common.Field{Name: "overwritten-field2", Overwrite: true, Type: "long"},
		common.Field{Name: "overwritten-field2", Type: "keyword", Dimension: &truthy},
		common.Field{
			Name: "nested-obj",
			Type: "object",
			Fields: common.Fields{
				common.Field{
					Name:       "object-of-keywords",
					Type:       "object",
					ObjectType: "keyword",
				},
				// todo: not supported
				common.Field{
					Name: "obj-type-params",
					ObjectTypeParams: []common.ObjectTypeCfg{
						{ObjectType: "keyword"},
						{ObjectType: "boolean"},
					},
					Type: "object",
				},
				common.Field{Name: "not-a-dimension", Type: "long"},
			},
		},
		common.Field{
			Name:       "obj1",
			Type:       "object",
			ObjectType: "keyword",
		},
		common.Field{
			Name:      "obj1-but-not-a-child-of-obj1",
			Dimension: &falsy,
		},
	}
)

func TestTimesSeriesIsDimension(t *testing.T) {
	processor, err := NewTimeSeriesProcessor(fields)
	require.NoError(t, err)

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
		{false, "nested-obj.second-level"},
		{true, "obj1.key1"},
		{false, "obj1-but-not-a-child-of-obj1.key1"},
	} {
		assert.Equal(t, test.isDim, tsProcessor.isDimension(test.field), test.field)
	}

}

func TestTimesSeriesHashes(t *testing.T) {
	timeseriesProcessor, err := NewTimeSeriesProcessor(fields)
	require.NoError(t, err)

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
