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
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

type globTest struct {
	pattern         string
	expectedMatches []string
}

func TestGlob(t *testing.T) {
	root, err := ioutil.TempDir("", "testglob")
	if err != nil {
		t.Fatal(err)
	}
	os.MkdirAll(filepath.Join(root, "foo/bar/baz/qux/quux"), 0755)
	for _, test := range globTests {
		pattern := filepath.Join(root, test.pattern)
		matches, err := Glob(pattern, 4)
		if err != nil {
			t.Fatal(err)
			continue
		}
		var normalizedMatches []string
		for _, m := range matches {
			if len(m) < len(root) {
				t.Fatalf("Matches for %q are expected to be under %s and %q is not", test.pattern, root, m)
			}
			var normalizedMatch string
			if len(m) > len(root) {
				normalizedMatch = m[len(root)+1:]
			} else {
				normalizedMatch = m[len(root):]
			}
			normalizedMatches = append(normalizedMatches, normalizedMatch)
		}
		matchError := func() {
			t.Fatalf("Pattern %q matched %q instead of %q", test.pattern, normalizedMatches, test.expectedMatches)
		}
		if len(normalizedMatches) != len(test.expectedMatches) {
			matchError()
			continue
		}
		for i, expectedMatch := range test.expectedMatches {
			if normalizedMatches[i] != expectedMatch {
				matchError()
			}
		}
	}
}

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
				break
			}
		}
	}
}
