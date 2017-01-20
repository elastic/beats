package outputs

import (
	"errors"
	"fmt"

	p "github.com/elastic/beats/libbeat/plugin"
)

type outputPlugin struct {
	name    string
	builder OutputBuilder
}

var pluginKey = "libbeat.output"

func Plugin(name string, l OutputBuilder) map[string][]interface{} {
	return p.MakePlugin(pluginKey, outputPlugin{name, l})
}

func init() {
	p.MustRegisterLoader(pluginKey, func(ifc interface{}) error {
		b, ok := ifc.(outputPlugin)
		if !ok {
			return errors.New("plugin does not match output plugin type")
		}

		name := b.name
		if outputsPlugins[name] != nil {
			return fmt.Errorf("output type %v already registered", name)
		}

		RegisterOutputPlugin(name, b.builder)
		return nil
	})
}
