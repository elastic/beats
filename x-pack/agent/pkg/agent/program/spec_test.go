// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package program

import (
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	"github.com/elastic/beats/x-pack/agent/pkg/agent/internal/yamltest"
	"github.com/elastic/beats/x-pack/agent/pkg/agent/transpiler"
)

func TestConfiguration(t *testing.T) {
	testcases := map[string]struct {
		programs []string
		expected int
		err      bool
	}{
		"single_config": {
			programs: []string{"filebeat", "metricbeat"},
			expected: 2,
		},
		"audit_config": {
			programs: []string{"auditbeat"},
			expected: 1,
		},
		"journal_config": {
			programs: []string{"journalbeat"},
			expected: 1,
		},
		"monitor_config": {
			programs: []string{"heartbeat"},
			expected: 1,
		},
		"enabled_true": {
			programs: []string{"filebeat"},
			expected: 1,
		},
		"enabled_false": {
			expected: 0,
		},
		"enabled_output_true": {
			programs: []string{"filebeat"},
			expected: 1,
		},
		"enabled_output_false": {
			expected: 0,
		},
		"multiple_output_true": {
			err: true,
		},
	}

	for name, test := range testcases {
		t.Run(name, func(t *testing.T) {
			singleConfig, err := ioutil.ReadFile(filepath.Join("testdata", name+".yml"))
			require.NoError(t, err)

			var m map[string]interface{}
			err = yaml.Unmarshal(singleConfig, &m)
			require.NoError(t, err)

			ast, err := transpiler.NewAST(m)
			require.NoError(t, err)

			programs, err := Programs(ast)
			if test.err {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.expected, len(programs))

			for _, program := range programs {
				programConfig, err := ioutil.ReadFile(filepath.Join(
					"testdata",
					name+"-"+strings.ToLower(program.Spec.Name)+".yml",
				))

				require.NoError(t, err)
				var m map[string]interface{}
				err = yamltest.FromYAML(programConfig, &m)
				require.NoError(t, err)

				compareMap := &transpiler.MapVisitor{}
				program.Config.Accept(compareMap)

				if !assert.True(t, cmp.Equal(m, compareMap.Content)) {
					diff := cmp.Diff(m, compareMap.Content)
					if diff != "" {
						t.Errorf("%s-%s mismatch (-want +got):\n%s", name, program.Spec.Name, diff)
					}
				}
			}
		})
	}
}

func TestSerialization(t *testing.T) {
	spec := Spec{
		Name: "hello",
		Cmd:  "hellocmd",
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
	for _, spec := range Supported {
		b, err := yaml.Marshal(spec)
		require.NoError(t, err)
		ioutil.WriteFile("../../../spec/"+strings.ToLower(spec.Name)+".yml", b, 0666)
	}
}
