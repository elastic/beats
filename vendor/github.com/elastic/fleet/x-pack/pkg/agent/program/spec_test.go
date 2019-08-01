// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package program

import (
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	"github.com/elastic/fleet/x-pack/pkg/agent/internal/yamltest"
	"github.com/elastic/fleet/x-pack/pkg/agent/transpiler"
)

func TestConfiguration(t *testing.T) {
	testcases := map[string]struct {
		programs []string
	}{
		"single_config": {
			programs: []string{"filebeat", "metricbeat"},
		},
		"audit_config": {
			programs: []string{"auditbeat"},
		},
		"journal_config": {
			programs: []string{"journalbeat"},
		},
		"monitor_config": {
			programs: []string{"heartbeat"},
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
			require.NoError(t, err)
			require.Equal(t, len(programs), len(test.programs))

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
