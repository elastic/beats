package console

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/op"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
)

func init() {
	outputs.RegisterOutputPlugin("console", New)
}

type console struct {
	config config
	out    *os.File
}

func New(_ string, config *common.Config, _ int) (outputs.Outputer, error) {
	c := &console{config: defaultConfig, out: os.Stdout}
	err := config.Unpack(&c.config)
	if err != nil {
		return nil, err
	}

	// check stdout actually being available
	if _, err = c.out.Stat(); err != nil {
		return nil, fmt.Errorf("console output initialization failed with: %v", err)
	}

	return c, nil
}

func newConsole(pretty bool, format string) *console {
	return &console{config: config{Pretty: pretty, Format: format}, out: os.Stdout}
}

func writeBuffer(buf []byte) error {
	written := 0
	for written < len(buf) {
		n, err := os.Stdout.Write(buf[written:])
		if err != nil {
			return err
		}

		written += n
	}
	return nil
}

// Implement Outputer
func (c *console) Close() error {
	return nil
}

func (c *console) PublishEvent(
	s op.Signaler,
	opts outputs.Options,
	event common.MapStr,
) error {
	var serializedEvent []byte
	var err error

	if c.config.Format != "" {
		serializedEvent, err = outputs.FormatEvent(event, c.config.Format)
		if err != nil {
			logp.Err("Failed to apply format %s on event (%#v) due to: %v", c.config.Format, event, err)
		}
	} else {
		if c.config.Pretty {
			serializedEvent, err = json.MarshalIndent(event, "", "  ")
		} else {
			serializedEvent, err = json.Marshal(event)
		}
		if err != nil {
			logp.Err("Fail to convert the event to JSON (%v): %#v", err, event)
			op.SigCompleted(s)
			return err
		}
	}

	if err = c.writeBuffer(serializedEvent); err != nil {
		goto fail
	}
	if err = c.writeBuffer([]byte{'\n'}); err != nil {
		goto fail
	}

	op.SigCompleted(s)
	return nil
fail:
	if opts.Guaranteed {
		logp.Critical("Unable to publish events to console: %v", err)
	}
	op.SigFailed(s, err)
	return err
}

func (c *console) writeBuffer(buf []byte) error {
	written := 0
	for written < len(buf) {
		n, err := c.out.Write(buf[written:])
		if err != nil {
			return err
		}

		written += n
	}
	return nil
}
