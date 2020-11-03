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

func TestReadInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
		res   string
	}{
		{
			name:  "Question 1?",
			input: "\n",
			res:   "",
		},
		{
			name:  "Question 2?",
			input: "full string input\n",
			res:   "full string input",
		},
		{
			name:  "Question 3?",
			input: "123456789\n",
			res:   "123456789",
		},
		{
			name:  "Question 4?",
			input: "false\n",
			res:   "false",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := strings.NewReader(test.input)
			result, err := input(r, test.name)
			assert.NoError(t, err)
			assert.Equal(t, test.res, result)
		})
	}
}
