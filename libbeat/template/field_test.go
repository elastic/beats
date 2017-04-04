package template

import (
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/stretchr/testify/assert"
)

func TestField(t *testing.T) {

	esVersion2, err := NewVersion("2.0.0")
	assert.NoError(t, err)

	tests := []struct {
		field  Field
		method func(f Field) common.MapStr
		output common.MapStr
	}{
		{
			field:  Field{Type: "long"},
			method: func(f Field) common.MapStr { return f.other() },
			output: common.MapStr{
				"type": "long",
			},
		},
		{
			field:  Field{Type: "scaled_float"},
			method: func(f Field) common.MapStr { return f.scaledFloat() },
			output: common.MapStr{
				"type":           "scaled_float",
				"scaling_factor": 1000,
			},
		},
		{
			field:  Field{Type: "scaled_float", ScalingFactor: 100},
			method: func(f Field) common.MapStr { return f.scaledFloat() },
			output: common.MapStr{
				"type":           "scaled_float",
				"scaling_factor": 100,
			},
		},
		{
			field:  Field{Type: "scaled_float", esVersion: *esVersion2},
			method: func(f Field) common.MapStr { return f.scaledFloat() },
			output: common.MapStr{
				"type": "float",
			},
		},
	}

	for _, test := range tests {

		output := test.method(test.field)
		assert.Equal(t, test.output, output)
	}
}
