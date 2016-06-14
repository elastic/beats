package prospector

import (
	"fmt"
	"regexp"
	"time"

	cfg "github.com/elastic/beats/filebeat/config"
)

const (
	DefaultIgnoreOlder   time.Duration = 0
	DefaultScanFrequency time.Duration = 10 * time.Second
)

var (
	defaultConfig = prospectorConfig{
		IgnoreOlder:   DefaultIgnoreOlder,
		ScanFrequency: DefaultScanFrequency,
		InputType:     cfg.DefaultInputType,
	}
)

type prospectorConfig struct {
	ExcludeFiles  []*regexp.Regexp `config:"exclude_files"`
	IgnoreOlder   time.Duration    `config:"ignore_older"`
	Paths         []string         `config:"paths"`
	ScanFrequency time.Duration    `config:"scan_frequency"`
	InputType     string           `config:"input_type"`
}

func (config *prospectorConfig) Validate() error {

	if config.InputType == cfg.LogInputType && len(config.Paths) == 0 {
		return fmt.Errorf("No paths were defined for prospector")
	}
	return nil
}
