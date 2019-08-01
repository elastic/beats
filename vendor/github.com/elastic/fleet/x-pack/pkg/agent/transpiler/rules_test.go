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

	"github.com/elastic/fleet/x-pack/pkg/agent/internal/yamltest"
)

func TestRules(t *testing.T) {
	testcases := map[string]struct {
		givenYAML    string
		expectedYAML string
		rule         Rule
	}{
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
					Translate("name", []TranslateKV{
						TranslateKV{K: "aurevoir", V: "a bientot"},
						TranslateKV{K: "hello", V: "bonjour"},
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
					Translate("donotexist", []TranslateKV{
						TranslateKV{K: "aurevoir", V: "a bientot"},
						TranslateKV{K: "hello", V: "bonjour"},
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
					Translate("input.type", []TranslateKV{
						TranslateKV{K: "aurevoir", V: "a bientot"},
						TranslateKV{K: "hello", V: "bonjour"},
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
						Translate("type", []TranslateKV{
							TranslateKV{K: "event/file", V: "log"},
							TranslateKV{K: "log/docker", V: "docker"},
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
						Translate("type", []TranslateKV{
							TranslateKV{K: "event/file", V: "log"},
							TranslateKV{K: "log/docker", V: "docker"},
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
