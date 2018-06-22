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

package mapstrstr

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
)

func TestConversions(t *testing.T) {
	input := map[string]interface{}{
		"testString":     "hello",
		"testInt":        "42",
		"testBool":       "true",
		"testFloat":      "42.1",
		"testObjString":  "hello, object",
		"testTime":       "2016-08-12T08:00:59.601478Z",
		"testError":      42,     // invalid, only strings are allowed
		"testErrorInt":   "12a",  // invalid integer
		"testErrorFloat": "12,2", // invalid float
		"testErrorBool":  "yes",  // invalid bool
	}

	schema := s.Schema{
		"test_string": Str("testString"),
		"test_int":    Int("testInt"),
		"test_bool":   Bool("testBool"),
		"test_float":  Float("testFloat"),
		"test_time":   Time(time.RFC3339Nano, "testTime"),
		"test_obj": s.Object{
			"test_obj_string": Str("testObjString"),
		},
		"test_notexistant": Str("notexistant", s.Optional),
		"test_error":       Str("testError", s.Optional),
		"test_error_int":   Int("testErrorInt", s.Optional),
		"test_error_float": Float("testErrorFloat", s.Optional),
		"test_error_bool":  Bool("testErrorBool", s.Optional),
	}

	ts, err := time.Parse(time.RFC3339Nano, "2016-08-12T08:00:59.601478Z")
	assert.NoError(t, err)

	expected := common.MapStr{
		"test_string": "hello",
		"test_int":    int64(42),
		"test_bool":   true,
		"test_float":  42.1,
		"test_time":   common.Time(ts),
		"test_obj": common.MapStr{
			"test_obj_string": "hello, object",
		},
	}

	output, _ := schema.Apply(input)
	assert.Equal(t, output, expected)
}
