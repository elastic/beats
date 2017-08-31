package pipeline

import (
	"errors"
	"fmt"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/processors"
)

// Config object for loading a pipeline instance via Load.
type Config struct {
	// Event processing configurations
	common.EventMetadata `config:",inline"`      // Fields and tags to add to each event.
	Processors           processors.PluginConfig `config:"processors"`

	// Event queue
	Queue common.ConfigNamespace `config:"queue"`
}

// validateClientConfig checks a ClientConfig can be used with (*Pipeline).ConnectWith.
func validateClientConfig(c *beat.ClientConfig) error {
	withDrop := false

	switch m := c.PublishMode; m {
	case beat.DefaultGuarantees, beat.GuaranteedSend:
	case beat.DropIfFull:
		withDrop = true
	default:
		return fmt.Errorf("unknown publishe mode %v", m)
	}

	fnCount := 0
	countPtr := func(b bool) {
		if b {
			fnCount++
		}
	}

	countPtr(c.ACKCount != nil)
	countPtr(c.ACKEvents != nil)
	countPtr(c.ACKLastEvent != nil)
	if fnCount > 1 {
		return fmt.Errorf("At most one of ACKCount, ACKEvents, ACKLastEvent can be configured")
	}

	// ACK handlers can not be registered DropIfFull is set, as dropping events
	// due to full broker can not be accounted for in the clients acker.
	if fnCount != 0 && withDrop {
		return errors.New("ACK handlers with DropIfFull mode not supported")
	}

	return nil
}
