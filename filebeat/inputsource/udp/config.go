package udp

import (
	"time"

	"github.com/elastic/beats/libbeat/common/cfgtype"
)

// Config options for the UDPServer
type Config struct {
	Host           string           `config:"host"`
	MaxMessageSize cfgtype.ByteSize `config:"max_message_size" validate:"positive,nonzero"`
	Timeout        time.Duration    `config:"timeout"`
}
