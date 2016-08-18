package mapstriface

import (
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/metricbeat/schema"
	"github.com/stretchr/testify/assert"
)

func TestConversions(t *testing.T) {
	ts := time.Now()

	input := map[string]interface{}{
		"testString":       "hello",
		"testInt":          42,
		"testIntFromFloat": 42.0,
		"testIntFromInt64": int64(42),
		"testBool":         true,
		"testObj": map[string]interface{}{
			"testObjString": "hello, object",
		},
		"testNonNestedObj": "hello from top level",
		"testTime":         ts,

		// wrong types
		"testErrorInt":    "42",
		"testErrorTime":   12,
		"testErrorBool":   "false",
		"testErrorString": 32,
	}

	schema := s.Schema{
		"test_string":         Str("testString"),
		"test_int":            Int("testInt"),
		"test_int_from_float": Int("testIntFromFloat"),
		"test_int_from_int64": Int("testIntFromInt64"),
		"test_bool":           Bool("testBool"),
		"test_time":           Time("testTime"),
		"test_obj_1": s.Object{
			"test": Str("testNonNestedObj"),
		},
		"test_obj_2": Dict("testObj", s.Schema{
			"test": Str("testObjString"),
		}),
		"test_error_int":    Int("testErrorInt", s.Optional),
		"test_error_time":   Time("testErrorTime", s.Optional),
		"test_error_bool":   Bool("testErrorBool", s.Optional),
		"test_error_string": Str("testErrorString", s.Optional),
	}

	expected := common.MapStr{
		"test_string":         "hello",
		"test_int":            int64(42),
		"test_int_from_float": int64(42),
		"test_int_from_int64": int64(42),
		"test_bool":           true,
		"test_time":           common.Time(ts),
		"test_obj_1": common.MapStr{
			"test": "hello from top level",
		},
		"test_obj_2": common.MapStr{
			"test": "hello, object",
		},
	}

	output := schema.Apply(input)
	assert.Equal(t, output, expected)
}
