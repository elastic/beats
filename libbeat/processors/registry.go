package processors

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

type Processor interface {
	Run(event common.MapStr) (common.MapStr, error)
	String() string
}

type Constructor func(config common.Config) (Processor, error)

var registry = NewNamespace()

func RegisterPlugin(name string, constructor Constructor) {
	logp.Debug("processors", "Register plugin %s", name)

	err := registry.Register(name, constructor)
	if err != nil {
		panic(err)
	}
}
