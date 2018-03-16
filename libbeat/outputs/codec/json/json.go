package json

import (
	"bytes"
	stdjson "encoding/json"

	"github.com/elastic/go-structform/gotype"
	"github.com/elastic/go-structform/json"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/outputs/codec"
)

// Encoder for serializing a beat.Event to json.
type Encoder struct {
	buf     bytes.Buffer
	folder  *gotype.Iterator
	pretty  bool
	version string
}

type config struct {
	Pretty bool
}

var defaultConfig = config{
	Pretty: false,
}

func init() {
	codec.RegisterType("json", func(info beat.Info, cfg *common.Config) (codec.Codec, error) {
		config := defaultConfig
		if cfg != nil {
			if err := cfg.Unpack(&config); err != nil {
				return nil, err
			}
		}

		return New(config.Pretty, info.Version), nil
	})
}

// New creates a new json Encoder.
func New(pretty bool, version string) *Encoder {
	e := &Encoder{pretty: pretty, version: version}
	e.reset()
	return e
}

func (e *Encoder) reset() {
	visitor := json.NewVisitor(&e.buf)

	var err error

	// create new encoder with custom time.Time encoding
	e.folder, err = gotype.NewIterator(visitor,
		gotype.Folders(
			codec.MakeTimestampEncoder(),
			codec.MakeBCTimestampEncoder(),
		),
	)
	if err != nil {
		panic(err)
	}
}

// Encode serializies a beat event to JSON. It adds additional metadata in the
// `@metadata` namespace.
func (e *Encoder) Encode(index string, event *beat.Event) ([]byte, error) {
	e.buf.Reset()
	err := e.folder.Fold(makeEvent(index, e.version, event))
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
