//+build !windows

package file

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestGlob(t *testing.T) {
	var tests = []struct {
		pattern         string
		expectedMatches []string
	}{
		{
			"*",
			[]string{
				"foo",
			},
		},
		{
			"foo/*",
			[]string{
				"foo/bar",
			},
		},
		{
			"*/*",
			[]string{
				"foo/bar",
			},
		},
		{
			"**",
			[]string{
				"foo",
				"foo/bar",
				"foo/bar/baz",
				"foo/bar/baz/qux",
			},
		},
		{
			"foo**",
			[]string{
				"foo",
			},
		},
		{
			"foo/**",
			[]string{
				"foo/bar",
				"foo/bar/baz",
				"foo/bar/baz/qux",
				"foo/bar/baz/qux/quux",
			},
		},
		{
			"foo/**/baz",
			[]string{
				"foo/bar/baz",
			},
		},
		{
			"foo/**/bazz",
			[]string{},
		},
		{
			"foo//bar",
			[]string{
				"foo/bar",
			},
		},
	}
	root, err := ioutil.TempDir("", "testglob")
	if err != nil {
		t.Fatal(err)
	}
	os.MkdirAll(filepath.Join(root, "foo/bar/baz/qux/quux"), 0755)
	for _, test := range tests {
		pattern := filepath.Join(root, test.pattern)
		matches, err := Glob(pattern, 4)
		if err != nil {
			t.Error(err)
			continue
		}
		var normalizedMatches []string
		for _, m := range matches {
			if len(m) < len(root)+1 {
				t.Fatalf("Matches are expected to be under %s and %q is not", root, m)
			}
			normalizedMatches = append(normalizedMatches, m[len(root)+1:])
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
