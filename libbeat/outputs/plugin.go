package outputs

import (
	"errors"
	"fmt"

	p "github.com/elastic/beats/libbeat/plugin"
)

type outputPlugin struct {
	name    string
	factory Factory
}

var pluginKey = "libbeat.out"

func Plugin(name string, f Factory) map[string][]interface{} {
	return p.MakePlugin(pluginKey, outputPlugin{name, f})
}

func init() {
	p.MustRegisterLoader(pluginKey, func(ifc interface{}) error {
		b, ok := ifc.(outputPlugin)
		if !ok {
			return errors.New("plugin does not match output plugin type")
		}

		name := b.name
		if outputReg[name] != nil {
			return fmt.Errorf("output type %v already registered", name)
		}

		RegisterType(name, b.factory)
		return nil
	})
}
