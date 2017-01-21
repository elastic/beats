package reader

import (
	"fmt"
	"regexp"
	"time"
)

type MultilineConfig struct {
	Negate   bool           `config:"negate"`
	Match    string         `config:"match"       validate:"required"`
	MaxLines *int           `config:"max_lines"`
	Pattern  *regexp.Regexp `config:"pattern"`
	Timeout  *time.Duration `config:"timeout"     validate:"positive"`
}

func (c *MultilineConfig) Validate() error {
	if c.Match != "after" && c.Match != "before" {
		return fmt.Errorf("unknown matcher type: %s", c.Match)
	}
	return nil
}
