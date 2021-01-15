// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package program

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/internal/yamltest"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/transpiler"
)

func TestGroupBy(t *testing.T) {
	t.Run("only named output", func(t *testing.T) {
		sConfig := map[string]interface{}{
			"outputs": map[string]interface{}{
				"special": map[string]interface{}{
					"type":     "elasticsearch",
					"hosts":    "xxx",
					"username": "myusername",
					"password": "mypassword",
				},
				"infosec1": map[string]interface{}{
					"type":     "elasticsearch",
					"hosts":    "yyy",
					"username": "anotherusername",
					"password": "anotherpassword",
				},
			},

			"inputs": []map[string]interface{}{
				{
					"type":       "log",
					"use_output": "special",
					"streams":    map[string]interface{}{"paths": "/var/log/hello.log"},
				},
				{
					"type":       "system/metrics",
					"use_output": "special",
				},
				{
					"type":       "log",
					"streams":    map[string]interface{}{"paths": "/var/log/infosec.log"},
					"use_output": "infosec1",
				},
			},
		}

		ast, err := transpiler.NewAST(sConfig)
		require.NoError(t, err)

		grouped, err := groupByOutputs(ast)
		require.NoError(t, err)
		require.Equal(t, 2, len(grouped))

		c1 := transpiler.MustNewAST(map[string]interface{}{
			"output": map[string]interface{}{
				"elasticsearch": map[string]interface{}{
					"hosts":    "xxx",
					"username": "myusername",
					"password": "mypassword",
				},
			},
			"inputs": []map[string]interface{}{
				{
					"type":       "log",
					"streams":    map[string]interface{}{"paths": "/var/log/hello.log"},
					"use_output": "special",
				},
				{
					"type":       "system/metrics",
					"use_output": "special",
				},
			},
		})

		c2, _ := transpiler.NewAST(map[string]interface{}{
			"output": map[string]interface{}{
				"elasticsearch": map[string]interface{}{
					"hosts":    "yyy",
					"username": "anotherusername",
					"password": "anotherpassword",
				},
			},
			"inputs": []map[string]interface{}{
				{
					"type":       "log",
					"streams":    map[string]interface{}{"paths": "/var/log/infosec.log"},
					"use_output": "infosec1",
				},
			},
		})

		defaultConfig, ok := grouped["special"]
		require.True(t, ok)
		require.Equal(t, c1.Hash(), defaultConfig.Hash())

		infosec1Config, ok := grouped["infosec1"]

		require.True(t, ok)
		require.Equal(t, c2.Hash(), infosec1Config.Hash())
	})

	t.Run("fail when the referenced named output doesn't exist", func(t *testing.T) {
		sConfig := map[string]interface{}{
			"monitoring": map[string]interface{}{
				"elasticsearch": map[string]interface{}{
					"hosts": "localhost",
				},
			},
			"outputs": map[string]interface{}{
				"default": map[string]interface{}{
					"type":     "elasticsearch",
					"hosts":    "xxx",
					"username": "myusername",
					"password": "mypassword",
				},
				"infosec1": map[string]interface{}{
					"type":     "elasticsearch",
					"hosts":    "yyy",
					"username": "anotherusername",
					"password": "anotherpassword",
				},
			},

			"inputs": []map[string]interface{}{
				{
					"type":       "log",
					"streams":    map[string]interface{}{"paths": "/var/log/hello.log"},
					"use_output": "special",
				},
				{
					"type":       "system/metrics",
					"use_output": "special",
				},
				{
					"type":       "log",
					"streams":    map[string]interface{}{"paths": "/var/log/infosec.log"},
					"use_output": "donotexist",
				},
			},
		}

		ast, err := transpiler.NewAST(sConfig)
		require.NoError(t, err)

		_, err = groupByOutputs(ast)
		require.Error(t, err)
	})

	t.Run("only default output", func(t *testing.T) {
		sConfig := map[string]interface{}{
			"outputs": map[string]interface{}{
				"default": map[string]interface{}{
					"type":     "elasticsearch",
					"hosts":    "xxx",
					"username": "myusername",
					"password": "mypassword",
				},
				"infosec1": map[string]interface{}{
					"type":     "elasticsearch",
					"hosts":    "yyy",
					"username": "anotherusername",
					"password": "anotherpassword",
				},
			},
			"inputs": []map[string]interface{}{
				{
					"type":    "log",
					"streams": map[string]interface{}{"paths": "/var/log/hello.log"},
				},

				{
					"type": "system/metrics",
				},
				{
					"type":    "log",
					"streams": map[string]interface{}{"paths": "/var/log/infosec.log"},
				},
			},
		}

		ast, err := transpiler.NewAST(sConfig)
		require.NoError(t, err)

		grouped, err := groupByOutputs(ast)
		require.NoError(t, err)
		require.Equal(t, 1, len(grouped))

		c1 := transpiler.MustNewAST(map[string]interface{}{
			"output": map[string]interface{}{
				"elasticsearch": map[string]interface{}{
					"hosts":    "xxx",
					"username": "myusername",
					"password": "mypassword",
				},
			},
			"inputs": []map[string]interface{}{
				{
					"type":    "log",
					"streams": map[string]interface{}{"paths": "/var/log/hello.log"},
				},

				{
					"type": "system/metrics",
				},

				{
					"type":    "log",
					"streams": map[string]interface{}{"paths": "/var/log/infosec.log"},
				},
			},
		})

		defaultConfig, ok := grouped["default"]
		require.True(t, ok)
		require.Equal(t, c1.Hash(), defaultConfig.Hash())

		_, ok = grouped["infosec1"]

		require.False(t, ok)
	})

	t.Run("default and named output", func(t *testing.T) {
		sConfig := map[string]interface{}{
			"outputs": map[string]interface{}{
				"default": map[string]interface{}{
					"type":     "elasticsearch",
					"hosts":    "xxx",
					"username": "myusername",
					"password": "mypassword",
				},
				"infosec1": map[string]interface{}{
					"type":     "elasticsearch",
					"hosts":    "yyy",
					"username": "anotherusername",
					"password": "anotherpassword",
				},
			},
			"inputs": []map[string]interface{}{
				{
					"type":    "log",
					"streams": map[string]interface{}{"paths": "/var/log/hello.log"},
				},

				{
					"type": "system/metrics",
				},

				{
					"type":       "log",
					"streams":    map[string]interface{}{"paths": "/var/log/infosec.log"},
					"use_output": "infosec1",
				},
			},
		}

		ast, err := transpiler.NewAST(sConfig)
		require.NoError(t, err)

		grouped, err := groupByOutputs(ast)
		require.NoError(t, err)
		require.Equal(t, 2, len(grouped))

		c1 := transpiler.MustNewAST(map[string]interface{}{
			"output": map[string]interface{}{
				"elasticsearch": map[string]interface{}{
					"hosts":    "xxx",
					"username": "myusername",
					"password": "mypassword",
				},
			},
			"inputs": []map[string]interface{}{
				{
					"type":    "log",
					"streams": map[string]interface{}{"paths": "/var/log/hello.log"},
				},

				{
					"type": "system/metrics",
				},
			},
		})

		c2, _ := transpiler.NewAST(map[string]interface{}{
			"output": map[string]interface{}{
				"elasticsearch": map[string]interface{}{
					"hosts":    "yyy",
					"username": "anotherusername",
					"password": "anotherpassword",
				},
			},
			"inputs": []map[string]interface{}{
				{
					"type":       "log",
					"streams":    map[string]interface{}{"paths": "/var/log/infosec.log"},
					"use_output": "infosec1",
				},
			},
		})

		defaultConfig, ok := grouped["default"]
		require.True(t, ok)
		require.Equal(t, c1.Hash(), defaultConfig.Hash())

		infosec1Config, ok := grouped["infosec1"]

		require.True(t, ok)
		require.Equal(t, c2.Hash(), infosec1Config.Hash())
	})

	t.Run("streams is an empty list", func(t *testing.T) {
		sConfig := map[string]interface{}{
			"outputs": map[string]interface{}{
				"default": map[string]interface{}{
					"type":     "elasticsearch",
					"hosts":    "xxx",
					"username": "myusername",
					"password": "mypassword",
				},
				"infosec1": map[string]interface{}{
					"type":     "elasticsearch",
					"hosts":    "yyy",
					"username": "anotherusername",
					"password": "anotherpassword",
				},
			},
			"datasources": []map[string]interface{}{},
		}

		ast, err := transpiler.NewAST(sConfig)
		require.NoError(t, err)

		grouped, err := groupByOutputs(ast)
		require.NoError(t, err)
		require.Equal(t, 0, len(grouped))
	})

	t.Run("no streams are defined", func(t *testing.T) {
		sConfig := map[string]interface{}{
			"outputs": map[string]interface{}{
				"default": map[string]interface{}{
					"type":     "elasticsearch",
					"hosts":    "xxx",
					"username": "myusername",
					"password": "mypassword",
				},
				"infosec1": map[string]interface{}{
					"type":     "elasticsearch",
					"hosts":    "yyy",
					"username": "anotherusername",
					"password": "anotherpassword",
				},
			},
		}

		ast, err := transpiler.NewAST(sConfig)
		require.NoError(t, err)

		grouped, err := groupByOutputs(ast)
		require.NoError(t, err)
		require.Equal(t, 0, len(grouped))
	})
}

