package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_CSVDump(t *testing.T) {
	type io struct {
		Fields []string
		Rows   [][]string
		Output string
	}

	tests := []io{
		io{
			Fields: []string{"f1", "f2"},
			Rows: [][]string{
				[]string{"11", "12"},
				[]string{"21", "22"},
			},
			Output: "f1,f2\n11,12\n21,22\n",
		},
		io{
			Fields: []string{"f1", "f2"},
			Rows: [][]string{
				[]string{"11"},
				[]string{"21", "22", "23"},
			},
			Output: "f1,f2\n11\n21,22,23\n",
		},
		io{
			Fields: []string{"f\n\n1", "f\n2"},
			Rows: [][]string{
				[]string{"11"},
				[]string{"2\r\n1", "2\r\n2", "23"},
			},
			Output: "f\\n\\n1,f\\n2\n11\n2\\r\\n1,2\\r\\n2,23\n",
		},
	}

	for _, test := range tests {
		assert.Equal(t, test.Output, DumpInCSVFormat(test.Fields, test.Rows))
	}
}
