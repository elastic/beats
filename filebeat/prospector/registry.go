package prospector

import (
	"fmt"

	"github.com/elastic/beats/filebeat/channel"
	"github.com/elastic/beats/filebeat/input/file"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

type Context struct {
	States        []file.State
	Done          chan struct{}
	BeatDone      chan struct{}
	DynamicFields *common.MapStrPointer
}

type Factory func(config *common.Config, outletFactory channel.Factory, context Context) (Prospectorer, error)

var registry = make(map[string]Factory)

func Register(name string, factory Factory) error {
	logp.Info("Registering prospector factory")
	if name == "" {
		return fmt.Errorf("Error registering prospector: name cannot be empty")
	}
	if factory == nil {
		return fmt.Errorf("Error registering prospector '%v': factory cannot be empty", name)
	}
	if _, exists := registry[name]; exists {
		return fmt.Errorf("Error registering prospector '%v': already registered", name)
	}

	registry[name] = factory
	logp.Info("Successfully registered prospector")

	return nil
}

func GetFactory(name string) (Factory, error) {
	if _, exists := registry[name]; !exists {
		return nil, fmt.Errorf("Error creating prospector. No such prospector type exist: '%v'", name)
	}
	return registry[name], nil
}
