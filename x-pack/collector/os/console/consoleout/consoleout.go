package consoleout

import (
	"bufio"
	"io"
	"os"

	"github.com/elastic/go-concert/unison"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/outputs/codec"
	"github.com/elastic/beats/v7/libbeat/outputs/codec/json"
	"github.com/elastic/beats/v7/x-pack/collector/internal/publishing"
)

type console struct {
	out   io.Writer
	codec codec.Codec
	index string
}

type consolePublisher struct {
	log    *logp.Logger
	writer *bufio.Writer
	codec  codec.Codec
	index  string
	acks   publishing.ACKCallback
}

type settings struct {
	Codec codec.Config `config:"codec"`

	// old pretty settings to use if no codec is configured
	Pretty bool `config:"pretty"`
}

func Plugin(info beat.Info) publishing.Plugin {
	return publishing.Plugin{
		Name:       "console",
		Stability:  feature.Stable,
		Deprecated: false,
		Configure: func(log *logp.Logger, cfg *common.Config) (publishing.Output, error) {
			return configure(info, log, cfg)
		},
	}
}

func configure(info beat.Info, log *logp.Logger, cfg *common.Config) (publishing.Output, error) {
	var settings settings
	if err := cfg.Unpack(&settings); err != nil {
		return nil, err
	}

	var enc codec.Codec
	if settings.Codec.Namespace.IsSet() {
		var err error
		enc, err = codec.CreateEncoder(info, settings.Codec)
		if err != nil {
			return nil, err
		}
	} else {
		enc = json.New(info.Version, json.Config{
			Pretty:     settings.Pretty,
			EscapeHTML: false,
		})
	}

	index := info.Beat

	return newConsole(index, os.Stdout, enc)
}

func newConsole(index string, out io.Writer, codec codec.Codec) (*console, error) {
	c := &console{out: out, codec: codec, index: index}
	return c, nil
}

func (c *console) Open(ctx unison.Canceler, log *logp.Logger, acks publishing.ACKCallback) (publishing.Publisher, error) {
	log.Debug("Open console output")

	writer := bufio.NewWriterSize(c.out, 8*1024)
	return &consolePublisher{
		log:    log,
		writer: writer,
		codec:  c.codec,
		index:  c.index,
		acks:   acks,
	}, nil
}

func (c *consolePublisher) Close() error {
	c.log.Debug("Closing console output")
	return nil
}

func (c *consolePublisher) Publish(mode beat.PublishMode, eventID publishing.EventID, event beat.Event) error {
	c.log.Debug("Publishing event")

	status := publishing.EventFailed
	if c.publishEvent(mode, event) {
		status = publishing.EventPublished
	}

	c.writer.Flush()
	c.acks.UpdateEventStatus(eventID, status)
	return nil
}

var nl = []byte("\n")

func (c *consolePublisher) publishEvent(mode beat.PublishMode, event beat.Event) bool {
	serializedEvent, err := c.codec.Encode(c.index, &event)
	if err != nil {
		if mode != beat.GuaranteedSend {
			return false
		}

		c.log.Errorf("Unable to encode event: %+v", err)
		c.log.Debugf("Failed event: %v", event)
		return false
	}

	if err := c.writeBuffer(serializedEvent); err != nil {
		c.log.Errorf("Unable to publish events to console: %+v", err)
		return false
	}

	if err := c.writeBuffer(nl); err != nil {
		c.log.Errorf("Error when appending newline to event: %+v", err)
		return false
	}

	return true
}

func (c *consolePublisher) writeBuffer(buf []byte) error {
	written := 0
	for written < len(buf) {
		n, err := c.writer.Write(buf[written:])
		if err != nil {
			return err
		}

		written += n
	}
	return nil
}
