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

package cfgtype

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnpack(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		expected ByteSize
	}{
		{
			name:     "friendly human value",
			s:        "1KiB",
			expected: ByteSize(1024),
		},
		{
			name:     "raw bytes",
			s:        "2024",
			expected: ByteSize(2024),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s := ByteSize(0)
			err := s.Unpack(test.s)
			if !assert.NoError(t, err) {
				return
			}
			assert.Equal(t, test.expected, s)
		})
	}
}
