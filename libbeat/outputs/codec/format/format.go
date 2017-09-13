package format

import (
	"errors"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/fmtstr"
	"github.com/elastic/beats/libbeat/outputs/codec"
)

type Encoder struct {
	Format *fmtstr.EventFormatString
}

type Config struct {
	String *fmtstr.EventFormatString `config:"string" validate:"required"`
}

func init() {
	codec.RegisterType("format", func(_ beat.Info, cfg *common.Config) (codec.Codec, error) {
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
	return e.Format.RunBytes(event)
}
