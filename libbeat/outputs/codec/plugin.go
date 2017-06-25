package codec

import (
	"errors"
	"fmt"

	"github.com/elastic/beats/libbeat/plugin"
)

type codecPlugin struct {
	name    string
	factory Factory
}

var pluginKey = "libbeat.output.codec"

func Plugin(name string, f Factory) map[string][]interface{} {
	return plugin.MakePlugin(name, codecPlugin{name, f})
}

func init() {
	plugin.MustRegisterLoader(pluginKey, func(ifc interface{}) (err error) {
		b, ok := ifc.(codecPlugin)
		if !ok {
			return errors.New("plugin does not match output codec plugin type")
		}

		defer func() {
			if msg := recover(); msg != nil {
				err = fmt.Errorf("%s", msg)
			}
		}()

		RegisterType(b.name, b.factory)
		return
	})
}
