package json

import (
	"encoding/json"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
)

type Encoder struct {
	Pretty bool
}

type Config struct {
	Pretty bool
}

var defaultConfig = Config{
	Pretty: false,
}

func init() {
	outputs.RegisterOutputCodec("json", func(cfg *common.Config) (outputs.Codec, error) {
		config := defaultConfig
		if cfg != nil {
			if err := cfg.Unpack(&config); err != nil {
				return nil, err
			}
		}

		return New(config.Pretty), nil
	})
}

func New(pretty bool) *Encoder {
	return &Encoder{pretty}
}

func (e *Encoder) Encode(event common.MapStr) ([]byte, error) {
	var err error
	var serializedEvent []byte

	if e.Pretty {
		serializedEvent, err = json.MarshalIndent(event, "", "  ")
	} else {
		serializedEvent, err = json.Marshal(event)
	}
	if err != nil {
		logp.Err("Fail to convert the event to JSON (%v): %#v", err, event)
	}

	return serializedEvent, err
}
