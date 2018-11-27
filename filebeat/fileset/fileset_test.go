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

// +build !integration

package fileset

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

func getModuleForTesting(t *testing.T, module, fileset string) *Fileset {
	modulesPath, err := filepath.Abs("../module")
	assert.NoError(t, err)
	fs, err := New(modulesPath, fileset, &ModuleConfig{Module: module}, &FilesetConfig{})
	assert.NoError(t, err)

	return fs
}

func TestLoadManifestNginx(t *testing.T) {
	fs := getModuleForTesting(t, "nginx", "access")

	manifest, err := fs.readManifest()
	assert.NoError(t, err)
	assert.Equal(t, manifest.ModuleVersion, "1.0")
	assert.Equal(t, manifest.IngestPipeline, "ingest/default.json")
	assert.Equal(t, manifest.Input, "config/nginx-access.yml")

	vars := manifest.Vars
	assert.Equal(t, "paths", vars[0]["name"])
	path := (vars[0]["default"]).([]interface{})[0].(string)
	assert.Equal(t, path, "/var/log/nginx/access.log*")
}

func TestGetBuiltinVars(t *testing.T) {
	fs := getModuleForTesting(t, "nginx", "access")

	vars, err := fs.getBuiltinVars()
	assert.NoError(t, err)

	assert.IsType(t, vars["hostname"], "a-mac-with-esc-key")
	assert.IsType(t, vars["domain"], "local")
}

func TestEvaluateVarsNginx(t *testing.T) {
	fs := getModuleForTesting(t, "nginx", "access")

	var err error
	fs.manifest, err = fs.readManifest()
	assert.NoError(t, err)

	vars, err := fs.evaluateVars()
	assert.NoError(t, err)

	builtin := vars["builtin"].(map[string]interface{})
	assert.IsType(t, "a-mac-with-esc-key", builtin["hostname"])
	assert.IsType(t, "local", builtin["domain"])

	assert.IsType(t, []interface{}{"/usr/local/var/log/nginx/access.log*"}, vars["paths"])
}

func TestEvaluateVarsNginxOverride(t *testing.T) {
	modulesPath, err := filepath.Abs("../module")
	assert.NoError(t, err)
	fs, err := New(modulesPath, "access", &ModuleConfig{Module: "nginx"}, &FilesetConfig{
		Var: map[string]interface{}{
			"pipeline": "no_plugins",
		},
	})
	assert.NoError(t, err)

	fs.manifest, err = fs.readManifest()
	assert.NoError(t, err)

	vars, err := fs.evaluateVars()
	assert.NoError(t, err)

	assert.Equal(t, "no_plugins", vars["pipeline"])
}

func TestEvaluateVarsMySQL(t *testing.T) {
	fs := getModuleForTesting(t, "mysql", "slowlog")

	var err error
	fs.manifest, err = fs.readManifest()
	assert.NoError(t, err)

	vars, err := fs.evaluateVars()
	assert.NoError(t, err)

	builtin := vars["builtin"].(map[string]interface{})
	assert.IsType(t, "a-mac-with-esc-key", builtin["hostname"])
	assert.IsType(t, "local", builtin["domain"])

	expectedPaths := []interface{}{
		"/var/log/mysql/mysql-slow.log*",
		fmt.Sprintf("/var/lib/mysql/%s-slow.log", builtin["hostname"]),
	}
	if runtime.GOOS == "darwin" {
		expectedPaths = []interface{}{
			fmt.Sprintf("/usr/local/var/mysql/%s-slow.log*", builtin["hostname"]),
		}
	}
	if runtime.GOOS == "windows" {
		expectedPaths = []interface{}{
			"c:/programdata/MySQL/MySQL Server*/mysql-slow.log*",
		}
	}

	assert.Equal(t, expectedPaths, vars["paths"])
}

