// +build !integration

package prospector

import (
	"testing"

	"github.com/elastic/beats/libbeat/common/match"
	"github.com/stretchr/testify/assert"
)

var matchTestsWindows = []struct {
	file         string
	paths        []string
	excludeFiles []match.Matcher
	result       bool
}{
	{
		`C:\\hello\test\test.log`,
		[]string{`C:\\hello/test/*.log`},
		nil,
		true,
	},
	{
		`C:\\hello\test\test.log`,
		[]string{`C:\\hello\test/*.log`},
		nil,
		true,
	},
	{
		`C:\\hello\test\test.log`,
		[]string{`C://hello/test/*.log`},
		nil,
		true,
	},
	{
		`C:\\hello\test\test.log`,
		[]string{`C://hello//test//*.log`},
		nil,
		true,
	},
	{
		`C://hello/test/test.log`,
		[]string{`C:\\hello\test\*.log`},
		nil,
		true,
	},
	{
		`C://hello/test/test.log`,
		[]string{`C:/hello/test/*.log`},
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
