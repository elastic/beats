package prospector

import (
	"errors"

	"github.com/elastic/beats/libbeat/plugin"
)

type prospectorPlugin struct {
	name    string
	factory Factory
}

const pluginKey = "filebeat.prospector"

func init() {
	plugin.MustRegisterLoader(pluginKey, func(ifc interface{}) error {
		p, ok := ifc.(prospectorPlugin)
		if !ok {
			return errors.New("plugin does not match filebeat prospector plugin type")
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
	return plugin.MakePlugin(pluginKey, prospectorPlugin{module, factory})
}
