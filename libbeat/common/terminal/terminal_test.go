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

package terminal

import (
	"os"
	"testing"
)

func withStdinInput(t *testing.T, input string) {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating stdin pipe: %v", err)
	}

	_, err = w.WriteString(input)
	if err != nil {
		t.Fatalf("writing test input: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("closing stdin writer: %v", err)
	}

	originalStdin := os.Stdin
	os.Stdin = r
	t.Cleanup(func() {
		os.Stdin = originalStdin
		_ = r.Close()
	})
}

func TestReadInput(t *testing.T) {
	withStdinInput(t, "value\n")

	input, err := ReadInput()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if input != "value" {
		t.Fatalf("expected value, got %q", input)
	}
}

func TestPromptYesNo(t *testing.T) {
	tests := []struct {
		name          string
		defaultAnswer bool
		input         string
		expected      bool
	}{
		{
			name:          "returns default true on empty answer",
			defaultAnswer: true,
			input:         "\n",
			expected:      true,
		},
		{
			name:          "returns default false on empty answer",
			defaultAnswer: false,
			input:         "\n",
			expected:      false,
		},
		{
			name:          "accepts yes in mixed case",
			defaultAnswer: false,
			input:         "YeS\n",
			expected:      true,
		},
		{
			name:          "accepts no with surrounding spaces",
			defaultAnswer: true,
			input:         "  no  \n",
			expected:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withStdinInput(t, tt.input)

			answer := PromptYesNo("Continue?", tt.defaultAnswer)
			if answer != tt.expected {
				t.Fatalf("expected %v, got %v", tt.expected, answer)
			}
		})
	}
}
