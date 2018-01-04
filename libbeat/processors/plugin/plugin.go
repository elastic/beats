package plugin

import (
	"plugin"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/processors"
)

type pluginProc struct {
	File string
}

func init() {
	processors.RegisterPlugin("plugin", newPlugin)
}

func newPlugin(c *common.Config) (processors.Processor, error) {
	config := struct {
		File string `config:"file"`
	}{}

	err := c.Unpack(&config)
	if err != nil {
		return nil, errors.Wrap(err, "fail to unpack the plugin processor configuration")
	}

	var plg pluginProc

	plg.File = config.File

	return plg, nil
}

func (p pluginProc) Run(event *beat.Event) (*beat.Event, error) {
	plg, err := plugin.Open(p.File)
	if err != nil {
		panic(err)
	}

	run, err := plg.Lookup("Run")

	if err != nil {
		panic(err)
	}

	return run.(func(*beat.Event) (*beat.Event, error))(event)
}

func (p pluginProc) String() string {
	return "plugin=[file=" + p.File + "]"
}
