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

func TestPlugins(t *testing.T) {
	builder := func(i interface{}) (interface{}, error) {
		return i, nil
	}

	pluginA := "A"
	pluginB := "B"
	pluginC := "C"

	key := PluginTypeKey("hello")

	t.Run("default to appending order", func(t *testing.T) {
		r := New(logp.NewLogger("testing"))
		err := r.RegisterType(key, builder)
		if !assert.NoError(t, err) {
			return
		}

		r.RegisterPlugin(key, "a", pluginA)
		r.RegisterPlugin(key, "b", pluginB)

		_, l, err := r.Plugins(key)
		if !assert.NoError(t, err) {
			return
		}
		assert.Equal(t, []Plugin{pluginA, pluginB}, l)
	})

	t.Run("allow to register a plugin before another one in the ordered set if the plugin exist", func(t *testing.T) {
		r := New(logp.NewLogger("testing"))
		err := r.RegisterType(key, builder)
		if !assert.NoError(t, err) {
			return
		}

		r.RegisterPlugin(key, "a", pluginA)
		r.RegisterPlugin(key, "c", pluginC)
		r.OrderedRegisterPlugin(key, Before, "c", "b", pluginB)

		_, l, err := r.Plugins(key)
		if !assert.NoError(t, err) {
			return
		}
		assert.Equal(t, []Plugin{pluginA, pluginB, pluginC}, l)
	})
}
