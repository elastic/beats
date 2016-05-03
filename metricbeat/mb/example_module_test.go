package mb_test

import (
	"github.com/elastic/beats/metricbeat/mb"
)

func init() {
	// Register the ModuleFactory function for the "example" module.
	if err := mb.Registry.AddModule("example", NewModule); err != nil {
		panic(err)
	}
}

type Module struct {
	mb.BaseModule
	Protocol string
}

func NewModule(base mb.BaseModule) (mb.Module, error) {
	// Unpack additional configuration options.
	config := struct {
		Protocol string `config:"protocol"`
	}{
		Protocol: "udp",
	}
	if err := base.UnpackConfig(&config); err != nil {
		return nil, err
	}

	return &Module{BaseModule: base, Protocol: config.Protocol}, nil
}

// ExampleModuleFactory demonstrates how to register a custom ModuleFactory
// and unpack additional configuration data.
func ExampleModuleFactory() {}
