// +build !integration

package log

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/filebeat/input/file"
	"github.com/elastic/beats/libbeat/common/match"
)

func TestInputFileExclude(t *testing.T) {
	p := Input{
		config: config{
			ExcludeFiles: []match.Matcher{match.MustCompile(`\.gz$`)},
		},
	}

	assert.True(t, p.isFileExcluded("/tmp/log/logw.gz"))
	assert.False(t, p.isFileExcluded("/tmp/log/logw.log"))
}

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

		l := Input{
			config: config{
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

func TestMatchesMeta(t *testing.T) {
	tests := []struct {
		Input  *Input
		Meta   map[string]string
		Result bool
	}{
		{
			Input: &Input{
				meta: map[string]string{
					"it": "matches",
				},
			},
			Meta: map[string]string{
				"it": "matches",
			},
			Result: true,
		},
		{
			Input: &Input{
				meta: map[string]string{
					"it":     "doesnt",
					"doesnt": "match",
				},
			},
			Meta: map[string]string{
				"it": "doesnt",
			},
			Result: false,
		},
		{
			Input: &Input{
				meta: map[string]string{
					"it": "doesnt",
				},
			},
			Meta: map[string]string{
				"it":     "doesnt",
				"doesnt": "match",
			},
			Result: false,
		},
		{
			Input: &Input{
				meta: map[string]string{},
			},
			Meta:   map[string]string{},
			Result: true,
		},
	}

	for _, test := range tests {
		assert.Equal(t, test.Result, test.Input.matchesMeta(test.Meta))
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
