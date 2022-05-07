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
	"fmt"
	"path/filepath"
	"runtime"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	conf "github.com/elastic/elastic-agent-libs/config"
)

func makeTestInfo(version string) beat.Info {
	return beat.Info{
		IndexPrefix: "filebeat",
		Version:     version,
	}
}

func getModuleForTesting(t *testing.T, module, fileset string) *Fileset {
	modulesPath, err := filepath.Abs("../module")
	require.NoError(t, err)
	fs, err := New(modulesPath, fileset, module, &FilesetConfig{})
	require.NoError(t, err)

	return fs
}

func TestLoadManifestNginx(t *testing.T) {
	fs := getModuleForTesting(t, "nginx", "access")

	manifest, err := fs.readManifest()
	require.NoError(t, err)
	assert.Equal(t, manifest.ModuleVersion, "1.0")
	assert.Equal(t, manifest.IngestPipeline, []string{"ingest/pipeline.yml"})
	assert.Equal(t, manifest.Input, "config/nginx-access.yml")

	vars := manifest.Vars
	assert.Equal(t, "paths", vars[0]["name"])
	path := (vars[0]["default"]).([]interface{})[0].(string)
	assert.Equal(t, path, "/var/log/nginx/access.log*")
}

func TestGetBuiltinVars(t *testing.T) {
	fs := getModuleForTesting(t, "nginx", "access")

	vars, err := fs.getBuiltinVars(makeTestInfo("6.6.0"))
	require.NoError(t, err)

	assert.IsType(t, vars["hostname"], "a-mac-with-esc-key")
	assert.IsType(t, vars["domain"], "local")
	assert.Equal(t, "nginx", vars["module"])
	assert.Equal(t, "access", vars["fileset"])
	assert.Equal(t, "6.6.0", vars["beatVersion"])
}

func TestEvaluateVarsNginx(t *testing.T) {
	fs := getModuleForTesting(t, "nginx", "access")

	var err error
	fs.manifest, err = fs.readManifest()
	require.NoError(t, err)

	vars, err := fs.evaluateVars(makeTestInfo("6.6.0"))
	require.NoError(t, err)

	builtin := vars["builtin"].(map[string]interface{})
	assert.IsType(t, "a-mac-with-esc-key", builtin["hostname"])
	assert.IsType(t, "local", builtin["domain"])

	assert.IsType(t, []interface{}{"/usr/local/var/log/nginx/access.log*"}, vars["paths"])
}

func TestEvaluateVarsNginxOverride(t *testing.T) {
	modulesPath, err := filepath.Abs("../module")
	require.NoError(t, err)
	fs, err := New(modulesPath, "access", "nginx", &FilesetConfig{
		Var: map[string]interface{}{
			"pipeline": "no_plugins",
		},
	})
	require.NoError(t, err)

	fs.manifest, err = fs.readManifest()
	require.NoError(t, err)

	vars, err := fs.evaluateVars(makeTestInfo("6.6.0"))
	require.NoError(t, err)

	assert.Equal(t, "no_plugins", vars["pipeline"])
}

func TestEvaluateVarsMySQL(t *testing.T) {
	fs := getModuleForTesting(t, "mysql", "slowlog")

	var err error
	fs.manifest, err = fs.readManifest()
	require.NoError(t, err)

	vars, err := fs.evaluateVars(makeTestInfo("6.6.0"))
	require.NoError(t, err)

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
				"value":   2,
				"builtin": map[string]interface{}{},
			},
			Expected: "test-2",
		},
		{
			Value: []interface{}{"test-{{.value}}", "test1-{{.value}}"},
			Vars: map[string]interface{}{
				"value":   2,
				"builtin": map[string]interface{}{},
			},
			Expected: []interface{}{"test-2", "test1-2"},
		},
	}

	for _, test := range tests {
		result, err := resolveVariable(test.Vars, test.Value)
		require.NoError(t, err)
		assert.Equal(t, test.Expected, result)
	}
}

