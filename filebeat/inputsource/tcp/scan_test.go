package tcp

import (
	"bufio"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCustomDelimiter(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		expected  []string
		delimiter []byte
	}{
		{
			name: "Multiple chars delimiter",
			text: "hello<END>bonjour<END>hola<END>hey",
			expected: []string{
				"hello",
				"bonjour",
				"hola",
				"hey",
			},
			delimiter: []byte("<END>"),
		},
		{
			name: "Multiple chars delimiter with half starting delimiter",
			text: "hello<END>bonjour<ENDhola<END>hey",
			expected: []string{
				"hello",
				"bonjour<ENDhola",
				"hey",
			},
			delimiter: []byte("<END>"),
		},
		{
			name: "Multiple chars delimiter with half ending delimiter",
			text: "hello<END>END>hola<END>hey",
			expected: []string{
				"hello",
				"END>hola",
				"hey",
			},
			delimiter: []byte("<END>"),
		},
		{
			name: "Delimiter end of string",
			text: "hello<END>bonjour<END>hola<END>hey<END>",
			expected: []string{
				"hello",
				"bonjour",
				"hola",
				"hey",
			},
			delimiter: []byte("<END>"),
		},
		{
			name: "Single char delimiter",
			text: "hello;bonjour;hola;hey",
			expected: []string{
				"hello",
				"bonjour",
				"hola",
				"hey",
			},
			delimiter: []byte(";"),
		},
		{
			name:      "Empty string",
			text:      "",
			expected:  []string(nil),
			delimiter: []byte(";"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			buf := strings.NewReader(test.text)
			scanner := bufio.NewScanner(buf)
			scanner.Split(factoryDelimiter(test.delimiter))
			var elements []string
			for scanner.Scan() {
				elements = append(elements, scanner.Text())
			}
			assert.EqualValues(t, test.expected, elements)
		})
	}
}
