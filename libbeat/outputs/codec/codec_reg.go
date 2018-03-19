package codec

import (
	"fmt"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

type Factory func(beat.Info, *common.Config) (Codec, error)

type Config struct {
	Namespace common.ConfigNamespace `config:",inline"`
}

var codecs = map[string]Factory{}

func RegisterType(name string, gen Factory) {
	if _, exists := codecs[name]; exists {
		panic(fmt.Sprintf("output codec '%v' already registered ", name))
	}
	codecs[name] = gen
}

func CreateEncoder(info beat.Info, cfg Config) (Codec, error) {
	// default to json codec
	codec := "json"
	if name := cfg.Namespace.Name(); name != "" {
		codec = name
	}

	factory := codecs[codec]
	if factory == nil {
		return nil, fmt.Errorf("'%v' output codec is not available", codec)
	}
	return factory(info, cfg.Namespace.Config())
}
