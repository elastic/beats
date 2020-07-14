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

package dissect

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name        string
		p           *parser
		expectError bool
	}{
		{
			name: "when we find reference field for all indirect field",
			p: &parser{
				fields:          []field{newIndirectField(1, "hello", "", 0), newNormalField(0, "hola", "", 1, 0, false)},
				referenceFields: []field{newPointerField(2, "hello", 0)},
			},
			expectError: false,
		},
		{
			name: "when we cannot find all the reference field for all indirect field",
			p: &parser{
				fields:          []field{newIndirectField(1, "hello", "", 0), newNormalField(0, "hola", "", 1, 0, false)},
				referenceFields: []field{newPointerField(2, "okhello", 0)},
			},
			expectError: true,
		},
	}

	for _, test := range tests {
		err := validate(test.p)
		assert.Equal(t, test.expectError, err != nil)
	}
}
