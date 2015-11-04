package console

import (
	"encoding/json"
	"os"
	"time"

	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/logp"
	"github.com/elastic/libbeat/outputs"
)

func init() {
	outputs.RegisterOutputPlugin("console", plugin{})
}

type plugin struct{}

func (p plugin) NewOutput(
	beat string,
	config *outputs.MothershipConfig,
	topologyExpire int,
) (outputs.Outputer, error) {
	pretty := config.Pretty != nil && *config.Pretty
	return newConsole(pretty), nil
}

type console struct {
	pretty bool
}

func newConsole(pretty bool) *console {
	return &console{pretty}
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

func (c *console) PublishEvent(
	s outputs.Signaler,
	ts time.Time,
	event common.MapStr,
) error {
	var jsonEvent []byte
	var err error

	if c.pretty {
		jsonEvent, err = json.MarshalIndent(event, "", "  ")
	} else {
		jsonEvent, err = json.Marshal(event)
	}
	if err != nil {
		logp.Err("Fail to convert the event to JSON: %s", err)
		outputs.SignalCompleted(s)
		return err
	}

	if err = writeBuffer(jsonEvent); err != nil {
		goto fail
	}
	if err = writeBuffer([]byte{'\n'}); err != nil {
		goto fail
	}

	outputs.SignalCompleted(s)
	return nil
fail:
	outputs.SignalFailed(s, err)
	return err
}
