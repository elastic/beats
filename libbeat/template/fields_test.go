package template

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
)

func TestHasKey(t *testing.T) {
	tests := []struct {
		key    string
		fields Fields
		result bool
	}{
		{
			key:    "test.find",
			fields: Fields{},
			result: false,
		},
		{
			key: "test.find",
			fields: Fields{
				Field{Name: "test"},
				Field{Name: "find"},
			},
			result: false,
		},
		{
			key: "test.find",
			fields: Fields{
				Field{
					Name: "test", Fields: Fields{
						Field{
							Name: "find",
						},
					},
				},
			},
			result: true,
		},
		{
			key: "test",
			fields: Fields{
				Field{
					Name: "test", Fields: Fields{
						Field{
							Name: "find",
						},
					},
				},
			},
			result: false,
		},
	}

	for _, test := range tests {
		assert.Equal(t, test.result, test.fields.HasKey(test.key))
	}
}

func TestPropertiesCombine(t *testing.T) {
	// Test common fields are combined even if they come from different objects
	fields := Fields{
		Field{
			Name: "test",
			Type: "group",
			Fields: Fields{
				Field{
					Name: "one",
					Type: "text",
				},
			},
		},
		Field{
			Name: "test",
			Type: "group",
			Fields: Fields{
				Field{
					Name: "two",
					Type: "text",
				},
			},
		},
	}

	output := common.MapStr{}
	version, err := common.NewVersion("6.0.0")
	if err != nil {
		t.Fatal(err)
	}

	err = fields.process("", *version, output)
	if err != nil {
		t.Fatal(err)
	}

	v1, err := output.GetValue("test.properties.one")
	if err != nil {
		t.Fatal(err)
	}
	v2, err := output.GetValue("test.properties.two")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, v1, common.MapStr{"type": "text", "norms": false})
	assert.Equal(t, v2, common.MapStr{"type": "text", "norms": false})
}