func TestGetInputConfigNginx(t *testing.T) {
	fs := getModuleForTesting(t, "nginx", "access")
	require.NoError(t, fs.Read(makeTestInfo("5.2.0")))

	cfg, err := fs.getInputConfig()
	require.NoError(t, err)

	assert.True(t, cfg.HasField("paths"))
	assert.True(t, cfg.HasField("exclude_files"))
	assert.True(t, cfg.HasField("pipeline"))
	pipelineID, err := cfg.String("pipeline", -1)
	require.NoError(t, err)
	assert.Equal(t, "filebeat-5.2.0-nginx-access-pipeline", pipelineID)
}

func TestGetInputConfigNginxOverrides(t *testing.T) {
	modulesPath, err := filepath.Abs("../module")
	require.NoError(t, err)

	tests := map[string]struct {
		input      map[string]interface{}
		expectedFn require.ValueAssertionFunc
	}{
		"close_eof": {
			map[string]interface{}{
				"close_eof": true,
			},
			func(t require.TestingT, cfg interface{}, rest ...interface{}) {
				c, ok := cfg.(*conf.C)
				if !ok {
					t.FailNow()
				}

				require.True(t, c.HasField("close_eof"))
				v, err := c.Bool("close_eof", -1)
				require.NoError(t, err)
				require.True(t, v)

				pipelineID, err := c.String("pipeline", -1)
				require.NoError(t, err)
				assert.Equal(t, "filebeat-5.2.0-nginx-access-pipeline", pipelineID)
			},
		},
		"pipeline": {
			map[string]interface{}{
				"pipeline": "foobar",
			},
			func(t require.TestingT, cfg interface{}, rest ...interface{}) {
				c, ok := cfg.(*conf.C)
				if !ok {
					t.FailNow()
				}

				v, err := c.String("pipeline", -1)
				require.NoError(t, err)
				require.Equal(t, "foobar", v)
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			fs, err := New(modulesPath, "access", "nginx", &FilesetConfig{
				Input: test.input,
			})
			require.NoError(t, err)

			require.NoError(t, fs.Read(makeTestInfo("5.2.0")))

			cfg, err := fs.getInputConfig()
			require.NoError(t, err)

			assert.True(t, cfg.HasField("paths"))
			assert.True(t, cfg.HasField("exclude_files"))
			assert.True(t, cfg.HasField("pipeline"))

			test.expectedFn(t, cfg)

			moduleName, err := cfg.String("_module_name", -1)
			require.NoError(t, err)
			assert.Equal(t, "nginx", moduleName)

			filesetName, err := cfg.String("_fileset_name", -1)
			require.NoError(t, err)
			assert.Equal(t, "access", filesetName)
		})
	}
}

func TestGetPipelineNginx(t *testing.T) {
	fs := getModuleForTesting(t, "nginx", "access")
	require.NoError(t, fs.Read(makeTestInfo("5.2.0")))

	version := common.MustNewVersion("5.2.0")
	pipelines, err := fs.GetPipelines(*version)
	require.NoError(t, err)
	assert.Len(t, pipelines, 1)

	pipeline := pipelines[0]
	assert.Equal(t, "filebeat-5.2.0-nginx-access-pipeline", pipeline.id)
	assert.Contains(t, pipeline.contents, "description")
	assert.Contains(t, pipeline.contents, "processors")
}

func TestGetTemplateFunctions(t *testing.T) {
	vars := map[string]interface{}{
		"builtin": map[string]interface{}{},
	}
	templateFunctions, err := getTemplateFunctions(vars)
	require.NoError(t, err)
	assert.IsType(t, template.FuncMap{}, templateFunctions)
	assert.Contains(t, templateFunctions, "inList")
	assert.Contains(t, templateFunctions, "tojson")
	assert.Contains(t, templateFunctions, "IngestPipeline")
}
