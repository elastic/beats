// +build !integration

package cfgfile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestConfig struct {
	Output ElasticsearchConfig
}

type ElasticsearchConfig struct {
	Elasticsearch Connection
}

type Connection struct {
	Port int
	Host string
}

func TestRead(t *testing.T) {
	absPath, err := filepath.Abs("../tests/files/")

	assert.NotNil(t, absPath)
	assert.Nil(t, err)

	config := &TestConfig{}

	err = Read(config, absPath+"/config.yml")
	assert.Nil(t, err)

	// Access config
	assert.Equal(t, "localhost", config.Output.Elasticsearch.Host)

	// Chat that it is integer
	assert.Equal(t, 9200, config.Output.Elasticsearch.Port)
}

func TestExpandEnv(t *testing.T) {
	var tests = []struct {
		in  string
		out string
		err string
	}{
		// Environment variables can be specified as ${env} only.
		{"${y}", "y", ""},
		{"$y", "$y", ""},

		// Environment variables are case-sensitive.
		{"${Y}", "", ""},

		// Defaults can be specified.
		{"x${Z:D}", "xD", ""},
		{"x${Z:A B C D}", "xA B C D", ""}, // Spaces are allowed in the default.
		{"x${Z:}", "x", ""},

		// Un-matched braces cause an error.
		{"x${Y ${Z:Z}", "", "unexpected character in variable expression: " +
			"U+0020 ' ', expected a default value or closing brace"},

		// Special environment variables are not replaced.
		{"$*", "$*", ""},
		{"${*}", "", "shell variable cannot start with U+002A '*'"},
		{"$@", "$@", ""},
		{"${@}", "", "shell variable cannot start with U+0040 '@'"},
		{"$1", "$1", ""},
		{"${1}", "", "shell variable cannot start with U+0031 '1'"},

		{"", "", ""},
		{"$$", "$$", ""},

		{"${a_b}", "", ""}, // Underscores are allowed in variable names.

		// ${} cannot be split across newlines.
		{"hello ${name: world\n}", "", "unterminated brace"},

		// To use a literal '${' you write '$${'.
		{`password: "abc$${!"`, `password: "abc${!"`, ""},

		// The full error contains the line number.
		{"shipper:\n  name: ${var", "", "failure while expanding environment " +
			"variables in config.yml at line=2, unterminated brace"},
	}

	for _, test := range tests {
		os.Setenv("y", "y")
		output, err := expandEnv("config.yml", []byte(test.in))

		switch {
		case test.err != "" && err == nil:
			t.Errorf("Expected an error for test case %+v", test)
		case test.err == "" && err != nil:
			t.Errorf("Unexpected error for test case %+v, %v", test, err)
		case err != nil:
			assert.Contains(t, err.Error(), test.err)
		default:
			assert.Equal(t, test.out, string(output), "Input: %s", test.in)
		}
	}
}
