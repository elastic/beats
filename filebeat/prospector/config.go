package prospector

import (
	"time"

	cfg "github.com/elastic/beats/filebeat/config"
)

var (
	defaultConfig = prospectorConfig{
		ScanFrequency: 10 * time.Second,
		InputType:     cfg.DefaultInputType,
	}
)

type prospectorConfig struct {
	ScanFrequency time.Duration `config:"scan_frequency" validate:"min=0,nonzero"`
	InputType     string        `config:"input_type"`
}
