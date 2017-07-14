package broker

import (
	"fmt"

	"github.com/elastic/beats/libbeat/common"
)

// Global broker type registry for configuring and loading a broker instance
// via common.Config
var brokerReg = map[string]Factory{}

// RegisterType registers a new broker type.
func RegisterType(name string, f Factory) {
	if brokerReg[name] != nil {
		panic(fmt.Errorf("broker type '%v' exists already", name))
	}
	brokerReg[name] = f
}

// FindFactory retrieves a broker types constructor. Returns nil if broker type is unknown
func FindFactory(name string) Factory {
	return brokerReg[name]
}

// Load instantiates a new broker.
func Load(eventer Eventer, config common.ConfigNamespace) (Broker, error) {
	t, cfg := config.Name(), config.Config()
	if t == "" {
		t = "mem"
	}

	factory := FindFactory(t)
	if factory == nil {
		return nil, fmt.Errorf("broker type %v undefined", t)
	}
	return factory(eventer, cfg)
}