func TestConfiguration(t *testing.T) {
	defer os.Remove("fleet.yml")

	testcases := map[string]struct {
		programs []string
		expected int
		empty    bool
		err      bool
	}{
		"single_config": {
			programs: []string{"filebeat", "heartbeat", "metricbeat", "endpoint", "packetbeat"},
			expected: 5,
		},
		// "audit_config": {
		// 	programs: []string{"auditbeat"},
		// 	expected: 1,
		// },
		// "journal_config": {
		// 	programs: []string{"journalbeat"},
		// 	expected: 1,
		// },
		"synthetics_config": {
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
			empty: true,
		},
		"endpoint_basic": {
			programs: []string{"endpoint"},
			expected: 1,
		},
		"endpoint_no_fleet": {
			expected: 0,
		},
		"endpoint_unknown_output": {
			expected: 0,
		},
		"endpoint_arm": {
			expected: 0,
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

			programs, err := Programs(&fakeAgentInfo{}, ast)
			if test.err {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if test.empty {
				require.Equal(t, 0, len(programs))
				return
			}

			require.Equal(t, 1, len(programs))

			defPrograms, ok := programs["default"]
			require.True(t, ok)
			require.Equal(t, test.expected, len(defPrograms))

			for _, program := range defPrograms {
				programConfig, err := ioutil.ReadFile(filepath.Join(
					"testdata",
					name+"-"+strings.ToLower(program.Spec.Cmd)+".yml",
				))

				require.NoError(t, err)
				var m map[string]interface{}
				err = yamltest.FromYAML(programConfig, &m)
				require.NoError(t, errors.Wrap(err, program.Cmd()))

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
