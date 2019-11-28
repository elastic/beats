package program

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/x-pack/agent/pkg/agent/transpiler"
)

func placeHolder(t *testing.T) {}

func TestGroupBy(t *testing.T) {
	t.Run("only named output", func(t *testing.T) {
		sConfig := map[string]interface{}{
			"monitoring": map[string]interface{}{
				"elasticsearch": map[string]interface{}{
					"hosts": "localhost",
				},
			},
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
			"streams": []map[string]interface{}{
				map[string]interface{}{
					"type": "log",
					"path": "/var/log/hello.log",
					"output": map[string]interface{}{
						"use_output": "special",
					},
				},
				map[string]interface{}{
					"type": "metrics/system",
					"output": map[string]interface{}{
						"use_output": "special",
					},
				},
				map[string]interface{}{
					"type": "log",
					"path": "/var/log/infosec.log",
					"output": map[string]interface{}{
						"use_output": "infosec1",
						"pipeline":   "custompipeline",
						"index_name": "myindex",
					},
				},
			},
		}

		ast, err := transpiler.NewAST(sConfig)
		require.NoError(t, err)

		grouped, err := groupBy(ast)
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
			"streams": []map[string]interface{}{
				map[string]interface{}{
					"type": "log",
					"path": "/var/log/hello.log",
					"output": map[string]interface{}{
						"use_output": "special",
					},
				},
				map[string]interface{}{
					"type": "metrics/system",
					"output": map[string]interface{}{
						"use_output": "special",
					},
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
					"output": map[string]interface{}{
						"use_output": "infosec1",
						"pipeline":   "custompipeline",
						"index_name": "myindex",
					},
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

	t.Run("outputs with monitoring options", placeHolder)

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
			"streams": []map[string]interface{}{
				map[string]interface{}{
					"type": "log",
					"path": "/var/log/hello.log",
				},
				map[string]interface{}{
					"type": "metrics/system",
				},
				map[string]interface{}{
					"type": "log",
					"path": "/var/log/infosec.log",
					"output": map[string]interface{}{
						"use_output": "donotexist",
						"pipeline":   "custompipeline",
						"index_name": "myindex",
					},
				},
			},
		}

		ast, err := transpiler.NewAST(sConfig)
		require.NoError(t, err)

		_, err = groupBy(ast)
		require.Error(t, err)
	})

	t.Run("only default output", func(t *testing.T) {
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
			"streams": []map[string]interface{}{
				map[string]interface{}{
					"type": "log",
					"path": "/var/log/hello.log",
				},
				map[string]interface{}{
					"type": "metrics/system",
				},
				map[string]interface{}{
					"type": "log",
					"path": "/var/log/infosec.log",
				},
			},
		}

		ast, err := transpiler.NewAST(sConfig)
		require.NoError(t, err)

		grouped, err := groupBy(ast)
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
			"streams": []map[string]interface{}{
				map[string]interface{}{
					"type": "log",
					"path": "/var/log/hello.log",
				},
				map[string]interface{}{
					"type": "metrics/system",
				},
				map[string]interface{}{
					"type": "log",
					"path": "/var/log/infosec.log",
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
			"streams": []map[string]interface{}{
				map[string]interface{}{
					"type": "log",
					"path": "/var/log/hello.log",
				},
				map[string]interface{}{
					"type": "metrics/system",
				},
				map[string]interface{}{
					"type": "log",
					"path": "/var/log/infosec.log",
					"output": map[string]interface{}{
						"use_output": "infosec1",
						"pipeline":   "custompipeline",
						"index_name": "myindex",
					},
				},
			},
		}

		ast, err := transpiler.NewAST(sConfig)
		require.NoError(t, err)

		grouped, err := groupBy(ast)
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
					"output": map[string]interface{}{
						"use_output": "infosec1",
						"pipeline":   "custompipeline",
						"index_name": "myindex",
					},
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
