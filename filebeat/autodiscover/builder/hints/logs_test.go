package hints

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
	"github.com/elastic/beats/libbeat/paths"
)

func TestGenerateHints(t *testing.T) {
	tests := []struct {
		msg    string
		event  bus.Event
		len    int
		result common.MapStr
	}{
		{
			msg: "Hints without host should return nothing",
			event: bus.Event{
				"hints": common.MapStr{
					"metrics": common.MapStr{
						"module": "prometheus",
					},
				},
			},
			len:    0,
			result: common.MapStr{},
		},
		{
			msg: "Empty event hints should return default config",
			event: bus.Event{
				"host": "1.2.3.4",
				"kubernetes": common.MapStr{
					"container": common.MapStr{
						"name": "foobar",
						"id":   "abc",
					},
				},
				"container": common.MapStr{
					"name": "foobar",
					"id":   "abc",
				},
			},
			len: 1,
			result: common.MapStr{
				"type": "docker",
				"containers": map[string]interface{}{
					"ids": []interface{}{"abc"},
				},
			},
		},
		{
			msg: "Hint with include|exclude_lines must be part of the input config",
			event: bus.Event{
				"host": "1.2.3.4",
				"kubernetes": common.MapStr{
					"container": common.MapStr{
						"name": "foobar",
						"id":   "abc",
					},
				},
				"container": common.MapStr{
					"name": "foobar",
					"id":   "abc",
				},
				"hints": common.MapStr{
					"logs": common.MapStr{
						"include_lines": "^test, ^test1",
						"exclude_lines": "^test2, ^test3",
					},
				},
			},
			len: 1,
			result: common.MapStr{
				"type": "docker",
				"containers": map[string]interface{}{
					"ids": []interface{}{"abc"},
				},
				"include_lines": []interface{}{"^test", "^test1"},
				"exclude_lines": []interface{}{"^test2", "^test3"},
			},
		},
		{
			msg: "Hint with multiline config must have a multiline in the input config",
			event: bus.Event{
				"host": "1.2.3.4",
				"kubernetes": common.MapStr{
					"container": common.MapStr{
						"name": "foobar",
						"id":   "abc",
					},
				},
				"container": common.MapStr{
					"name": "foobar",
					"id":   "abc",
				},
				"hints": common.MapStr{
					"logs": common.MapStr{
						"multiline": common.MapStr{
							"pattern": "^test",
							"negate":  "true",
						},
					},
				},
			},
			len: 1,
			result: common.MapStr{
				"type": "docker",
				"containers": map[string]interface{}{
					"ids": []interface{}{"abc"},
				},
				"multiline": map[string]interface{}{
					"pattern": "^test",
					"negate":  "true",
				},
			},
		},
		{
			msg: "Hint with module should attach input to its filesets",
			event: bus.Event{
				"host": "1.2.3.4",
				"kubernetes": common.MapStr{
					"container": common.MapStr{
						"name": "foobar",
						"id":   "abc",
					},
				},
				"container": common.MapStr{
					"name": "foobar",
					"id":   "abc",
				},
				"hints": common.MapStr{
					"logs": common.MapStr{
						"module": "apache2",
					},
				},
			},
			len: 1,
			result: common.MapStr{
				"module": "apache2",
				"error": map[string]interface{}{
					"enabled": true,
					"input": map[string]interface{}{
						"type": "docker",
						"containers": map[string]interface{}{
							"stream": "all",
							"ids":    []interface{}{"abc"},
						},
					},
				},
				"access": map[string]interface{}{
					"enabled": true,
					"input": map[string]interface{}{
						"type": "docker",
						"containers": map[string]interface{}{
							"stream": "all",
							"ids":    []interface{}{"abc"},
						},
					},
				},
			},
		},
		{
			msg: "Hint with module should honor defined filesets",
			event: bus.Event{
				"host": "1.2.3.4",
				"kubernetes": common.MapStr{
					"container": common.MapStr{
						"name": "foobar",
						"id":   "abc",
					},
				},
				"container": common.MapStr{
					"name": "foobar",
					"id":   "abc",
				},
				"hints": common.MapStr{
					"logs": common.MapStr{
						"module":  "apache2",
						"fileset": "access",
					},
				},
			},
			len: 1,
			result: common.MapStr{
				"module": "apache2",
				"access": map[string]interface{}{
					"enabled": true,
					"input": map[string]interface{}{
						"type": "docker",
						"containers": map[string]interface{}{
							"stream": "all",
							"ids":    []interface{}{"abc"},
						},
					},
				},
				"error": map[string]interface{}{
					"enabled": false,
					"input": map[string]interface{}{
						"type": "docker",
						"containers": map[string]interface{}{
							"stream": "all",
							"ids":    []interface{}{"abc"},
						},
					},
				},
			},
		},
		{
			msg: "Hint with module should honor defined filesets with streams",
			event: bus.Event{
				"host": "1.2.3.4",
				"kubernetes": common.MapStr{
					"container": common.MapStr{
						"name": "foobar",
						"id":   "abc",
					},
				},
				"container": common.MapStr{
					"name": "foobar",
					"id":   "abc",
				},
				"hints": common.MapStr{
					"logs": common.MapStr{
						"module":         "apache2",
						"fileset.stdout": "access",
						"fileset.stderr": "error",
					},
				},
			},
			len: 1,
			result: common.MapStr{
				"module": "apache2",
				"access": map[string]interface{}{
					"enabled": true,
					"input": map[string]interface{}{
						"type": "docker",
						"containers": map[string]interface{}{
							"stream": "stdout",
							"ids":    []interface{}{"abc"},
						},
					},
				},
				"error": map[string]interface{}{
					"enabled": true,
					"input": map[string]interface{}{
						"type": "docker",
						"containers": map[string]interface{}{
							"stream": "stderr",
							"ids":    []interface{}{"abc"},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		cfg, _ := common.NewConfigFrom(map[string]interface{}{
			"type": "docker",
			"containers": map[string]interface{}{
				"ids": []string{
					"${data.container.id}",
				},
			},
		})

		// Configure path for modules access
		abs, _ := filepath.Abs("../../..")
		err := paths.InitPaths(&paths.Path{
			Home: abs,
		})

		l, err := NewLogHints(cfg)
		if err != nil {
			t.Fatal(err)
		}

		cfgs := l.CreateConfig(test.event)
		assert.Equal(t, len(cfgs), test.len, test.msg)

		if test.len != 0 {
			config := common.MapStr{}
			err := cfgs[0].Unpack(&config)
			assert.Nil(t, err, test.msg)

			assert.Equal(t, test.result, config, test.msg)
		}

	}
}
