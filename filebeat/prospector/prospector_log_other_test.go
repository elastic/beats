// +build !windows

package prospector

import (
	"regexp"
	"testing"

	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/filebeat/input/file"

	"github.com/stretchr/testify/assert"
)

var matchTests = []struct {
	file         string
	paths        []string
	excludeFiles []*regexp.Regexp
	result       bool
}{
	{
		"test/test.log",
		[]string{"test/*"},
		nil,
		true,
	},
	{
		"notest/test.log",
		[]string{"test/*"},
		nil,
		false,
	},
	{
		"test/test.log",
		[]string{"test/*.log"},
		nil,
		true,
	},
	{
		"test/test.log",
		[]string{"test/*.nolog"},
		nil,
		false,
	},
	{
		"test/test.log",
		[]string{"test/*"},
		[]*regexp.Regexp{regexp.MustCompile("test.log")},
		false,
	},
	{
		"test/test.log",
		[]string{"test/*"},
		[]*regexp.Regexp{regexp.MustCompile("test2.log")},
		true,
	},
}

func TestMatchFile(t *testing.T) {

	for _, test := range matchTests {

		p := ProspectorLog{
			config: prospectorConfig{
				Paths:        test.paths,
				ExcludeFiles: test.excludeFiles,
			},
		}

		assert.Equal(t, test.result, p.matchesFile(test.file))
	}
}

var initStateTests = []struct {
	states []file.State // list of states
	paths  []string     // prospector glob
	count  int          // expected states in prospector
}{
	{
		[]file.State{
			file.State{Source: "test"},
		},
		[]string{"test"},
		1,
	},
	{
		[]file.State{
			file.State{Source: "notest"},
		},
		[]string{"test"},
		0,
	},
	{
		[]file.State{
			file.State{Source: "test1.log", FileStateOS: file.StateOS{Inode: 1}},
			file.State{Source: "test2.log", FileStateOS: file.StateOS{Inode: 2}},
		},
		[]string{"*.log"},
		2,
	},
	{
		[]file.State{
			file.State{Source: "test1.log", FileStateOS: file.StateOS{Inode: 1}},
			file.State{Source: "test2.log", FileStateOS: file.StateOS{Inode: 2}},
		},
		[]string{"test1.log"},
		1,
	},
	{
		[]file.State{
			file.State{Source: "test1.log", FileStateOS: file.StateOS{Inode: 1}},
			file.State{Source: "test2.log", FileStateOS: file.StateOS{Inode: 2}},
		},
		[]string{"test.log"},
		0,
	},
	{
		[]file.State{
			file.State{Source: "test1.log", FileStateOS: file.StateOS{Inode: 1}},
			file.State{Source: "test2.log", FileStateOS: file.StateOS{Inode: 1}},
		},
		[]string{"*.log"},
		1, // Expecting only 1 state because of some inode (this is only a theoretical case)
	},
}

// TestInit checks that the correct states are in a prospector after the init phase
// This means only the ones that match the glob and not exclude files
func TestInit(t *testing.T) {

	for _, test := range initStateTests {
		p := ProspectorLog{
			Prospector: &Prospector{
				states: &file.States{},
				outlet: TestOutlet{},
			},
			config: prospectorConfig{
				Paths: test.paths,
			},
		}
		states := file.NewStates()
		states.SetStates(test.states)
		err := p.Init(*states)
		assert.NoError(t, err)
		assert.Equal(t, test.count, p.Prospector.states.Count())
	}

}

// TestOutlet is an empty outlet for testing
type TestOutlet struct{}

func (o TestOutlet) OnEvent(event *input.Event) bool { return true }