func TestResolveVariable(t *testing.T) {
	tests := []struct {
		Value    interface{}
		Vars     map[string]interface{}
		Expected interface{}
	}{
		{
			Value: "test-{{.value}}",
			Vars: map[string]interface{}{
				"value": 2,
			},
			Expected: "test-2",
		},
		{
			Value: []interface{}{"test-{{.value}}", "test1-{{.value}}"},
			Vars: map[string]interface{}{
				"value": 2,
			},
			Expected: []interface{}{"test-2", "test1-2"},
		},
	}

	for _, test := range tests {
		result, err := resolveVariable(test.Vars, test.Value)
		assert.NoError(t, err)
		assert.Equal(t, test.Expected, result)
	}
}

func TestGetInputConfigNginx(t *testing.T) {
	fs := getModuleForTesting(t, "nginx", "access")
	assert.NoError(t, fs.Read("5.2.0"))

	cfg, err := fs.getInputConfig()
	assert.NoError(t, err)

	assert.True(t, cfg.HasField("paths"))
	assert.True(t, cfg.HasField("exclude_files"))
	assert.True(t, cfg.HasField("pipeline"))
	pipelineID, err := cfg.String("pipeline", -1)
	assert.NoError(t, err)
	assert.Equal(t, "filebeat-5.2.0-nginx-access-default", pipelineID)
}

func TestGetInputConfigNginxOverrides(t *testing.T) {
	modulesPath, err := filepath.Abs("../module")
	assert.NoError(t, err)
	fs, err := New(modulesPath, "access", &ModuleConfig{Module: "nginx"}, &FilesetConfig{
		Input: map[string]interface{}{
			"close_eof": true,
		},
	})
	assert.NoError(t, err)

	assert.NoError(t, fs.Read("5.2.0"))

	cfg, err := fs.getInputConfig()
	assert.NoError(t, err)

	assert.True(t, cfg.HasField("paths"))
	assert.True(t, cfg.HasField("exclude_files"))
	assert.True(t, cfg.HasField("close_eof"))
	assert.True(t, cfg.HasField("pipeline"))
	pipelineID, err := cfg.String("pipeline", -1)
	assert.NoError(t, err)
	assert.Equal(t, "filebeat-5.2.0-nginx-access-default", pipelineID)

	moduleName, err := cfg.String("_module_name", -1)
	assert.NoError(t, err)
	assert.Equal(t, "nginx", moduleName)

	filesetName, err := cfg.String("_fileset_name", -1)
	assert.NoError(t, err)
	assert.Equal(t, "access", filesetName)
}

func TestGetPipelineNginx(t *testing.T) {
	fs := getModuleForTesting(t, "nginx", "access")
	assert.NoError(t, fs.Read("5.2.0"))

	version := common.MustNewVersion("5.2.0")
	pipelineID, content, err := fs.GetPipeline(*version)
	assert.NoError(t, err)
	assert.Equal(t, "filebeat-5.2.0-nginx-access-default", pipelineID)
	assert.Contains(t, content, "description")
	assert.Contains(t, content, "processors")
}

func TestGetPipelineConvertTS(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("fileset", "modules"))

	// load system/syslog
	modulesPath, err := filepath.Abs("../module")
	assert.NoError(t, err)
	fs, err := New(modulesPath, "syslog", &ModuleConfig{Module: "system"}, &FilesetConfig{
		Var: map[string]interface{}{
			"convert_timezone": true,
		},
	})
	assert.NoError(t, err)
	assert.NoError(t, fs.Read("6.1.0"))

	cases := map[string]struct {
		Beat     string
		Timezone bool
	}{
		"6.0.0": {Timezone: false},
		"6.1.0": {Timezone: true},
		"6.2.0": {Timezone: true},
	}

	for esVersion, cfg := range cases {
		pipelineName := "filebeat-6.1.0-system-syslog-pipeline"

		t.Run(fmt.Sprintf("es=%v", esVersion), func(t *testing.T) {
			ver := common.MustNewVersion(esVersion)
			pipelineID, content, err := fs.GetPipeline(*ver)
			require.NoError(t, err)
			assert.Equal(t, pipelineName, pipelineID)

			marshaled, err := json.Marshal(content)
			require.NoError(t, err)
			if cfg.Timezone {
				assert.Contains(t, string(marshaled), "beat.timezone")
			} else {
				assert.NotContains(t, string(marshaled), "beat.timezone")
			}
		})
	}
}
