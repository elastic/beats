package common

import (
	"testing"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/stretchr/testify/assert"
)

func TestConvertNestedMapStr(t *testing.T) {
	logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"*"})

	type io struct {
		Input  MapStr
		Output MapStr
	}

	type String string

	tests := []io{
		io{
			Input: MapStr{
				"key": MapStr{
					"key1": "value1",
				},
			},
			Output: MapStr{
				"key": MapStr{
					"key1": "value1",
				},
			},
		},
		io{
			Input: MapStr{
				"key": MapStr{
					"key1": String("value1"),
				},
			},
			Output: MapStr{
				"key": MapStr{
					"key1": String("value1"),
				},
			},
		},
		io{
			Input: MapStr{
				"key": MapStr{
					"key1": []string{"value1", "value2"},
				},
			},
			Output: MapStr{
				"key": MapStr{
					"key1": []string{"value1", "value2"},
				},
			},
		},
		io{
			Input: MapStr{
				"key": MapStr{
					"key1": []String{"value1", "value2"},
				},
			},
			Output: MapStr{
				"key": MapStr{
					"key1": []String{"value1", "value2"},
				},
			},
		},
		io{
			Input: MapStr{
				"@timestamp": MustParseTime("2015-03-01T12:34:56.123Z"),
			},
			Output: MapStr{
				"@timestamp": MustParseTime("2015-03-01T12:34:56.123Z"),
			},
		},
	}

	for _, test := range tests {
		assert.EqualValues(t, test.Output, ConvertToGenericEvent(test.Input))
	}

}

func TestConvertNestedStruct(t *testing.T) {
	logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"*"})

	type io struct {
		Input  MapStr
		Output MapStr
	}

	type TestStruct struct {
		A string
		B int
	}

	tests := []io{
		io{
			Input: MapStr{
				"key": MapStr{
					"key1": TestStruct{
						A: "hello",
						B: 5,
					},
				},
			},
			Output: MapStr{
				"key": MapStr{
					"key1": MapStr{
						"A": "hello",
						"B": float64(5),
					},
				},
			},
		},
	}

	for _, test := range tests {
		assert.EqualValues(t, test.Output, ConvertToGenericEvent(test.Input))
	}

}
