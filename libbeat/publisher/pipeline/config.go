package pipeline

import (
	"errors"
	"fmt"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/publisher/beat"
)

// Config object for loading a pipeline instance via Load.
type Config struct {
	WaitShutdown time.Duration          `config:"wait_shutdown"`
	Broker       common.ConfigNamespace `config:"broker"`
	Output       common.ConfigNamespace `config:"output"`
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
