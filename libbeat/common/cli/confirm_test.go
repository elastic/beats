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
	tests := map[string]struct {
		input  string
		prompt string
		def    bool
		result bool
		error  bool
	}{
		"Test default yes": {
			input:  "\n",
			prompt: "> [Y/n]:",
			def:    true,
			result: true,
		},
		"Test default no": {
			input:  "\n",
			prompt: "> [y/N]:",
			def:    false,
			result: false,
		},
		"Test YeS": {
			input:  "YeS\n",
			prompt: "> [y/N]:",
			def:    false,
			result: true,
		},
		"Test Y": {
			input:  "Y\n",
			prompt: "> [y/N]:",
			def:    false,
			result: true,
		},
		"Test No": {
			input:  "No\n",
			def:    true,
			prompt: "> [Y/n]:",
			result: false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var buf strings.Builder
			r := strings.NewReader(test.input)

			result, err := confirm(r, &buf, ">", test.def)
			assert.Equal(t, test.result, result)

			if test.error {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, test.prompt, buf.String())
		})
	}
}
