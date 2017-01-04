package format

import (
	"errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/fmtstr"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
)

type Encoder struct {
	Format *fmtstr.EventFormatString
}

type Config struct {
	String *fmtstr.EventFormatString `config:"string" validate:"required"`
}

func init() {
	outputs.RegisterOutputCodec("format", func(cfg *common.Config) (outputs.Codec, error) {
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

func (w *Encoder) Encode(event common.MapStr) ([]byte, error) {
	serializedEvent, err := w.Format.RunBytes(event)
	if err != nil {
		logp.Err("Fail to format event (%v): %#v", err, event)
	}
	return serializedEvent, err
}
