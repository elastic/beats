package console

import (
	"fmt"
	"os"
	"runtime"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/op"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
)

func init() {
	outputs.RegisterOutputPlugin("console", New)
}

type console struct {
	config Config
	out    *os.File
	writer outputs.Writer
}

func New(_ string, config *common.Config, _ int) (outputs.Outputer, error) {
	var unpackedConfig Config
	err := config.Unpack(&unpackedConfig)
	if err != nil {
		return nil, err
	}
	c, err := newConsole(unpackedConfig)
	if err != nil {
		return nil, fmt.Errorf("console output initialization failed with: %v", err)
	}
	// check stdout actually being available
	if runtime.GOOS != "windows" {
		if _, err = c.out.Stat(); err != nil {
			return nil, fmt.Errorf("console output initialization failed with: %v", err)
		}
	}

	return c, nil
}

func newConsole(config Config) (*console, error) {

	writer := outputs.CreateWriter(config.WriterConfig)

	return &console{config: config, writer: writer, out: os.Stdout}, nil
}

// Implement Outputer
func (c *console) Close() error {
	return nil
}

func (c *console) PublishEvent(
	s op.Signaler,
	opts outputs.Options,
	data outputs.Data,
) error {
	defer op.SigCompleted(s)

	serializedEvent, err := c.writer.Write(data.Event)

	if err = c.writeBuffer(serializedEvent); err != nil {
		goto fail
	}
	if err = c.writeBuffer([]byte{'\n'}); err != nil {
		goto fail
	}

	op.SigCompleted(s)
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
