package console

import (
	"github.com/elastic/beats/libbeat/outputs"
)

type Config struct {
	Pretty       bool                 `config:"pretty"`
	WriterConfig outputs.WriterConfig `config:"writer"`
}
