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

package aerospike

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	as "github.com/aerospike/aerospike-client-go"
)

func TestParseHost(t *testing.T) {
	tests := []struct {
		Name         string
		Host         string
		expectedHost *as.Host
		expectedErr  error
	}{
		{
			Name:         "with hostname and port",
			Host:         "localhost:3000",
			expectedHost: as.NewHost("localhost", 3000),
		},
		{
			Name:        "without port",
			Host:        "localhost",
			expectedErr: errors.New("Can't parse host localhost"),
		},
		{
			Name:        "with wrong port",
			Host:        "localhost:wrong",
			expectedErr: errors.New("Can't parse port: strconv.Atoi: parsing \"wrong\": invalid syntax"),
		},
	}

	for _, test := range tests {
		result, err := ParseHost(test.Host)
		if err != nil {
			if test.expectedErr != nil {
				assert.Equal(t, test.expectedErr.Error(), err.Error())
				continue
			}
			t.Error(err)
			continue
		}

		assert.Equal(t, test.expectedHost.String(), result.String(), test.Name)
	}
}

func TestParseInfo(t *testing.T) {
	tests := []struct {
		Name     string
		info     string
		expected map[string]interface{}
	}{
		{
			Name: "with kv",
			info: "key1=value1;key2=value2",
			expected: map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
			},
		},
		{
			Name:     "without kv",
			info:     "wrong result",
			expected: map[string]interface{}{},
		},
		{
			Name:     "mixed",
			info:     "wrong result;key=value",
			expected: map[string]interface{}{"key": "value"},
		},
	}

	for _, test := range tests {
		result := ParseInfo(test.info)
		assert.Equal(t, test.expected, result, test.Name)
	}
}
