package processors

import (
	"errors"
	"fmt"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/feature"
	p "github.com/elastic/beats/libbeat/plugin"
)

// Namespace exposes the processor type.
var Namespace = "libbeat.processor"

type processorPlugin struct {
	name   string
	constr Constructor
}

func Plugin(name string, c Constructor) map[string][]interface{} {
	return p.MakePlugin(Namespace, processorPlugin{name, c})
}

func init() {
	p.MustRegisterLoader(Namespace, func(ifc interface{}) error {
		p, ok := ifc.(processorPlugin)
		if !ok {
			return errors.New("plugin does not match processor plugin type")
		}

		f := feature.New(Namespace, p.name, p.constr, feature.Undefined)
		return feature.Register(f)
	})
}

type Constructor func(config *common.Config) (Processor, error)

// RegisterPlugin is a backward compatible shim over the new Feature api.
func RegisterPlugin(name string, factory Constructor) {
	f := Feature(name, factory, feature.Undefined)
	feature.MustRegister(f)
}

// Feature define a new feature.
func Feature(name string, factory Constructor, stability feature.Stability) *feature.Feature {
	return feature.New(Namespace, name, factory, stability)
}

// Find returns the processor factory and wrap it into a NewConditonal.
func Find(name string) (Constructor, error) {
	f, err := feature.Registry.Find(Namespace, name)
	if err != nil {
		return nil, err
	}

	factory, ok := f.Factory().(Constructor)
	if !ok {
		return nil, fmt.Errorf("invalid processor type, received: %T", f.Factory())
	}

	return NewConditional(factory), nil
}
