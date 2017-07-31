package queue

import (
	"fmt"

	"github.com/elastic/beats/libbeat/common"
)

// Global queue type registry for configuring and loading a queue instance
// via common.Config
var queueReg = map[string]Factory{}

// RegisterType registers a new queue type.
func RegisterType(name string, f Factory) {
	if queueReg[name] != nil {
		panic(fmt.Errorf("queue type '%v' exists already", name))
	}
	queueReg[name] = f
}

// FindFactory retrieves a queue types constructor. Returns nil if queue type is unknown
func FindFactory(name string) Factory {
	return queueReg[name]
}

// Load instantiates a new queue.
func Load(eventer Eventer, config common.ConfigNamespace) (Queue, error) {
	t, cfg := config.Name(), config.Config()
	if t == "" {
		t = "mem"
	}

	factory := FindFactory(t)
	if factory == nil {
		return nil, fmt.Errorf("queue type %v undefined", t)
	}
	return factory(eventer, cfg)
}
