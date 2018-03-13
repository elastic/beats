package common

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/go-ucfg/yaml"
)

func TestFieldsHasKey(t *testing.T) {
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

func TestDynamicYaml(t *testing.T) {
	tests := []struct {
		input  []byte
		output Field
		error  bool
	}{
		{
			input: []byte(`
name: test
dynamic: true`),
			output: Field{
				Name:    "test",
				Dynamic: DynamicType{true},
			},
		},
		{
			input: []byte(`
name: test
dynamic: "true"`),
			output: Field{
				Name:    "test",
				Dynamic: DynamicType{true},
			},
		},
		{
			input: []byte(`
name: test
dynamic: "blue"`),
			error: true,
		},
		{
			input: []byte(`
name: test
dynamic: "strict"`),
			output: Field{
				Name:    "test",
				Dynamic: DynamicType{"strict"},
			},
		},
	}

	for _, test := range tests {
		keys := Field{}

		cfg, err := yaml.NewConfig(test.input)
		assert.NoError(t, err)
		err = cfg.Unpack(&keys)

		if err != nil {
			assert.True(t, test.error)
		} else {
			assert.Equal(t, test.output.Dynamic, keys.Dynamic)
		}
	}
}

func TestGetKeys(t *testing.T) {
	tests := []struct {
		fields Fields
		keys   []string
	}{
		{
			fields: Fields{
				Field{
					Name: "test", Fields: Fields{
						Field{
							Name: "find",
						},
					},
				},
			},
			keys: []string{"test.find"},
		},
		{
			fields: Fields{
				Field{
					Name: "a", Fields: Fields{
						Field{
							Name: "b",
						},
					},
				},
				Field{
					Name: "a", Fields: Fields{
						Field{
							Name: "c",
						},
					},
				},
			},
			keys: []string{"a.b", "a.c"},
		},
		{
			fields: Fields{
				Field{
					Name: "a",
				},
				Field{
					Name: "b",
				},
				Field{
					Name: "c",
				},
			},
			keys: []string{"a", "b", "c"},
		},
	}

	for _, test := range tests {
		assert.Equal(t, test.keys, test.fields.GetKeys())
	}
}
