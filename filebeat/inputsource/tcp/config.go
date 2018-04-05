package tcp

import (
	"fmt"
	"time"

	"github.com/dustin/go-humanize"
)

type size uint64

// Config exposes the tcp configuration.
type Config struct {
	Host           string        `config:"host"`
	LineDelimiter  string        `config:"line_delimiter" validate:"nonzero"`
	Timeout        time.Duration `config:"timeout" validate:"nonzero,positive"`
	MaxMessageSize size          `config:"max_message_size" validate:"nonzero,positive"`
}

// Validate validates the Config option for the tcp input.
func (c *Config) Validate() error {
	if len(c.Host) == 0 {
		return fmt.Errorf("need to specify the host using the `host:port` syntax")
	}
	return nil
}

func (s *size) Unpack(value string) error {
	sz, err := humanize.ParseBytes(value)
	if err != nil {
		return err
	}

	*s = size(sz)
	return nil
}
