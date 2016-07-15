package schema

import (
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/stretchr/testify/assert"
)

func nop(key string, data map[string]interface{}) (interface{}, error) {
	return data[key], nil
}

func TestSchema(t *testing.T) {
	schema := Schema{
		"test": Conv{Key: "test", Func: nop},
		"test_obj": Object{
			"test_a": Conv{Key: "testA", Func: nop},
			"test_b": Conv{Key: "testB", Func: nop},
		},
	}

	source := map[string]interface{}{
		"test":      "hello",
		"testA":     "helloA",
		"testB":     "helloB",
		"other_key": "meh",
	}

	event := schema.Apply(source)
	assert.Equal(t, event, common.MapStr{
		"test": "hello",
		"test_obj": common.MapStr{
			"test_a": "helloA",
			"test_b": "helloB",
		},
	})
}

func test(key string, opts ...SchemaOption) Conv {
	return SetOptions(Conv{Key: key, Func: nop}, opts)
}

func TestOptions(t *testing.T) {
	conv := test("test", Optional)
	assert.Equal(t, conv.Key, "test")
	assert.Equal(t, conv.Optional, true)
}
