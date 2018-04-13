package appenders

import (
	"errors"

	"github.com/elastic/beats/libbeat/autodiscover"
	p "github.com/elastic/beats/libbeat/plugin"
)

type appenderPlugin struct {
	name     string
	appender autodiscover.AppenderBuilder
}

var pluginKey = "libbeat.autodiscover.appender"

// Plugin accepts a AppenderBuilder to be registered as a plugin
func Plugin(name string, appender autodiscover.AppenderBuilder) map[string][]interface{} {
	return p.MakePlugin(pluginKey, appenderPlugin{name, appender})
}

func init() {
	p.MustRegisterLoader(pluginKey, func(ifc interface{}) error {
		app, ok := ifc.(appenderPlugin)
		if !ok {
			return errors.New("plugin does not match appender plugin type")
		}

		return autodiscover.Registry.AddAppender(app.name, app.appender)
	})
}
