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

package redis

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		Name  string
		Input redisConfig
		Valid bool
	}{
		{"No config", redisConfig{Key: "", Index: ""}, true},
		{"Only key", redisConfig{Key: "test", Index: ""}, true},
		{"Only index", redisConfig{Key: "", Index: "test"}, true},
		{"Both", redisConfig{Key: "test", Index: "test"}, true},

		{"Invalid Datatype", redisConfig{Key: "test", DataType: "something"}, false},
		{"List Datatype", redisConfig{Key: "test", DataType: "list"}, true},
		{"Channel Datatype", redisConfig{Key: "test", DataType: "channel"}, true},
	}

	for _, test := range tests {
		assert.Equal(t, test.Input.Validate() == nil, test.Valid, test.Name)
	}
}
