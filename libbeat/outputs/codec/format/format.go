package format

import (
	"errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/fmtstr"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs/codec"
	"github.com/elastic/beats/libbeat/publisher/beat"
)

type Encoder struct {
	Format *fmtstr.EventFormatString
}

type Config struct {
	String *fmtstr.EventFormatString `config:"string" validate:"required"`
}

func init() {
	codec.RegisterType("format", func(cfg *common.Config) (codec.Codec, error) {
		config := Config{}
		if cfg == nil {
			return nil, errors.New("empty format codec configuration")
		}

		if err := cfg.Unpack(&config); err != nil {
			return nil, err
		}

		return New(config.String), nil
	})
}

func New(fmt *fmtstr.EventFormatString) *Encoder {
	return &Encoder{fmt}
}

func (e *Encoder) Encode(_ string, event *beat.Event) ([]byte, error) {
	serializedEvent, err := e.Format.RunBytes(event)
	if err != nil {
		logp.Err("Fail to format event (%v): %#v", err, event)
	}
	return serializedEvent, err
}
