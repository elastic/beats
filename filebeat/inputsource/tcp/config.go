package tcp

import (
	"fmt"
	"time"

	"github.com/elastic/beats/libbeat/common/cfgtype"
	"github.com/elastic/beats/libbeat/common/transport/tlscommon"
)

// Name is the human readable name and identifier.
const Name = "tcp"

type size uint64

// Config exposes the tcp configuration.
type Config struct {
	Host           string                  `config:"host"`
	LineDelimiter  string                  `config:"line_delimiter" validate:"nonzero"`
	Timeout        time.Duration           `config:"timeout" validate:"nonzero,positive"`
	MaxMessageSize cfgtype.ByteSize        `config:"max_message_size" validate:"nonzero,positive"`
	TLS            *tlscommon.ServerConfig `config:"ssl"`
}

// Validate validates the Config option for the tcp input.
func (c *Config) Validate() error {
	if len(c.Host) == 0 {
		return fmt.Errorf("need to specify the host using the `host:port` syntax")
	}
	return nil
}
