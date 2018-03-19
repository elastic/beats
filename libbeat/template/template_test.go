// +build !integration

package template

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
)

func TestNumberOfRoutingShards(t *testing.T) {

	beatVersion := "6.1.0"
	beatName := "testbeat"
	config := TemplateConfig{}

	// Test it exists in 6.1
	template, err := New(beatVersion, beatName, "6.1.0", config)
	assert.NoError(t, err)

	data := template.generate(nil, nil)
	shards, err := data.GetValue("settings.index.number_of_routing_shards")
	assert.NoError(t, err)

	assert.Equal(t, 30, shards.(int))

	// Test it does not exist in 6.0
	template, err = New(beatVersion, beatName, "6.0.0", config)
	assert.NoError(t, err)

	data = template.generate(nil, nil)
	shards, err = data.GetValue("settings.index.number_of_routing_shards")
	assert.Error(t, err)
	assert.Equal(t, nil, shards)
}

func TestNumberOfRoutingShardsOverwrite(t *testing.T) {

	beatVersion := "6.1.0"
	beatName := "testbeat"
	config := TemplateConfig{
		Settings: TemplateSettings{
			Index: map[string]interface{}{"number_of_routing_shards": 5},
		},
	}

	// Test it exists in 6.1
	template, err := New(beatVersion, beatName, "6.1.0", config)
	assert.NoError(t, err)

	data := template.generate(nil, nil)
	shards, err := data.GetValue("settings.index.number_of_routing_shards")
	assert.NoError(t, err)

	assert.Equal(t, 5, shards.(int))
}

func TestAppendFields(t *testing.T) {
	tests := []struct {
		fields       common.Fields
		appendFields common.Fields
		error        bool
	}{
		{
			fields: common.Fields{
				common.Field{
					Name: "a",
					Fields: common.Fields{
						common.Field{
							Name: "b",
						},
					},
				},
			},
			appendFields: common.Fields{
				common.Field{
					Name: "a",
					Fields: common.Fields{
						common.Field{
							Name: "c",
						},
					},
				},
			},
			error: false,
		},
		{
			fields: common.Fields{
				common.Field{
					Name: "a",
					Fields: common.Fields{
						common.Field{
							Name: "b",
						},
						common.Field{
							Name: "c",
						},
					},
				},
			},
			appendFields: common.Fields{
				common.Field{
					Name: "a",
					Fields: common.Fields{
						common.Field{
							Name: "c",
						},
					},
				},
			},
			error: true,
		},
		{
			fields: common.Fields{
				common.Field{
					Name: "a",
				},
			},
			appendFields: common.Fields{
				common.Field{
					Name: "a",
					Fields: common.Fields{
						common.Field{
							Name: "c",
						},
					},
				},
			},
			error: true,
		},
		{
			fields: common.Fields{
				common.Field{
					Name: "a",
					Fields: common.Fields{
						common.Field{
							Name: "c",
						},
					},
				},
			},
			appendFields: common.Fields{
				common.Field{
					Name: "a",
				},
			},
			error: true,
		},
	}

	for _, test := range tests {
		_, err := appendFields(test.fields, test.appendFields)
		if test.error {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
	}
}
