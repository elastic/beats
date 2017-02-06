// +build !integration

package fileset

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/paths"
	"github.com/stretchr/testify/assert"
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

	configs := []ModuleConfig{
		{Module: "nginx"},
		{Module: "mysql"},
		{Module: "system"},
	}

	reg, err := newModuleRegistry(modulesPath, configs, nil, "5.2.0")
	assert.NoError(t, err)
	assert.NotNil(t, reg)

	expectedModules := map[string][]string{
		"nginx":  {"access", "error"},
		"mysql":  {"slowlog", "error"},
		"system": {"syslog"},
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
			_, err = fileset.getProspectorConfig()
			assert.NoError(t, err, fmt.Sprintf("module: %s, fileset: %s", module, name))
		}
	}
}

func TestNewModuleRegistryConfig(t *testing.T) {
	modulesPath, err := filepath.Abs("../module")
	assert.NoError(t, err)

	falseVar := false

	configs := []ModuleConfig{
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

	reg, err := newModuleRegistry(modulesPath, configs, nil, "5.2.0")
	assert.NoError(t, err)
	assert.NotNil(t, reg)

	nginxAccess := reg.registry["nginx"]["access"]
	assert.NotNil(t, nginxAccess)
	assert.Equal(t, []interface{}{"/hello/test"}, nginxAccess.vars["paths"])

	assert.NotContains(t, reg.registry["nginx"], "error")
}

func TestAppplyOverrides(t *testing.T) {

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
				Prospector: map[string]interface{}{
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
		configs  []ModuleConfig
		modules  []string
		expected []ModuleConfig
	}{
		{
			name:    "just modules",
			configs: []ModuleConfig{},
			modules: []string{"moduleA", "moduleB", "moduleC"},
			expected: []ModuleConfig{
				{Module: "moduleA"},
				{Module: "moduleB"},
				{Module: "moduleC"},
			},
		},
		{
			name: "eliminate a duplicate, no override",
			configs: []ModuleConfig{
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
			expected: []ModuleConfig{
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
			configs: []ModuleConfig{
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
			expected: []ModuleConfig{
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

	reg, err := NewModuleRegistry(configs, "5.2.0")
	assert.NoError(t, err)
	assert.NotNil(t, reg)

	// this should return an empty list, but no error
	prospectors, err := reg.GetProspectorConfigs()
	assert.NoError(t, err)
	assert.Equal(t, 0, len(prospectors))
}
