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

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/internal/yamltest"
)

func TestRules(t *testing.T) {
	testcases := map[string]struct {
		givenYAML    string
		expectedYAML string
		rule         Rule
	}{
		"fix streams": {
			givenYAML: `
inputs:
  - name: All default
    type: file
    streams:
      - paths: /var/log/mysql/error.log
  - name: Specified namespace
    type: file
    data_stream.namespace: nsns
    streams:
      - paths: /var/log/mysql/error.log
  - name: Specified dataset
    type: file
    streams:
      - paths: /var/log/mysql/error.log
        data_stream.dataset: dsds
  - name: All specified
    type: file
    data_stream.namespace: nsns
    streams:
      - paths: /var/log/mysql/error.log
        data_stream.dataset: dsds
  - name: All specified with empty strings
    type: file
    data_stream.namespace: ""
    streams:
      - paths: /var/log/mysql/error.log
        data_stream.dataset: ""
`,
			expectedYAML: `
inputs:
  - name: All default
    type: file
    data_stream.namespace: default
    streams:
      - paths: /var/log/mysql/error.log
        data_stream.dataset: generic
  - name: Specified namespace
    type: file
    data_stream.namespace: nsns
    streams:
      - paths: /var/log/mysql/error.log
        data_stream.dataset: generic
  - name: Specified dataset
    type: file
    data_stream.namespace: default
    streams:
      - paths: /var/log/mysql/error.log
        data_stream.dataset: dsds
  - name: All specified
    type: file
    data_stream.namespace: nsns
    streams:
      - paths: /var/log/mysql/error.log
        data_stream.dataset: dsds
  - name: All specified with empty strings
    type: file
    data_stream.namespace: default
    streams:
      - paths: /var/log/mysql/error.log
        data_stream.dataset: generic
`,
			rule: &RuleList{
				Rules: []Rule{
					FixStream(),
				},
			},
		},

		"inject index": {
			givenYAML: `
inputs:
  - name: All default
    type: file
    streams:
      - paths: /var/log/mysql/error.log
  - name: Specified namespace
    type: file
    data_stream.namespace: nsns
    streams:
      - paths: /var/log/mysql/error.log

  - name: Specified dataset
    type: file
    streams:
      - paths: /var/log/mysql/error.log
        data_stream.dataset: dsds
  - name: All specified
    type: file
    data_stream.namespace: nsns
    streams:
      - paths: /var/log/mysql/error.log
        data_stream.dataset: dsds
  - name: All specified with empty strings
    type: file
    data_stream.namespace: ""
    streams:
      - paths: /var/log/mysql/error.log
        data_stream.dataset: ""
`,
			expectedYAML: `
inputs:
  - name: All default
    type: file
    streams:
      - paths: /var/log/mysql/error.log
        index: mytype-generic-default
  - name: Specified namespace
    type: file
    data_stream.namespace: nsns
    streams:
      - paths: /var/log/mysql/error.log
        index: mytype-generic-nsns

  - name: Specified dataset
    type: file
    streams:
      - paths: /var/log/mysql/error.log
        data_stream.dataset: dsds
        index: mytype-dsds-default
  - name: All specified
    type: file
    data_stream.namespace: nsns
    streams:
      - paths: /var/log/mysql/error.log
        data_stream.dataset: dsds
        index: mytype-dsds-nsns
  - name: All specified with empty strings
    type: file
    data_stream.namespace: ""
    streams:
      - paths: /var/log/mysql/error.log
        data_stream.dataset: ""
        index: mytype-generic-default
`,
			rule: &RuleList{
				Rules: []Rule{
					InjectIndex("mytype"),
				},
			},
		},

		"inject agent info": {
			givenYAML: `
inputs:
  - name: No processors
    type: file
  - name: With processors
    type: file
    processors:
      - add_fields:
          target: other
          fields:
            data: more
`,
			expectedYAML: `
inputs:
  - name: No processors
    type: file
    processors:
      - add_fields:
          target: elastic_agent
          fields:
            id: agent-id
            snapshot: false
            version: 8.0.0
  - name: With processors
    type: file
    processors:
      - add_fields:
          target: other
          fields:
            data: more
      - add_fields:
          target: elastic_agent
          fields:
            id: agent-id
            snapshot: false
            version: 8.0.0
`,
			rule: &RuleList{
				Rules: []Rule{
					InjectAgentInfo(),
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
		"select into": {
			givenYAML: `
level_one:
  key1: val1
  key2:
    d_key1: val2
    d_key2: val3
rest: of
`,
			expectedYAML: `
level_one:
  key1: val1
  key2:
    d_key1: val2
    d_key2: val3
  level_two:
    key1: val1
    key2:
      d_key1: val2
      d_key2: val3
rest: of
`,
			rule: &RuleList{
				Rules: []Rule{
					SelectInto("level_one.level_two", "level_one.key1", "level_one.key2"),
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
					CopyToList("namespace", "inputs", "insert_after"),
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
		"insert defaults into existing": {
			givenYAML: `
level_one:
  key1: val1
  key2:
    d_key1: val2
    d_key2: val3
  level_two:
    key2:
      d_key3: val3
      d_key4: val4
rest: of
`,
			expectedYAML: `
level_one:
  key1: val1
  key2:
    d_key1: val2
    d_key2: val3
  level_two:
    key1: val1
    key2:
      d_key3: val3
      d_key4: val4
rest: of
`,
			rule: &RuleList{
				Rules: []Rule{
					InsertDefaults("level_one.level_two", "level_one.key1", "level_one.key2"),
				},
			},
		},

		"insert defaults into not existing": {
			givenYAML: `
level_one:
  key1: val1
  key2:
    d_key1: val2
    d_key2: val3
rest: of
`,
			expectedYAML: `
level_one:
  key1: val1
  key2:
    d_key1: val2
    d_key2: val3
  level_two:
    key1: val1
    key2:
      d_key1: val2
      d_key2: val3
rest: of
`,
			rule: &RuleList{
				Rules: []Rule{
					InsertDefaults("level_one.level_two", "level_one.key1", "level_one.key2"),
				},
			},
		},

		"inject auth headers: no headers": {
			givenYAML: `
outputs:
  elasticsearch:
    hosts:
      - "127.0.0.1:9201"
      - "127.0.0.1:9202"
  logstash:
    port: 5
`,
			expectedYAML: `
outputs:
  elasticsearch:
    headers:
      h1: test-header
    hosts:
      - "127.0.0.1:9201"
      - "127.0.0.1:9202"
  logstash:
    port: 5
`,
			rule: &RuleList{
				Rules: []Rule{
					InjectHeaders(),
				},
			},
		},

		"inject auth headers: existing headers": {
			givenYAML: `
outputs:
  elasticsearch:
    headers:
      sample-header: existing
    hosts:
      - "127.0.0.1:9201"
      - "127.0.0.1:9202"
  logstash:
    port: 5
`,
			expectedYAML: `
outputs:
  elasticsearch:
    headers:
      sample-header: existing
      h1: test-header
    hosts:
      - "127.0.0.1:9201"
      - "127.0.0.1:9202"
  logstash:
    port: 5
`,
			rule: &RuleList{
				Rules: []Rule{
					InjectHeaders(),
				},
			},
		},
	}

	for name, test := range testcases {
		t.Run(name, func(t *testing.T) {
			a, err := makeASTFromYAML(test.givenYAML)
			require.NoError(t, err)

			err = test.rule.Apply(FakeAgentInfo(), a)
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

			if !assert.True(t, cmp.Equal(m, v.Content)) {
				diff := cmp.Diff(m, v.Content)
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
		InjectStreamProcessor("insert_after", "index-type"),
		CopyToList("t1", "t2", "insert_after"),
		CopyAllToList("t2", "insert_before", "a", "b"),
		FixStream(),
		SelectInto("target", "s1", "s2"),
		InsertDefaults("target", "s1", "s2"),
		InjectHeaders(),
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
- inject_stream_processor:
    type: index-type
    on_conflict: insert_after
- copy_to_list:
    item: t1
    to: t2
    on_conflict: insert_after
- copy_all_to_list:
    to: t2
    except:
    - a
    - b
    on_conflict: insert_before
- fix_stream: {}
- select_into:
    selectors:
    - s1
    - s2
    path: target
- insert_defaults:
    selectors:
    - s1
    - s2
    path: target
- inject_headers: {}
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

type fakeAgentInfo struct{}

func (*fakeAgentInfo) AgentID() string {
	return "agent-id"
}

func (*fakeAgentInfo) Version() string {
	return "8.0.0"
}

func (*fakeAgentInfo) Snapshot() bool {
	return false
}

func (*fakeAgentInfo) Headers() map[string]string {
	return map[string]string{
		"h1": "test-header",
	}
}

func FakeAgentInfo() AgentInfo {
	return &fakeAgentInfo{}
}
