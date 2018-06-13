package registry

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/logp"
)

func TestRegisterPluginType(t *testing.T) {
	builder := func(i interface{}) (interface{}, error) {
		return i, nil
	}

	t.Run("when it does not exist", func(t *testing.T) {
		r := New(logp.NewLogger("testing"))
		err := r.RegisterType("hello", builder)
		assert.NoError(t, err)
	})

	t.Run("when it already exist", func(t *testing.T) {
		r := New(logp.NewLogger("testing"))
		err := r.RegisterType("hello", builder)
		if !assert.NoError(t, err) {
			return
		}
		err = r.RegisterType("hello", builder)
		assert.Error(t, err)
	})
}

func TestRemovePluginType(t *testing.T) {
	t.Run("when it does not exist", func(t *testing.T) {
		r := New(logp.NewLogger("testing"))
		err := r.RemoveType("hello")
		assert.Error(t, err)
	})

	t.Run("when it already exist", func(t *testing.T) {
		builder := func(i interface{}) (interface{}, error) {
			return i, nil
		}

		r := New(logp.NewLogger("testing"))
		err := r.RegisterType("hello", builder)
		if !assert.NoError(t, err) {
			return
		}
		err = r.RemoveType("hello")
		assert.NoError(t, err)
	})
}

func TestAddPlugin(t *testing.T) {
	builder := func(i interface{}) (interface{}, error) {
		return i, nil
	}

	plugin := struct{}{}

	t.Run("when the plugin type doesn't exist", func(t *testing.T) {
		r := New(logp.NewLogger("testing"))
		err := r.RegisterPlugin("main", "superplugin", plugin)
		assert.Error(t, err)
	})

	t.Run("when the plugin type exists", func(t *testing.T) {
		r := New(logp.NewLogger("testing"))
		err := r.RegisterType("hello", builder)
		if !assert.NoError(t, err) {
			return
		}
		err = r.RegisterPlugin("hello", "superplugin", plugin)
		if !assert.NoError(t, err) {
			return
		}

		b, p, err := r.Plugin("hello", "superplugin")
		if !assert.NoError(t, err) {
			return
		}

		b1, _ := builder(1)
		b2, _ := b(1)
		if !assert.Equal(t, b1, b2) {
			return
		}

		if !assert.Equal(t, plugin, p) {
			return
		}
	})
}

func TestRemovePlugin(t *testing.T) {
	builder := func(i interface{}) (interface{}, error) {
		return i, nil
	}

	plugin := struct{}{}

	t.Run("when the plugin type doesn't exist", func(t *testing.T) {
		r := New(logp.NewLogger("testing"))
		err := r.RemovePlugin("hello", "myplugin")
		assert.Error(t, err)
	})

	t.Run("when the plugin type exists and the plugin exists", func(t *testing.T) {
		r := New(logp.NewLogger("testing"))
		err := r.RegisterType("hello", builder)
		if !assert.NoError(t, err) {
			return
		}
		err = r.RegisterPlugin("hello", "superplugin", plugin)
		if !assert.NoError(t, err) {
			return
		}

		err = r.RemovePlugin("hello", "superplugin")
		assert.NoError(t, err)
	})

	t.Run("when the plugin type exists and the plugin doesn't exist", func(t *testing.T) {
		r := New(logp.NewLogger("testing"))
		err := r.RegisterType("hello", builder)
		if !assert.NoError(t, err) {
			return
		}
		err = r.RemovePlugin("hello", "superplugin")
		assert.Error(t, err)
	})
}

func TestPluginsInsert(t *testing.T) {
	builder := func(i interface{}) (interface{}, error) {
		return i, nil
	}

	pluginA := "A"
	pluginB := "B"
	pluginC := "C"

	key := PluginTypeKey("hello")

	tests := []struct {
		name     string
		add      func(r *Registry)
		expected []Plugin
	}{
		{
			name: "default append at the end",
			add: func(r *Registry) {
				r.RegisterPlugin(key, "a", pluginA)
				r.RegisterPlugin(key, "b", pluginB)
				r.RegisterPlugin(key, "c", pluginC)
			},
			expected: []Plugin{pluginA, pluginB, pluginC},
		},
		{
			name: "allow to register a plugin before another one in the ordered set if the plugin exist",
			add: func(r *Registry) {
				r.RegisterPlugin(key, "a", pluginA)
				r.RegisterPlugin(key, "b", pluginB)
				r.OrderedRegisterPlugin(key, Before, "b", "c", pluginC)
			},
			expected: []Plugin{pluginA, pluginC, pluginB},
		},
		{
			name: "allow to register a plugin before another one in the ordered set if the plugin exist",
			add: func(r *Registry) {
				r.RegisterPlugin(key, "a", pluginA)
				r.RegisterPlugin(key, "b", pluginB)
				r.OrderedRegisterPlugin(key, Before, "a", "c", pluginC)
			},
			expected: []Plugin{pluginC, pluginA, pluginB},
		},
		{
			name: "allow to register a plugin after another one in the ordered set if the plugin exist",
			add: func(r *Registry) {
				r.RegisterPlugin(key, "a", pluginA)
				r.RegisterPlugin(key, "b", pluginB)
				r.OrderedRegisterPlugin(key, After, "a", "c", pluginC)
			},
			expected: []Plugin{pluginA, pluginC, pluginB},
		},
		{
			name: "allow to register a plugin after another one in the ordered set if the plugin exist",
			add: func(r *Registry) {
				r.RegisterPlugin(key, "a", pluginA)
				r.RegisterPlugin(key, "b", pluginB)
				r.OrderedRegisterPlugin(key, After, "b", "c", pluginC)
			},
			expected: []Plugin{pluginA, pluginB, pluginC},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := New(logp.NewLogger("testing"))
			err := r.RegisterType(key, builder)
			if !assert.NoError(t, err) {
				return
			}

			test.add(r)

			_, l, err := r.Plugins(key)
			if !assert.NoError(t, err) {
				return
			}
			assert.Equal(t, test.expected, l)
		})
	}
}
