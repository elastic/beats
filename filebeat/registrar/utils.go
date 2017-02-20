package registrar

import (
	"github.com/elastic/beats/filebeat/input/file"
	"strings"
)

func dirname(source string) string {
	i := strings.LastIndex(source, "/")
	return source[0:i]
}

type DirFilesState struct {
	states []file.State
}

func (d *DirFilesState) Add(state file.State) {
	d.states = append(d.states, state)
}

func (d *DirFilesState) Len() int {
	return len(d.states)
}

func (d *DirFilesState) Remove(del file.State) {
	oldLen := len(d.states)
	for i, state := range d.states {
		if state.Source == del.Source {
			for j := i; j < oldLen-1; j++ {
				d.states[j] = d.states[j+1]
			}
			d.states = d.states[0 : oldLen-1]
			return
		}
	}
}
