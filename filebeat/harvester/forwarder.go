package harvester

import (
	"errors"

	"github.com/elastic/beats/filebeat/util"
	"github.com/elastic/beats/libbeat/logp"
)

// Outlet interface is used for forwarding events
type Outlet interface {
	OnEvent(data *util.Data) bool
}

// Forwarder contains shared options between all harvesters needed to forward events
type Forwarder struct {
	Outlet Outlet
}

// ForwarderConfig contains all config options shared by all harvesters
type ForwarderConfig struct {
	Type string `config:"type"`
}

// NewForwarder creates a new forwarder instances and initialises processors if configured
func NewForwarder(outlet Outlet) *Forwarder {
	return &Forwarder{Outlet: outlet}
}

// Send updates the input state and sends the event to the spooler
// All state updates done by the input itself are synchronous to make sure no states are overwritten
func (f *Forwarder) Send(data *util.Data) error {
	ok := f.Outlet.OnEvent(data)
	if !ok {
		logp.Info("Input outlet closed")
		return errors.New("input outlet closed")
	}

	return nil
}
