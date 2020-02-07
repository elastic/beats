// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package transpiler

import (
	"regexp"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	"github.com/elastic/beats/x-pack/agent/pkg/agent/internal/yamltest"
)

func TestRules(t *testing.T) {
	testcases := map[string]struct {
		givenYAML    string
		expectedYAML string
		rule         Rule
	}{
		"inject index": {
			givenYAML: `
datasources:
  - name: All default
    inputs:
    - type: file
      streams:
        - paths: /var/log/mysql/error.log
  - name: Specified namespace
    namespace: nsns
    inputs:
    - type: file
      streams:
        - paths: /var/log/mysql/access.log
  - name: Specified dataset
    inputs:
    - type: file
      streams:
        - paths: /var/log/mysql/access.log
          dataset: dsds
  - name: All specified
    namespace: nsns
    inputs:
    - type: file
      streams:
        - paths: /var/log/mysql/access.log
          dataset: dsds
`,
			expectedYAML: `
datasources:
  - name: All default
    inputs:
    - type: file
      streams:
        - paths: /var/log/mysql/error.log
          index: mytype-default-generic
  - name: Specified namespace
    namespace: nsns
    inputs:
    - type: file
      streams:
        - paths: /var/log/mysql/access.log
          index: mytype-nsns-generic
  - name: Specified dataset
    inputs:
    - type: file
      streams:
        - paths: /var/log/mysql/access.log
          dataset: dsds
          index: mytype-default-dsds
  - name: All specified
    namespace: nsns
    inputs:
    - type: file
      streams:
        - paths: /var/log/mysql/access.log
          dataset: dsds
          index: mytype-nsns-dsds
`,
			rule: &RuleList{
				Rules: []Rule{
					InjectIndex("mytype"),
				},
			},
		},

		"extract items from array": {
			givenYAML: `
streams:
  - name: MySQL error log
    input:
      type:	file
      path:	/var/log/mysql/error.log
  - name: MySQL access log
    input:
      type:	file
      path:	/var/log/mysql/access.log
  - name: MySQL metrics
    input:
      type: mysql
      host: localhost
      port: 3306
`,
			expectedYAML: `
streams:
  - name: MySQL error log
    input:
      type:	file
      path:	/var/log/mysql/error.log
  - name: MySQL access log
    input:
      type:	file
      path: /var/log/mysql/access.log
  - name: MySQL metrics
    input:
      type: mysql
      host: localhost
      port: 3306
inputs:
  - type: file
    path: /var/log/mysql/error.log
  - type: file
    path: /var/log/mysql/access.log
  - type: mysql
    host: localhost
    port: 3306
`,
			rule: &RuleList{
				Rules: []Rule{
					ExtractListItem("streams", "input", "inputs"),
				},
			},
		},
		"two level rename": {
			givenYAML: `
output:
  elasticsearch:
    hosts:
      - "127.0.0.1:9201"
      - "127.0.0.1:9202"
  logstash:
    port: 5
`,
			expectedYAML: `
output:
  what:
    hosts:
      - "127.0.0.1:9201"
      - "127.0.0.1:9202"
  logstash:
    port: 5
`,
			rule: &RuleList{
				Rules: []Rule{
					Rename("output.elasticsearch", "what"),
				},
			},
		},
		"rename non existing key": {
			givenYAML: `
output:
  elasticsearch:
    hosts:
      - "127.0.0.1:9201"
      - "127.0.0.1:9202"
  logstash:
    port: 5
`,
			expectedYAML: `
output:
  elasticsearch:
    hosts:
      - "127.0.0.1:9201"
      - "127.0.0.1:9202"
  logstash:
    port: 5
`,
			rule: &RuleList{
				Rules: []Rule{
					Rename("donoexist", "what"),
				},
			},
		},
		"copy top level slice": {
			givenYAML: `
inputs:
  - type: event/file
  - type: metric/docker
`,
			expectedYAML: `
inputs:
  - type: event/file
  - type: metric/docker
filebeat:
  inputs:
    - type: event/file
    - type: metric/docker
`,
			rule: &RuleList{
				Rules: []Rule{
					Copy("inputs", "filebeat"),
				},
			},
		},
		"copy keep ordering for filtering": {
			givenYAML: `
inputs:
  - type: event/file
  - type: metric/docker
`,
			expectedYAML: `
filebeat:
  inputs:
    - type: event/file
    - type: metric/docker
`,
			rule: &RuleList{
				Rules: []Rule{
					Copy("inputs", "filebeat"),
					Filter("filebeat"),
				},
			},
		},
		"copy non existing key": {
			givenYAML: `
inputs:
  - type: event/file
  - type: metric/docker
`,
			expectedYAML: `
inputs:
  - type: event/file
  - type: metric/docker
`,
			rule: &RuleList{
				Rules: []Rule{
					Copy("what-inputs", "filebeat"),
				},
			},
		},
		"translate key values to another value": {
			givenYAML: `
name: "hello"
`,
			expectedYAML: `
name: "bonjour"
`,
			rule: &RuleList{
				Rules: []Rule{
					Translate("name", map[string]interface{}{
						"aurevoir": "a bientot",
						"hello":    "bonjour",
					}),
				},
			},
		},
		"translate on non existing key": {
			givenYAML: `
name: "hello"
`,
			expectedYAML: `
name: "hello"
`,
			rule: &RuleList{
				Rules: []Rule{
					Translate("donotexist", map[string]interface{}{
						"aurevoir": "a bientot",
						"hello":    "bonjour",
					}),
				},
			},
		},
		"translate 1 level deep key values to another value": {
			givenYAML: `
input:
  type: "aurevoir"
`,
			expectedYAML: `
input:
  type: "a bientot"
`,
			rule: &RuleList{
				Rules: []Rule{
					Translate("input.type", map[string]interface{}{
						"aurevoir": "a bientot",
						"hello":    "bonjour",
					}),
				},
			},
		},
		"map operation on array": {
			givenYAML: `
inputs:
  - type: event/file
  - type: log/docker
`,
			expectedYAML: `
inputs:
  - type: log
  - type: docker
`,
			rule: &RuleList{
				Rules: []Rule{
					Map("inputs",
						Translate("type", map[string]interface{}{
							"event/file": "log",
							"log/docker": "docker",
						})),
				},
			},
		},
		"map operation on non existing": {
			givenYAML: `
inputs:
  - type: event/file
  - type: log/docker
`,
			expectedYAML: `
inputs:
  - type: event/file
  - type: log/docker
`,
			rule: &RuleList{
				Rules: []Rule{
					Map("no-inputs",
						Translate("type", map[string]interface{}{
							"event/file": "log",
							"log/docker": "docker",
						})),
				},
			},
		},
		"single selector on top level keys": {
			givenYAML: `
inputs:
  - type: event/file
output:
  logstash:
    port: 5
`,
			expectedYAML: `
output:
  logstash:
    port: 5
`,
			rule: &RuleList{
				Rules: []Rule{
					Filter("output"),
				},
			},
		},
		"multiple selectors on top level keys": {
			givenYAML: `
inputs:
  - type: event/file
filebeat:
  - type: docker
output:
  logstash:
    port: 5
`,
			expectedYAML: `
inputs:
  - type: event/file
output:
  logstash:
    port: 5
`,
			rule: &RuleList{
				Rules: []Rule{
					Filter("output", "inputs"),
				},
			},
		},
		"filter for non existing keys": {
			givenYAML: `
inputs:
  - type: event/file
filebeat:
  - type: docker
output:
  logstash:
    port: 5
`,
			expectedYAML: ``,
			rule: &RuleList{
				Rules: []Rule{
					Filter("no-output", "no-inputs"),
				},
			},
		},

		"filter for values": {
			givenYAML: `
inputs:
  - type: log
  - type: tcp
  - type: udp
`,
			expectedYAML: `
inputs:
  - type: log
  - type: tcp
`,
			rule: &RuleList{
				Rules: []Rule{
					FilterValues("inputs", "type", "log", "tcp"),
				},
			},
		},
		"filter for regexp": {
			givenYAML: `
inputs:
  - type: metric/log
  - type: metric/tcp
  - type: udp
  - type: unknown
`,
			expectedYAML: `
inputs:
  - type: metric/log
  - type: metric/tcp
`,
			rule: &RuleList{
				Rules: []Rule{
					FilterValuesWithRegexp("inputs", "type", regexp.MustCompile("^metric/.*")),
				},
			},
		},
		"translate with regexp": {
			givenYAML: `
inputs:
  - type: metric/log
  - type: metric/tcp
`,
			expectedYAML: `
inputs:
  - type: log
  - type: tcp
`,
			rule: &RuleList{
				Rules: []Rule{
					Map("inputs", TranslateWithRegexp("type", regexp.MustCompile("^metric/(.*)"), "$1")),
				},
			},
		},

		"remove key": {
			givenYAML: `
key1: val1
key2: val2
`,
			expectedYAML: `
key1: val1
`,
			rule: &RuleList{
				Rules: []Rule{
					RemoveKey("key2"),
				},
			},
		},

		"copy item to list": {
			givenYAML: `
namespace: testing
inputs:
  - type: metric/log
  - type: metric/tcp
`,
			expectedYAML: `
namespace: testing
inputs:
  - type: metric/log
    namespace: testing
  - type: metric/tcp
    namespace: testing
`,
			rule: &RuleList{
				Rules: []Rule{
					CopyToList("namespace", "inputs"),
				},
			},
		},

		"Make array": {
			givenYAML: `
sample:
  log: "log value"
`,
			expectedYAML: `
sample:
  log: "log value"
logs:
  - "log value"
`,
			rule: &RuleList{
				Rules: []Rule{
					MakeArray("sample.log", "logs"),
				},
			},
		},
	}

	for name, test := range testcases {
		t.Run(name, func(t *testing.T) {
			a, err := makeASTFromYAML(test.givenYAML)
			require.NoError(t, err)

			err = test.rule.Apply(a)
			require.NoError(t, err)

			v := &MapVisitor{}
			a.Accept(v)

			var m map[string]interface{}
			if len(test.expectedYAML) == 0 {
				m = make(map[string]interface{})
			} else {
				err := yamltest.FromYAML([]byte(test.expectedYAML), &m)
				require.NoError(t, err)
			}

			if !assert.True(t, cmp.Equal(v.Content, m)) {
				diff := cmp.Diff(v.Content, m)
				if diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func makeASTFromYAML(yamlStr string) (*AST, error) {
	var m map[string]interface{}
	if err := yaml.Unmarshal([]byte(yamlStr), &m); err != nil {
		return nil, err
	}

	return NewAST(m)
}

func TestSerialization(t *testing.T) {
	value := NewRuleList(
		Rename("from-value", "to-value"),
		Copy("from-value", "to-value"),
		Translate("path-value", map[string]interface{}{
			"key-v-1": "value-v-1",
			"key-v-2": "value-v-2",
		}),
		TranslateWithRegexp("path-value", regexp.MustCompile("^metric/(.+)"), "log/$1"),
		Map("path-value",
			Rename("from-value", "to-value"),
			Copy("from-value", "to-value"),
		),
		Filter("f1", "f2"),
		FilterValues("select-v", "key-v", "v1", "v2"),
		FilterValuesWithRegexp("inputs", "type", regexp.MustCompile("^metric/.*")),
		ExtractListItem("path.p", "item", "target"),
		InjectIndex("index-type"),
		CopyToList("t1", "t2"),
	)

	y := `- rename:
    from: from-value
    to: to-value
- copy:
    from: from-value
    to: to-value
- translate:
    path: path-value
    mapper:
      key-v-1: value-v-1
      key-v-2: value-v-2
- translate_with_regexp:
    path: path-value
    re: ^metric/(.+)
    with: log/$1
- map:
    path: path-value
    rules:
    - rename:
        from: from-value
        to: to-value
    - copy:
        from: from-value
        to: to-value
- filter:
    selectors:
    - f1
    - f2
- filter_values:
    selector: select-v
    key: key-v
    values:
    - v1
    - v2
- filter_values_with_regexp:
    key: type
    re: ^metric/.*
    selector: inputs
- extract_list_items:
    path: path.p
    item: item
    to: target
- inject_index:
    type: index-type
- copy_to_list:
    item: t1
    to: t2
`

	t.Run("serialize_rules", func(t *testing.T) {
		b, err := yaml.Marshal(value)
		require.NoError(t, err)
		assert.Equal(t, string(b), y)
	})

	t.Run("unserialize_rules", func(t *testing.T) {
		v := &RuleList{}
		err := yaml.Unmarshal([]byte(y), v)
		require.NoError(t, err)
		assert.Equal(t, value, v)
	})
}
