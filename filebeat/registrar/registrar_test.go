// +build !windows,!integration

package registrar

import (
	"sort"
	"testing"

	"github.com/elastic/beats/filebeat/input/file"
	"github.com/stretchr/testify/assert"
)

func TestConvertOldStates(t *testing.T) {
	type io struct {
		Name   string
		Input  map[string]file.State
		Output []string
	}
	tests := []io{
		{
			Name: "Simple test with three files",
			Input: map[string]file.State{
				"test":  {Source: "test", FileStateOS: file.StateOS{Inode: 5}},
				"test1": {Source: "test1", FileStateOS: file.StateOS{Inode: 3}},
				"test2": {Source: "test2", FileStateOS: file.StateOS{Inode: 2}},
			},
			Output: []string{"test", "test1", "test2"},
		},
		{
			Name: "De-duplicate inodes. Bigger offset wins (1)",
			Input: map[string]file.State{
				"test":  {Source: "test", FileStateOS: file.StateOS{Inode: 2}},
				"test1": {Source: "test1", FileStateOS: file.StateOS{Inode: 3}},
				"test2": {Source: "test2", FileStateOS: file.StateOS{Inode: 2}, Offset: 2},
			},
			Output: []string{"test1", "test2"},
		},
		{
			Name: "De-duplicate inodes. Bigger offset wins (2)",
			Input: map[string]file.State{
				"test":  {Source: "test", FileStateOS: file.StateOS{Inode: 2}, Offset: 2},
				"test1": {Source: "test1", FileStateOS: file.StateOS{Inode: 3}},
				"test2": {Source: "test2", FileStateOS: file.StateOS{Inode: 2}, Offset: 0},
			},
			Output: []string{"test", "test1"},
		},
	}

	for _, test := range tests {
		result := convertOldStates(test.Input)
		resultSources := []string{}
		for _, state := range result {
			resultSources = append(resultSources, state.Source)
		}
		sort.Strings(resultSources)
		assert.Equal(t, test.Output, resultSources, test.Name)
	}
}
