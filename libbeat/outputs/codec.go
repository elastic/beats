package outputs

import (
	"fmt"

	"github.com/elastic/beats/libbeat/common"
)

type Codec interface {
	Encode(Event common.MapStr) ([]byte, error)
}

type CodecConfig struct {
	Namespace common.ConfigNamespace `config:",inline"`
}

type CodecFactory func(*common.Config) (Codec, error)

var outputCodecs = map[string]CodecFactory{}

func RegisterOutputCodec(name string, gen CodecFactory) {
	if _, exists := outputCodecs[name]; exists {
		panic(fmt.Sprintf("output codec '%v' already registered ", name))
	}
	outputCodecs[name] = gen
}

func CreateEncoder(cfg CodecConfig) (Codec, error) {
	// default to json codec
	codec := "json"
	if name := cfg.Namespace.Name(); name != "" {
		codec = name
	}

	factory := outputCodecs[codec]
	if factory == nil {
		return nil, fmt.Errorf("'%v' output codec is not available", codec)
	}
	return factory(cfg.Namespace.Config())
}
