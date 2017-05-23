package json

import (
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

	buffer, err := common.JSONEncode(event, e.Pretty)
	if err != nil {
		logp.Err("Fail to convert the event to JSON (%v): %#v", err, event)
		return nil, err
	}

	return buffer, nil
}
