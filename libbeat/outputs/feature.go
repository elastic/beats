package outputs

import (
	"fmt"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/feature"
)

// Namespace exposes the output type.
var Namespace = "libbeat.output"

// Factory is used by output plugins to build an output instance
type Factory func(
	beat beat.Info,
	stats Observer,
	cfg *common.Config) (Group, error)

// Group configures and combines multiple clients into load-balanced group of clients
// being managed by the publisher pipeline.
type Group struct {
	Clients   []Client
	BatchSize int
	Retry     int
}

// RegisterType registers a new output type.
func RegisterType(name string, f Factory) {
	feature.MustRegister(Feature(name, f, feature.Undefined))
}

// FindFactory finds an output type its factory if available.
func FindFactory(name string) (Factory, error) {
	f, err := feature.Registry.Lookup(Namespace, name)
	if err != nil {
		return nil, err
	}

	factory, ok := f.Factory().(Factory)
	if !ok {
		return nil, fmt.Errorf("invalid output type, received: %T", f.Factory())
	}

	return factory, nil
}

// Load creates and configures a output Group using a configuration object..
func Load(info beat.Info, stats Observer, name string, config *common.Config) (Group, error) {
	factory, err := FindFactory(name)
	if err != nil {
		return Group{}, err
	}

	if err := cfgwarn.CheckRemoved5xSetting(config, "flush_interval"); err != nil {
		return Fail(err)
	}

	if stats == nil {
		stats = NewNilObserver()
	}
	return factory(info, stats, config)
}

// Feature creates a new output.
func Feature(name string, factory Factory, stability feature.Stability) *feature.Feature {
	return feature.New(Namespace, name, factory, stability)
}
