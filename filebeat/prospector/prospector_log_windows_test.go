// +build windows

package prospector

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

var matchTestsWindows = []struct {
	file         string
	paths        []string
	excludeFiles []*regexp.Regexp
	result       bool
}{
	{
		"C:\\\\hello\\test\\test.log",      // Path are always in windows format
		[]string{"C:\\\\hello/test/*.log"}, // Globs can also be with forward slashes
		nil,
		true,
	},
	{
		"C:\\\\hello\\test\\test.log",       // Path are always in windows format
		[]string{"C:\\\\hello\\test/*.log"}, // Globs can also be mixed
		nil,
		true,
	},
}

// TestMatchFileWindows test if match works correctly on windows
// Separate test are needed on windows because of automated path conversion
func TestMatchFileWindows(t *testing.T) {

	for _, test := range matchTestsWindows {

		p := ProspectorLog{
			config: prospectorConfig{
				Paths:        test.paths,
				ExcludeFiles: test.excludeFiles,
			},
		}

		assert.Equal(t, test.result, p.matchesFile(test.file))
	}
}
