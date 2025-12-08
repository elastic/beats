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

package file

import (
	"testing"
)

type globPatternsTest struct {
	pattern          string
	expectedPatterns []string
	expectedError    bool
}

func TestGlobPatterns(t *testing.T) {
	for _, test := range globPatternsTests {
		patterns, err := GlobPatterns(test.pattern, 2)
		if err != nil {
			if test.expectedError {
				continue
			}
			t.Fatal(err)
		}
		if len(patterns) != len(test.expectedPatterns) {
			t.Fatalf("%q expanded to %q (%d) instead of %q (%d)", test.pattern, patterns, len(patterns),
				test.expectedPatterns, len(test.expectedPatterns))
		}
		for i, p := range patterns {
			if p != test.expectedPatterns[i] {
				t.Fatalf("%q expanded to %q instead of %q", test.pattern, patterns, test.expectedPatterns)
			}
		}
	}
}
