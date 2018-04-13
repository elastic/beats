package builder

import (
	"errors"

	"github.com/elastic/beats/libbeat/autodiscover"
	p "github.com/elastic/beats/libbeat/plugin"
)

type builderPlugin struct {
	name    string
	builder autodiscover.BuilderConstructor
}

var pluginKey = "libbeat.autodiscover.builder"

// Plugin accepts a BuilderConstructor to be registered as a plugin
func Plugin(name string, b autodiscover.BuilderConstructor) map[string][]interface{} {
	return p.MakePlugin(pluginKey, builderPlugin{name, b})
}

func init() {
	p.MustRegisterLoader(pluginKey, func(ifc interface{}) error {
		b, ok := ifc.(builderPlugin)
		if !ok {
			return errors.New("plugin does not match builder plugin type")
		}

		return autodiscover.Registry.AddBuilder(b.name, b.builder)
	})
}
