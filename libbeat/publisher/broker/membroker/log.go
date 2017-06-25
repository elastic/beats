package membroker

import (
	"github.com/elastic/beats/libbeat/logp"
)

type logger interface {
	Debug(...interface{})
	Debugf(string, ...interface{})
}

var defaultLogger logger = logp.NewLogger("membroker")
