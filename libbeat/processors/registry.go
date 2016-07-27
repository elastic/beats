package processors

import (
	"fmt"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

type Processor interface {
	Run(event common.MapStr) (common.MapStr, error)
	String() string
}

type Constructor func(config common.Config) (Processor, error)

var constructors = map[string]Constructor{}

func RegisterPlugin(name string, constructor Constructor) error {

	logp.Debug("processors", "Register plugin %s", name)

	if _, exists := constructors[name]; exists {
		return fmt.Errorf("plugin %s already registered", name)
	}
	constructors[name] = NewConditional(constructor)
	return nil
}
