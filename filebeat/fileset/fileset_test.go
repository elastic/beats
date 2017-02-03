// +build !integration

package fileset

import (
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
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
	assert.Equal(t, manifest.IngestPipeline, "ingest/{{.pipeline}}.json")
	assert.Equal(t, manifest.Prospector, "config/nginx-access.yml")

	vars := manifest.Vars
	assert.Equal(t, "paths", vars[0]["name"])
	path := (vars[0]["default"]).([]interface{})[0].(string)
	assert.Equal(t, path, "/var/log/nginx/access.log*")

	assert.Equal(t, "pipeline", vars[1]["name"])
	assert.Equal(t, "with_plugins", vars[1]["default"])
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

	assert.Equal(t, "with_plugins", vars["pipeline"])
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

func TestGetProspectorConfigNginx(t *testing.T) {
	fs := getModuleForTesting(t, "nginx", "access")
	assert.NoError(t, fs.Read())

	cfg, err := fs.getProspectorConfig()
	assert.NoError(t, err)

	assert.True(t, cfg.HasField("paths"))
	assert.True(t, cfg.HasField("exclude_files"))
	assert.True(t, cfg.HasField("pipeline"))
	pipelineID, err := cfg.String("pipeline", -1)
	assert.NoError(t, err)
	assert.Equal(t, "nginx-access-with_plugins", pipelineID)
}

func TestGetProspectorConfigNginxOverrides(t *testing.T) {
	modulesPath, err := filepath.Abs("../module")
	assert.NoError(t, err)
	fs, err := New(modulesPath, "access", &ModuleConfig{Module: "nginx"}, &FilesetConfig{
		Prospector: map[string]interface{}{
			"close_eof": true,
		},
	})
	assert.NoError(t, err)

	assert.NoError(t, fs.Read())

	cfg, err := fs.getProspectorConfig()
	assert.NoError(t, err)

	assert.True(t, cfg.HasField("paths"))
	assert.True(t, cfg.HasField("exclude_files"))
	assert.True(t, cfg.HasField("close_eof"))
	assert.True(t, cfg.HasField("pipeline"))
	pipelineID, err := cfg.String("pipeline", -1)
	assert.NoError(t, err)
	assert.Equal(t, "nginx-access-with_plugins", pipelineID)

}

func TestGetPipelineNginx(t *testing.T) {
	fs := getModuleForTesting(t, "nginx", "access")
	assert.NoError(t, fs.Read())

	pipelineID, content, err := fs.GetPipeline()
	assert.NoError(t, err)
	assert.Equal(t, "nginx-access-with_plugins", pipelineID)
	assert.Contains(t, content, "description")
	assert.Contains(t, content, "processors")
}
