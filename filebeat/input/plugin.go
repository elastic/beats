package input

import (
	"errors"

	"github.com/elastic/beats/libbeat/plugin"
)

type inputPlugin struct {
	name    string
	factory Factory
}

const pluginKey = "filebeat.input"

func init() {
	plugin.MustRegisterLoader(pluginKey, func(ifc interface{}) error {
		p, ok := ifc.(inputPlugin)
		if !ok {
			return errors.New("plugin does not match filebeat input plugin type")
		}

		if p.factory != nil {
			if err := Register(p.name, p.factory); err != nil {
				return err
			}
		}

		return nil
	})
}

func Plugin(
	module string,
	factory Factory,
) map[string][]interface{} {
	return plugin.MakePlugin(pluginKey, inputPlugin{module, factory})
}
