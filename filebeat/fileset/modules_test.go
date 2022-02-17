// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

//go:build !integration
// +build !integration

package fileset

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/paths"
)

func load(t *testing.T, from interface{}) *common.Config {
	config, err := common.NewConfigFrom(from)
	if err != nil {
		t.Fatalf("Config err: %v", err)
	}
	return config
}

func TestNewModuleRegistry(t *testing.T) {
	modulesPath, err := filepath.Abs("../module")
	require.NoError(t, err)

	configs := []*ModuleConfig{
		{Module: "nginx"},
		{Module: "mysql"},
		{Module: "system"},
		{Module: "auditd"},
	}

	reg, err := newModuleRegistry(modulesPath, configs, nil, nil, beat.Info{Version: "5.2.0"})
	require.NoError(t, err)
	assert.NotNil(t, reg)

	expectedModules := map[string][]string{
		"auditd": {"log"},
		"nginx":  {"access", "error", "ingress_controller"},
		"mysql":  {"slowlog", "error"},
		"system": {"syslog", "auth"},
	}

	assert.Equal(t, len(expectedModules), len(reg.registry))
	for name, filesets := range reg.registry {
		expectedFilesets, exists := expectedModules[name]
		assert.True(t, exists)

		assert.Equal(t, len(expectedFilesets), len(filesets))
		for _, fileset := range expectedFilesets {
			fs := filesets[fileset]
			assert.NotNil(t, fs)
		}
<<<<<<< HEAD
=======
		sort.Strings(filesetList)
		sort.Strings(expectedFilesets)
		assert.Equal(t, filesetList, expectedFilesets)
>>>>>>> 8ab92d8e7d (Fix TestNewModuleRegistry flakiness (#30453))
	}

	for module, filesets := range reg.registry {
		for name, fileset := range filesets {
			cfg, err := fileset.getInputConfig()
			require.NoError(t, err, fmt.Sprintf("module: %s, fileset: %s", module, name))

			moduleName, err := cfg.String("_module_name", -1)
			require.NoError(t, err)
			assert.Equal(t, module, moduleName)

			filesetName, err := cfg.String("_fileset_name", -1)
			require.NoError(t, err)
			assert.Equal(t, name, filesetName)
		}
	}
}

func TestNewModuleRegistryConfig(t *testing.T) {
	modulesPath, err := filepath.Abs("../module")
	require.NoError(t, err)

	falseVar := false

	configs := []*ModuleConfig{
		{
			Module: "nginx",
			Filesets: map[string]*FilesetConfig{
				"access": {
					Var: map[string]interface{}{
						"paths": []interface{}{"/hello/test"},
					},
				},
				"error": {
					Enabled: &falseVar,
				},
			},
		},
		{
			Module:  "mysql",
			Enabled: &falseVar,
		},
	}

	reg, err := newModuleRegistry(modulesPath, configs, nil, nil, beat.Info{Version: "5.2.0"})
	require.NoError(t, err)
	assert.NotNil(t, reg)

	nginxAccess := reg.registry["nginx"]["access"]
	if assert.NotNil(t, nginxAccess) {
		assert.Equal(t, []interface{}{"/hello/test"}, nginxAccess.vars["paths"])
	}

	assert.NotContains(t, reg.registry["nginx"], "error")
}

func TestMovedModule(t *testing.T) {
	modulesPath, err := filepath.Abs("./test/moved_module")
	require.NoError(t, err)

	configs := []*ModuleConfig{
		{
			Module: "old",
			Filesets: map[string]*FilesetConfig{
				"test": {},
			},
		},
	}

	reg, err := newModuleRegistry(modulesPath, configs, nil, nil, beat.Info{Version: "5.2.0"})
	require.NoError(t, err)
	assert.NotNil(t, reg)
}

func TestApplyOverrides(t *testing.T) {
	falseVar := false
	trueVar := true

	tests := []struct {
		name            string
		fcfg            FilesetConfig
		module, fileset string
		overrides       *ModuleOverrides
		expected        FilesetConfig
		hasOverride     bool
	}{
		{
			name: "no overrides",
			fcfg: FilesetConfig{
				Var: map[string]interface{}{
					"a":   "test",
					"b.c": "test",
				},
				Input: map[string]interface{}{},
			},
			module:  "nginx",
			fileset: "access",
			expected: FilesetConfig{
				Var: map[string]interface{}{
					"a":   "test",
					"b.c": "test",
				},
				Input: map[string]interface{}{},
			},
			hasOverride: false,
		},
		{
			name: "var overrides",
			fcfg: FilesetConfig{
				Var: map[string]interface{}{
					"a":   "test",
					"b.c": "test",
				},
				Input: map[string]interface{}{},
			},
			module:  "nginx",
			fileset: "access",
			overrides: &ModuleOverrides{
				"nginx": map[string]*common.Config{
					"access": load(t, map[string]interface{}{
						"var.a":   "test1",
						"var.b.c": "test2"}),
				},
			},
			expected: FilesetConfig{
				Var: map[string]interface{}{
					"a": "test1",
					"b": map[string]interface{}{"c": "test2"},
				},
				Input: map[string]interface{}{},
			},
			hasOverride: true,
		},
		{
			name: "enable and var overrides",
			fcfg: FilesetConfig{
				Enabled: &falseVar,
				Var: map[string]interface{}{
					"paths": []string{"/var/log/nginx"},
				},
				Input: map[string]interface{}{},
			},
			module:  "nginx",
			fileset: "access",
			overrides: &ModuleOverrides{
				"nginx": map[string]*common.Config{
					"access": load(t, map[string]interface{}{
						"enabled":   true,
						"var.paths": []interface{}{"/var/local/nginx/log"}}),
				},
			},
			expected: FilesetConfig{
				Enabled: &trueVar,
				Var: map[string]interface{}{
					"paths": []interface{}{"/var/local/nginx/log"},
				},
				Input: map[string]interface{}{},
			},
			hasOverride: true,
		},
		{
			name:    "input overrides",
			fcfg:    FilesetConfig{},
			module:  "nginx",
			fileset: "access",
			overrides: &ModuleOverrides{
				"nginx": map[string]*common.Config{
					"access": load(t, map[string]interface{}{
						"input.close_eof": true,
					}),
				},
			},
			expected: FilesetConfig{
				Input: map[string]interface{}{
					"close_eof": true,
				},
				Var: map[string]interface{}{},
			},
			hasOverride: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, hasOverride, err := applyOverrides(&test.fcfg, test.module, test.fileset, test.overrides)
			require.NoError(t, err)
			assert.Equal(t, &test.expected, result)
			assert.Equal(t, test.hasOverride, hasOverride)
		})
	}
}

func TestAppendWithoutDuplicates(t *testing.T) {
	falseVar := false
	tests := []struct {
		name     string
		configs  []*ModuleConfig
		modules  []string
		expected []*ModuleConfig
	}{
		{
			name:    "just modules",
			configs: []*ModuleConfig{},
			modules: []string{"moduleA", "moduleB", "moduleC"},
			expected: []*ModuleConfig{
				{Module: "moduleA"},
				{Module: "moduleB"},
				{Module: "moduleC"},
			},
		},
		{
			name: "eliminate a duplicate, no override",
			configs: []*ModuleConfig{
				{
					Module: "moduleB",
					Filesets: map[string]*FilesetConfig{
						"fileset": {
							Var: map[string]interface{}{
								"paths": "test",
							},
						},
					},
				},
			},
			modules: []string{"moduleA", "moduleB", "moduleC"},
			expected: []*ModuleConfig{
				{
					Module: "moduleB",
					Filesets: map[string]*FilesetConfig{
						"fileset": {
							Var: map[string]interface{}{
								"paths": "test",
							},
						},
					},
				},
				{Module: "moduleA"},
				{Module: "moduleC"},
			},
		},
		{
			name: "disabled config",
			configs: []*ModuleConfig{
				{
					Module:  "moduleB",
					Enabled: &falseVar,
					Filesets: map[string]*FilesetConfig{
						"fileset": {
							Var: map[string]interface{}{
								"paths": "test",
							},
						},
					},
				},
			},
			modules: []string{"moduleA", "moduleB", "moduleC"},
			expected: []*ModuleConfig{
				{
					Module:  "moduleB",
					Enabled: &falseVar,
					Filesets: map[string]*FilesetConfig{
						"fileset": {
							Var: map[string]interface{}{
								"paths": "test",
							},
						},
					},
				},
				{Module: "moduleA"},
				{Module: "moduleB"},
				{Module: "moduleC"},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := appendWithoutDuplicates(test.configs, test.modules)
			require.NoError(t, err)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestMcfgFromConfig(t *testing.T) {
	falseVar := false
	tests := []struct {
		name                       string
		config                     *common.Config
		expectedModuleConfig       ModuleConfig
		expectedConfiguredFilesets map[string]struct{}
	}{
		{
			name: "not defined fileset",
			config: load(t, map[string]interface{}{
				"module": "nginx",
			}),
			expectedModuleConfig: ModuleConfig{
				Module: "nginx",
			},
			expectedConfiguredFilesets: map[string]struct{}{},
		},
		{
			name: "disable fileset",
			config: load(t, map[string]interface{}{
				"module":        "nginx",
				"error.enabled": false,
			}),
			expectedModuleConfig: ModuleConfig{
				Module: "nginx",
				Filesets: map[string]*FilesetConfig{
					"error": {
						Enabled: &falseVar,
						Var:     nil,
						Input:   nil,
					},
				},
			},
			expectedConfiguredFilesets: map[string]struct{}{
				"error": struct{}{},
			},
		},
		{
			name: "set variable",
			config: load(t, map[string]interface{}{
				"module":          "nginx",
				"access.var.test": false,
			}),
			expectedModuleConfig: ModuleConfig{
				Module: "nginx",
				Filesets: map[string]*FilesetConfig{
					"access": {
						Var: map[string]interface{}{
							"test": false,
						},
						Input: nil,
					},
				},
			},
			expectedConfiguredFilesets: map[string]struct{}{
				"access": struct{}{},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, configuredFilets, err := mcfgFromConfig(test.config)
			require.NoError(t, err)
			assert.Equal(t, test.expectedConfiguredFilesets, configuredFilets)
			assert.Equal(t, test.expectedModuleConfig.Module, result.Module)
			assert.Equal(t, len(test.expectedModuleConfig.Filesets), len(result.Filesets))
			for name, fileset := range test.expectedModuleConfig.Filesets {
				assert.Equal(t, fileset, result.Filesets[name])
			}
		})
	}
}

func TestMissingModuleFolder(t *testing.T) {
	home := paths.Paths.Home
	paths.Paths.Home = "/no/such/path"
	defer func() { paths.Paths.Home = home }()

	configs := []*common.Config{
		load(t, map[string]interface{}{"module": "nginx"}),
	}

	reg, err := NewModuleRegistry(configs, beat.Info{Version: "5.2.0"}, true)
	require.NoError(t, err)
	assert.NotNil(t, reg)

	// this should return an empty list, but no error
	inputs, err := reg.GetInputConfigs()
	require.NoError(t, err)
	assert.Equal(t, 0, len(inputs))
}

func TestInterpretError(t *testing.T) {
	tests := []struct {
		Test   string
		Input  string
		Output string
	}{
		{
			Test:  "other plugin not installed",
			Input: `{"error":{"root_cause":[{"type":"parse_exception","reason":"No processor type exists with name [hello_test]","header":{"processor_type":"hello_test"}}],"type":"parse_exception","reason":"No processor type exists with name [hello_test]","header":{"processor_type":"hello_test"}},"status":400}`,
			Output: "this module requires an Elasticsearch plugin that provides the hello_test processor. " +
				"Please visit the Elasticsearch documentation for instructions on how to install this plugin. " +
				"Response body: " + `{"error":{"root_cause":[{"type":"parse_exception","reason":"No processor type exists with name [hello_test]","header":{"processor_type":"hello_test"}}],"type":"parse_exception","reason":"No processor type exists with name [hello_test]","header":{"processor_type":"hello_test"}},"status":400}`,
		},
		{
			Test:   "Elasticsearch 2.4",
			Input:  `{"error":{"root_cause":[{"type":"invalid_index_name_exception","reason":"Invalid index name [_ingest], must not start with '_'","index":"_ingest"}],"type":"invalid_index_name_exception","reason":"Invalid index name [_ingest], must not start with '_'","index":"_ingest"},"status":400}`,
			Output: `the Ingest Node functionality seems to be missing from Elasticsearch. The Filebeat modules require Elasticsearch >= 5.0. This is the response I got from Elasticsearch: {"error":{"root_cause":[{"type":"invalid_index_name_exception","reason":"Invalid index name [_ingest], must not start with '_'","index":"_ingest"}],"type":"invalid_index_name_exception","reason":"Invalid index name [_ingest], must not start with '_'","index":"_ingest"},"status":400}`,
		},
		{
			Test:   "Elasticsearch 1.7",
			Input:  `{"error":"InvalidIndexNameException[[_ingest] Invalid index name [_ingest], must not start with '_']","status":400}`,
			Output: `the Filebeat modules require Elasticsearch >= 5.0. This is the response I got from Elasticsearch: {"error":"InvalidIndexNameException[[_ingest] Invalid index name [_ingest], must not start with '_']","status":400}`,
		},
		{
			Test:   "bad json",
			Input:  `blah`,
			Output: `couldn't load pipeline: test. Additionally, error decoding response body: blah`,
		},
		{
			Test:  "another error",
			Input: `{"error":{"root_cause":[{"type":"test","reason":""}],"type":"test","reason":""},"status":400}`,
			Output: "couldn't load pipeline: test. Response body: " +
				`{"error":{"root_cause":[{"type":"test","reason":""}],"type":"test","reason":""},"status":400}`,
		},
	}

	for _, test := range tests {
		t.Run(test.Test, func(t *testing.T) {
			errResult := interpretError(errors.New("test"), []byte(test.Input))
			assert.Equal(t, errResult.Error(), test.Output, test.Test)
		})
	}
}
