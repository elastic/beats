// +build !integration

package fileset

import (
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/paths"
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
	assert.NoError(t, err)

	configs := []*ModuleConfig{
		&ModuleConfig{Module: "nginx"},
		&ModuleConfig{Module: "mysql"},
		&ModuleConfig{Module: "system"},
		&ModuleConfig{Module: "auditd"},
	}

	reg, err := newModuleRegistry(modulesPath, configs, nil, "5.2.0")
	assert.NoError(t, err)
	assert.NotNil(t, reg)

	expectedModules := map[string][]string{
		"auditd": {"log"},
		"nginx":  {"access", "error"},
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
	}

	for module, filesets := range reg.registry {
		for name, fileset := range filesets {
			cfg, err := fileset.getInputConfig()
			assert.NoError(t, err, fmt.Sprintf("module: %s, fileset: %s", module, name))

			moduleName, err := cfg.String("_module_name", -1)
			assert.NoError(t, err)
			assert.Equal(t, module, moduleName)

			filesetName, err := cfg.String("_fileset_name", -1)
			assert.NoError(t, err)
			assert.Equal(t, name, filesetName)
		}
	}
}

func TestNewModuleRegistryConfig(t *testing.T) {
	modulesPath, err := filepath.Abs("../module")
	assert.NoError(t, err)

	falseVar := false

	configs := []*ModuleConfig{
		&ModuleConfig{
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
		&ModuleConfig{
			Module:  "mysql",
			Enabled: &falseVar,
		},
	}

	reg, err := newModuleRegistry(modulesPath, configs, nil, "5.2.0")
	assert.NoError(t, err)
	assert.NotNil(t, reg)

	nginxAccess := reg.registry["nginx"]["access"]
	assert.NotNil(t, nginxAccess)
	assert.Equal(t, []interface{}{"/hello/test"}, nginxAccess.vars["paths"])

	assert.NotContains(t, reg.registry["nginx"], "error")
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
	}{
		{
			name: "var overrides",
			fcfg: FilesetConfig{
				Var: map[string]interface{}{
					"a":   "test",
					"b.c": "test",
				},
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
			},
		},
		{
			name: "enable and var overrides",
			fcfg: FilesetConfig{
				Enabled: &falseVar,
				Var: map[string]interface{}{
					"paths": []string{"/var/log/nginx"},
				},
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
			},
		},
		{
			name:    "prospector overrides",
			fcfg:    FilesetConfig{},
			module:  "nginx",
			fileset: "access",
			overrides: &ModuleOverrides{
				"nginx": map[string]*common.Config{
					"access": load(t, map[string]interface{}{
						"prospector.close_eof": true,
					}),
				},
			},
			expected: FilesetConfig{
				Input: map[string]interface{}{
					"close_eof": true,
				},
				Prospector: map[string]interface{}{
					"close_eof": true,
				},
			},
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
			},
		},
	}

	for _, test := range tests {
		result, err := applyOverrides(&test.fcfg, test.module, test.fileset, test.overrides)
		assert.NoError(t, err)
		assert.Equal(t, &test.expected, result, test.name)
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
				&ModuleConfig{Module: "moduleA"},
				&ModuleConfig{Module: "moduleB"},
				&ModuleConfig{Module: "moduleC"},
			},
		},
		{
			name: "eliminate a duplicate, no override",
			configs: []*ModuleConfig{
				&ModuleConfig{
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
				&ModuleConfig{
					Module: "moduleB",
					Filesets: map[string]*FilesetConfig{
						"fileset": {
							Var: map[string]interface{}{
								"paths": "test",
							},
						},
					},
				},
				&ModuleConfig{Module: "moduleA"},
				&ModuleConfig{Module: "moduleC"},
			},
		},
		{
			name: "disabled config",
			configs: []*ModuleConfig{
				&ModuleConfig{
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
				&ModuleConfig{
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
				&ModuleConfig{Module: "moduleA"},
				&ModuleConfig{Module: "moduleB"},
				&ModuleConfig{Module: "moduleC"},
			},
		},
	}

	for _, test := range tests {
		result, err := appendWithoutDuplicates(test.configs, test.modules)
		assert.NoError(t, err, test.name)
		assert.Equal(t, test.expected, result, test.name)
	}
}

func TestMcfgFromConfig(t *testing.T) {
	falseVar := false
	tests := []struct {
		name     string
		config   *common.Config
		expected ModuleConfig
	}{
		{
			name: "disable fileset",
			config: load(t, map[string]interface{}{
				"module":        "nginx",
				"error.enabled": false,
			}),
			expected: ModuleConfig{
				Module: "nginx",
				Filesets: map[string]*FilesetConfig{
					"error": {
						Enabled: &falseVar,
					},
				},
			},
		},
		{
			name: "set variable",
			config: load(t, map[string]interface{}{
				"module":          "nginx",
				"access.var.test": false,
			}),
			expected: ModuleConfig{
				Module: "nginx",
				Filesets: map[string]*FilesetConfig{
					"access": {
						Var: map[string]interface{}{
							"test": false,
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		result, err := mcfgFromConfig(test.config)
		assert.NoError(t, err, test.name)
		assert.Equal(t, test.expected.Module, result.Module)
		assert.Equal(t, len(test.expected.Filesets), len(result.Filesets))
		for name, fileset := range test.expected.Filesets {
			assert.Equal(t, fileset, result.Filesets[name])
		}
	}
}

func TestMissingModuleFolder(t *testing.T) {
	home := paths.Paths.Home
	paths.Paths.Home = "/no/such/path"
	defer func() { paths.Paths.Home = home }()

	configs := []*common.Config{
		load(t, map[string]interface{}{"module": "nginx"}),
	}

	reg, err := NewModuleRegistry(configs, "5.2.0", true)
	assert.NoError(t, err)
	assert.NotNil(t, reg)

	// this should return an empty list, but no error
	inputs, err := reg.GetInputConfigs()
	assert.NoError(t, err)
	assert.Equal(t, 0, len(inputs))
}

func TestInterpretError(t *testing.T) {
	tests := []struct {
		Test   string
		Input  string
		Output string
	}{
		{
			Test:   "geoip not installed",
			Input:  `{"error":{"root_cause":[{"type":"parse_exception","reason":"No processor type exists with name [geoip]","header":{"processor_type":"geoip"}}],"type":"parse_exception","reason":"No processor type exists with name [geoip]","header":{"processor_type":"geoip"}},"status":400}`,
			Output: "This module requires the ingest-geoip plugin to be installed in Elasticsearch. You can install it using the following command in the Elasticsearch home directory:\n    sudo bin/elasticsearch-plugin install ingest-geoip",
		},
		{
			Test:   "user-agent not installed",
			Input:  `{"error":{"root_cause":[{"type":"parse_exception","reason":"No processor type exists with name [user_agent]","header":{"processor_type":"user_agent"}}],"type":"parse_exception","reason":"No processor type exists with name [user_agent]","header":{"processor_type":"user_agent"}},"status":400}`,
			Output: "This module requires the ingest-user-agent plugin to be installed in Elasticsearch. You can install it using the following command in the Elasticsearch home directory:\n    sudo bin/elasticsearch-plugin install ingest-user-agent",
		},
		{
			Test:  "other plugin not installed",
			Input: `{"error":{"root_cause":[{"type":"parse_exception","reason":"No processor type exists with name [hello_test]","header":{"processor_type":"hello_test"}}],"type":"parse_exception","reason":"No processor type exists with name [hello_test]","header":{"processor_type":"hello_test"}},"status":400}`,
			Output: "This module requires an Elasticsearch plugin that provides the hello_test processor. " +
				"Please visit the Elasticsearch documentation for instructions on how to install this plugin. " +
				"Response body: " + `{"error":{"root_cause":[{"type":"parse_exception","reason":"No processor type exists with name [hello_test]","header":{"processor_type":"hello_test"}}],"type":"parse_exception","reason":"No processor type exists with name [hello_test]","header":{"processor_type":"hello_test"}},"status":400}`,
		},
		{
			Test:   "Elasticsearch 2.4",
			Input:  `{"error":{"root_cause":[{"type":"invalid_index_name_exception","reason":"Invalid index name [_ingest], must not start with '_'","index":"_ingest"}],"type":"invalid_index_name_exception","reason":"Invalid index name [_ingest], must not start with '_'","index":"_ingest"},"status":400}`,
			Output: `The Ingest Node functionality seems to be missing from Elasticsearch. The Filebeat modules require Elasticsearch >= 5.0. This is the response I got from Elasticsearch: {"error":{"root_cause":[{"type":"invalid_index_name_exception","reason":"Invalid index name [_ingest], must not start with '_'","index":"_ingest"}],"type":"invalid_index_name_exception","reason":"Invalid index name [_ingest], must not start with '_'","index":"_ingest"},"status":400}`,
		},
		{
			Test:   "Elasticsearch 1.7",
			Input:  `{"error":"InvalidIndexNameException[[_ingest] Invalid index name [_ingest], must not start with '_']","status":400}`,
			Output: `The Filebeat modules require Elasticsearch >= 5.0. This is the response I got from Elasticsearch: {"error":"InvalidIndexNameException[[_ingest] Invalid index name [_ingest], must not start with '_']","status":400}`,
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
		errResult := interpretError(errors.New("test"), []byte(test.Input))
		assert.Equal(t, errResult.Error(), test.Output, test.Test)
	}
}
