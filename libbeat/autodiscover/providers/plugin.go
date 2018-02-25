package providers

import (
	"errors"

	"github.com/elastic/beats/libbeat/autodiscover"
	p "github.com/elastic/beats/libbeat/plugin"
)

type providerPlugin struct {
	name     string
	provider autodiscover.ProviderBuilder
}

var pluginKey = "libbeat.autodiscover.provider"

// Plugin accepts a ProviderBuilder to be registered as a plugin
func Plugin(name string, provider autodiscover.ProviderBuilder) map[string][]interface{} {
	return p.MakePlugin(pluginKey, providerPlugin{name, provider})
}

func init() {
	p.MustRegisterLoader(pluginKey, func(ifc interface{}) error {
		prov, ok := ifc.(providerPlugin)
		if !ok {
			return errors.New("plugin does not match processor plugin type")
		}

		return autodiscover.Registry.AddProvider(prov.name, prov.provider)
	})
}
