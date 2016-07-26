package reader

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnmarshal(t *testing.T) {
	type io struct {
		Name   string
		Input  string
		Output map[string]interface{}
	}

	tests := []io{
		io{
			Name:  "Top level int, float, string, bool",
			Input: `{"a": 3, "b": 2.0, "c": "hello", "d": true}`,
			Output: map[string]interface{}{
				"a": int64(3),
				"b": float64(2),
				"c": "hello",
				"d": true,
			},
		},
		io{
			Name:  "Nested objects with ints",
			Input: `{"a": 3, "b": {"c": {"d": 5}}}`,
			Output: map[string]interface{}{
				"a": int64(3),
				"b": map[string]interface{}{
					"c": map[string]interface{}{
						"d": int64(5),
					},
				},
			},
		},
		io{
			Name:  "Array of floats",
			Input: `{"a": 3, "b": {"c": [4.0, 4.1, 4.2]}}`,
			Output: map[string]interface{}{
				"a": int64(3),
				"b": map[string]interface{}{
					"c": []interface{}{
						float64(4.0), float64(4.1), float64(4.2),
					},
				},
			},
		},
		io{
			Name:  "Array of mixed ints and floats",
			Input: `{"a": 3, "b": {"c": [4, 4.1, 4.2]}}`,
			Output: map[string]interface{}{
				"a": int64(3),
				"b": map[string]interface{}{
					"c": []interface{}{
						int64(4), float64(4.1), float64(4.2),
					},
				},
			},
		},
		io{
			Name:  "Negative values",
			Input: `{"a": -3, "b": -1.0}`,
			Output: map[string]interface{}{
				"a": int64(-3),
				"b": float64(-1),
			},
		},
	}

	for _, test := range tests {
		t.Logf("Running test %s", test.Name)
		var output map[string]interface{}
		err := unmarshal([]byte(test.Input), &output)
		assert.NoError(t, err)
		assert.Equal(t, test.Output, output)
	}
}
