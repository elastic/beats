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

package tcp

import (
	"bufio"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCustomDelimiter(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		expected  []string
		delimiter []byte
	}{
		{
			name: "Multiple chars delimiter",
			text: "hello<END>bonjour<END>hola<END>hey",
			expected: []string{
				"hello",
				"bonjour",
				"hola",
				"hey",
			},
			delimiter: []byte("<END>"),
		},
		{
			name: "Multiple chars delimiter with half starting delimiter",
			text: "hello<END>bonjour<ENDhola<END>hey",
			expected: []string{
				"hello",
				"bonjour<ENDhola",
				"hey",
			},
			delimiter: []byte("<END>"),
		},
		{
			name: "Multiple chars delimiter with half ending delimiter",
			text: "hello<END>END>hola<END>hey",
			expected: []string{
				"hello",
				"END>hola",
				"hey",
			},
			delimiter: []byte("<END>"),
		},
		{
			name: "Delimiter end of string",
			text: "hello<END>bonjour<END>hola<END>hey<END>",
			expected: []string{
				"hello",
				"bonjour",
				"hola",
				"hey",
			},
			delimiter: []byte("<END>"),
		},
		{
			name: "Single char delimiter",
			text: "hello;bonjour;hola;hey",
			expected: []string{
				"hello",
				"bonjour",
				"hola",
				"hey",
			},
			delimiter: []byte(";"),
		},
		{
			name:      "Empty string",
			text:      "",
			expected:  []string(nil),
			delimiter: []byte(";"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			buf := strings.NewReader(test.text)
			scanner := bufio.NewScanner(buf)
			scanner.Split(factoryDelimiter(test.delimiter))
			var elements []string
			for scanner.Scan() {
				elements = append(elements, scanner.Text())
			}
			assert.EqualValues(t, test.expected, elements)
		})
	}
}
