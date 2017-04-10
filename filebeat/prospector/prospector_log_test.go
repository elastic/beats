// +build !integration

package prospector

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/elastic/beats/filebeat/input/file"
	"github.com/stretchr/testify/assert"
)

var cleanInactiveTests = []struct {
	cleanInactive time.Duration
	fileTime      time.Time
	result        bool
}{
	{
		cleanInactive: 0,
		fileTime:      time.Now(),
		result:        false,
	},
	{
		cleanInactive: 1 * time.Second,
		fileTime:      time.Now().Add(-5 * time.Second),
		result:        true,
	},
	{
		cleanInactive: 10 * time.Second,
		fileTime:      time.Now().Add(-5 * time.Second),
		result:        false,
	},
}

func TestIsCleanInactive(t *testing.T) {

	for _, test := range cleanInactiveTests {

		l := Log{
			config: prospectorConfig{
				CleanInactive: test.cleanInactive,
			},
		}
		state := file.State{
			Fileinfo: TestFileInfo{
				time: test.fileTime,
			},
		}

		assert.Equal(t, test.result, l.isCleanInactive(state))
	}
}

type TestFileInfo struct {
	time time.Time
}

func (t TestFileInfo) Name() string       { return "" }
func (t TestFileInfo) Size() int64        { return 0 }
func (t TestFileInfo) Mode() os.FileMode  { return 0 }
func (t TestFileInfo) ModTime() time.Time { return t.time }
func (t TestFileInfo) IsDir() bool        { return false }
func (t TestFileInfo) Sys() interface{}   { return nil }

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
			},
		},
	}
	root, err := ioutil.TempDir("", "testglob")
	if err != nil {
		t.Fatal(err)
	}
	os.MkdirAll(filepath.Join(root, "foo/bar/baz"), 0755)
	for _, test := range tests {
		matches, err := glob(filepath.Join(root, test.pattern))
		if err != nil {
			t.Error(err)
			continue
		}
		var normalizedMatches []string
		for _, m := range matches {
			normalizedMatches = append(normalizedMatches, m[len(root)+1:])
		}
		matchError := func() {
			t.Errorf("Pattern %q matched %q instead of %q", test.pattern, normalizedMatches, test.expectedMatches)
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
