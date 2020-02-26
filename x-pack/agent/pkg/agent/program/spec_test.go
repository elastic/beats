// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package program

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	"github.com/elastic/beats/x-pack/agent/pkg/agent/transpiler"
)

func TestSerialization(t *testing.T) {
	spec := Spec{
		Name:         "hello",
		Cmd:          "hellocmd",
		Configurable: "file",
		Args:         []string{"-c", "first"},
		Rules: transpiler.NewRuleList(
			transpiler.Copy("inputs", "filebeat"),
			transpiler.Filter("filebeat", "output", "keystore"),
			transpiler.Rename("filebeat", "notfilebeat"),
			transpiler.Translate("type", map[string]interface{}{
				"event/file":  "log",
				"event/stdin": "stdin",
			}),
			transpiler.TranslateWithRegexp("type", regexp.MustCompile("^metric/(.+)"), "$1/hello"),
			transpiler.Map("inputs",
				transpiler.Translate("type", map[string]interface{}{
					"event/file": "log",
				})),
			transpiler.FilterValues(
				"inputs",
				"type",
				"log",
			),
		),
		When: "1 == 1",
	}
	yml := `name: hello
cmd: hellocmd
configurable: file
args:
- -c
- first
rules:
- copy:
    from: inputs
    to: filebeat
- filter:
    selectors:
    - filebeat
    - output
    - keystore
- rename:
    from: filebeat
    to: notfilebeat
- translate:
    path: type
    mapper:
      event/file: log
      event/stdin: stdin
- translate_with_regexp:
    path: type
    re: ^metric/(.+)
    with: $1/hello
- map:
    path: inputs
    rules:
    - translate:
        path: type
        mapper:
          event/file: log
- filter_values:
    selector: inputs
    key: type
    values:
    - log
when: 1 == 1
`
	t.Run("serialization", func(t *testing.T) {
		b, err := yaml.Marshal(spec)
		require.NoError(t, err)
		assert.Equal(t, string(b), yml)
	})

	t.Run("deserialization", func(t *testing.T) {
		s := Spec{}
		err := yaml.Unmarshal([]byte(yml), &s)
		require.NoError(t, err)
		assert.Equal(t, spec, s)
	})
}

func TestExport(t *testing.T) {
	dir, err := ioutil.TempDir("", "test_export")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	for _, spec := range Supported {
		b, err := yaml.Marshal(spec)
		require.NoError(t, err)
		err = ioutil.WriteFile(filepath.Join(dir, strings.ToLower(spec.Name)+".yml"), b, 0666)
		require.NoError(t, err)
	}
}
