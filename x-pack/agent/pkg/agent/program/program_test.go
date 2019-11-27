package program

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/x-pack/agent/pkg/agent/transpiler"
)

func TestGroupBy(t *testing.T) {
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
			"streams": []map[string]interface{}{
				map[string]interface{}{
					"type": "log",
					"path": "/var/log/path1.log",
				},
				map[string]interface{}{
					"type": "metrics/system",
				},
				map[string]interface{}{
					"type": "log",
					"path": "/var/log/path2.log",
					"output": map[string]interface{}{
						"user_output": "infosec1",
					},
				},
			},
		}

		ast, err := transpiler.NewAST(sConfig)
		require.NoError(t, err)

		grouped, err := groupBy(ast)
		require.NoError(t, err)
		require.Equal(t, 2, len(grouped))

		c1, _ := transpiler.NewAST(map[string]interface{}{
			"output": map[string]interface{}{
				"elasticsearch": map[string]interface{}{
					"hosts":    "xxx",
					"username": "myusername",
					"password": "mypassword",
				},
			},
			"streams": []map[string]interface{}{
				map[string]interface{}{
					"type": "log",
					"path": "/var/log/hello.log",
				},
				map[string]interface{}{
					"type": "metrics/system",
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
			"streams": []map[string]interface{}{
				map[string]interface{}{
					"type": "log",
					"path": "/var/log/infosec.log",
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
}
