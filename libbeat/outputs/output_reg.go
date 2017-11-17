package outputs

import (
	"fmt"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
)

var outputReg = map[string]Factory{}

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
	if outputReg[name] != nil {
		panic(fmt.Errorf("output type  '%v' exists already", name))
	}
	outputReg[name] = f
}

// FindFactory finds an output type its factory if available.
func FindFactory(name string) Factory {
	return outputReg[name]
}

// Load creates and configures a output Group using a configuration object..
func Load(info beat.Info, stats Observer, name string, config *common.Config) (Group, error) {
	factory := FindFactory(name)
	if factory == nil {
		return Group{}, fmt.Errorf("output type %v undefined", name)
	}

	if err := cfgwarn.CheckRemoved5xSetting(config, "flush_interval"); err != nil {
		return Fail(err)
	}

	if stats == nil {
		stats = NewNilObserver()
	}
	return factory(info, stats, config)
}
