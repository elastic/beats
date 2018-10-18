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

package cli

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfirm(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		def    bool
		result bool
		error  bool
	}{
		{
			name:   "Test default yes",
			input:  "\n",
			def:    true,
			result: true,
		},
		{
			name:   "Test default no",
			input:  "\n",
			def:    false,
			result: false,
		},
		{
			name:   "Test YeS",
			input:  "YeS\n",
			def:    false,
			result: true,
		},
		{
			name:   "Test Y",
			input:  "Y\n",
			def:    false,
			result: true,
		},
		{
			name:   "Test No",
			input:  "No\n",
			def:    true,
			result: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := strings.NewReader(test.input)
			result, err := confirm(r, "prompt", test.def)
			assert.Equal(t, test.result, result)

			if test.error {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
