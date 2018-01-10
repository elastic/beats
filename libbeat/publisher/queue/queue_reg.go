package queue

import (
	"fmt"

	"github.com/elastic/beats/libbeat/common"
)

const defaultQueueType = "mem"

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
	t := config.Name()
	if t == "" {
		t = defaultQueueType
	}

	queueConfig := config.Config()
	if queueConfig == nil {
		queueConfig = common.NewConfig()
	}

	factory := FindFactory(t)
	if factory == nil {
		return nil, fmt.Errorf("queue type %v undefined", t)
	}
	q, err := factory(eventer, queueConfig)
	if err != nil {
		return nil, err
	}

	// Return a wrapped queue that can do QoS.
	return NewQueueWrapper(q, queueConfig), nil
}
