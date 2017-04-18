package codecs

import (
	"errors"
	"fmt"

	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/plugin"
)

type codecPlugin struct {
	name    string
	factory outputs.CodecFactory
}

var pluginKey = "libbeat.output.codec"

func Plugin(name string, f outputs.CodecFactory) map[string][]interface{} {
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

		outputs.RegisterOutputCodec(b.name, b.factory)
		return
	})
}
