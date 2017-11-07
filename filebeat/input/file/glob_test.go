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
