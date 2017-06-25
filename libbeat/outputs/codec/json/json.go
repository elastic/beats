package json

import (
	"bytes"
	stdjson "encoding/json"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/outputs/codec"
	"github.com/elastic/beats/libbeat/publisher/beat"
	"github.com/urso/go-structform/gotype"
	"github.com/urso/go-structform/json"
)

type Encoder struct {
	buf    bytes.Buffer
	folder *gotype.Iterator
	pretty bool
}

type config struct {
	Pretty bool
}

var defaultConfig = config{
	Pretty: false,
}

func init() {
	codec.RegisterType("json", func(cfg *common.Config) (codec.Codec, error) {
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
	e := &Encoder{pretty: pretty}
	e.reset()
	return e
}

func (e *Encoder) reset() {
	visitor := json.NewVisitor(&e.buf)

	var err error

	// create new encoder with custom time.Time encoding
	e.folder, err = gotype.NewIterator(visitor,
		gotype.Folders(codec.TimestampEncoder, codec.BcTimestampEncoder),
	)
	if err != nil {
		panic(err)
	}
}

func (e *Encoder) Encode(index string, event *beat.Event) ([]byte, error) {
	e.buf.Reset()
	err := e.folder.Fold(makeEvent(index, event))
	if err != nil {
		e.reset()
		return nil, err
	}

	json := e.buf.Bytes()
	if !e.pretty {
		return json, nil
	}

	var buf bytes.Buffer
	if err = stdjson.Indent(&buf, json, "", "  "); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
