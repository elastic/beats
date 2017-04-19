package console

import (
	"fmt"
	"os"
	"runtime"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/op"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/codecs/json"
)

func init() {
	outputs.RegisterOutputPlugin("console", New)
}

type console struct {
	out   *os.File
	codec outputs.Codec
}

func New(_ common.BeatInfo, config *common.Config) (outputs.Outputer, error) {
	var unpackedConfig Config
	err := config.Unpack(&unpackedConfig)
	if err != nil {
		return nil, err
	}

	var codec outputs.Codec
	if unpackedConfig.Codec.Namespace.IsSet() {
		codec, err = outputs.CreateEncoder(unpackedConfig.Codec)
		if err != nil {
			return nil, err
		}
	} else {
		codec = json.New(unpackedConfig.Pretty)
	}

	c, err := newConsole(codec)
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

func newConsole(codec outputs.Codec) (*console, error) {
	return &console{codec: codec, out: os.Stdout}, nil
}

// Implement Outputer
func (c *console) Close() error {
	return nil
}

var nl = []byte{'\n'}

func (c *console) PublishEvent(
	s op.Signaler,
	opts outputs.Options,
	data outputs.Data,
) error {
	serializedEvent, err := c.codec.Encode(data.Event)
	if err = c.writeBuffer(serializedEvent); err != nil {
		goto fail
	}
	if err = c.writeBuffer(nl); err != nil {
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
