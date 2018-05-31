package multiline

import (
	"fmt"
	"time"

	"github.com/elastic/beats/libbeat/common/match"
)

type Config struct {
	Negate       bool           `config:"negate"`
	Match        string         `config:"match" validate:"required"`
	MaxLines     *int           `config:"max_lines"`
	Pattern      *match.Matcher `config:"pattern" validate:"required"`
	Timeout      *time.Duration `config:"timeout" validate:"positive"`
	FlushPattern *match.Matcher `config:"flush_pattern"`
}

func (c *Config) Validate() error {
	if c.Match != "after" && c.Match != "before" {
		return fmt.Errorf("unknown matcher type: %s", c.Match)
	}
	return nil
}
