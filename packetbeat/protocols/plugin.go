package protocols

import (
	"errors"

	"github.com/elastic/beats/libbeat/plugin"
	"github.com/elastic/beats/packetbeat/protos"
)

type protocolPlugin struct {
	name string
	p    protos.ProtocolPlugin
}

const pluginKey = "packetbeat.protocol"

func init() {
	plugin.MustRegisterLoader(pluginKey, func(ifc interface{}) error {
		p, ok := ifc.(protocolPlugin)
		if !ok {
			return errors.New("plugin does not match protocol plugin type")
		}

		protos.Register(p.name, p.p)
		return nil
	})
}

func Plugin(name string, p protos.ProtocolPlugin) map[string][]interface{} {
	return plugin.MakePlugin(pluginKey, protocolPlugin{name, p})
}
