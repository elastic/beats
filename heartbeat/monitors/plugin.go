package monitors

import (
	"errors"

	"github.com/elastic/beats/libbeat/plugin"
)

type monitorPlugin struct {
	name    string
	typ     Type
	builder ActiveBuilder
}

var pluginKey = "heartbeat.monitor"

func ActivePlugin(name string, b ActiveBuilder) map[string][]interface{} {
	return plugin.MakePlugin(pluginKey, monitorPlugin{name, ActiveMonitor, b})
}

func init() {
	plugin.MustRegisterLoader(pluginKey, func(ifc interface{}) error {
		p, ok := ifc.(monitorPlugin)
		if !ok {
			return errors.New("plugin does not match monitor plugin type")
		}

		return Registry.Register(p.name, p.typ, p.builder)
	})
}
