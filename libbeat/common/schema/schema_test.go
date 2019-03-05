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

package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
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

	event, _ := schema.Apply(source)
	assert.Equal(t, event, common.MapStr{
		"test": "hello",
		"test_obj": common.MapStr{
			"test_a": "helloA",
			"test_b": "helloB",
		},
	})
}

func TestHasKey(t *testing.T) {
	schema := Schema{
		"test": Conv{Key: "Test", Func: nop},
		"test_obj": Object{
			"test_a": Conv{Key: "TestA", Func: nop},
			"test_b": Conv{Key: "TestB", Func: nop},
		},
	}

	assert.True(t, schema.HasKey("Test"))
	assert.True(t, schema.HasKey("TestA"))
	assert.True(t, schema.HasKey("TestB"))
	assert.False(t, schema.HasKey("test"))
	assert.False(t, schema.HasKey("test_obj"))
	assert.False(t, schema.HasKey("test_a"))
	assert.False(t, schema.HasKey("test_b"))
	assert.False(t, schema.HasKey("other"))
}

func test(key string, opts ...SchemaOption) Conv {
	return SetOptions(Conv{Key: key, Func: nop}, opts)
}

func TestOptions(t *testing.T) {
	conv := test("test", Optional)
	assert.Equal(t, conv.Key, "test")
	assert.Equal(t, conv.Optional, true)
}
