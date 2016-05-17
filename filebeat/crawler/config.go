package crawler

import (
	"regexp"
	"time"

	"github.com/elastic/beats/filebeat/harvester"
)

const (
	DefaultIgnoreOlder   time.Duration = 0
	DefaultScanFrequency time.Duration = 10 * time.Second
)

var (
	defaultConfig = prospectorConfig{
		IgnoreOlder:   DefaultIgnoreOlder,
		ScanFrequency: DefaultScanFrequency,
	}
)

type prospectorConfig struct {
	ExcludeFiles  []*regexp.Regexp          `config:"exclude_files"`
	Harvester     harvester.HarvesterConfig `config:",inline"`
	IgnoreOlder   time.Duration             `config:"ignore_older"`
	Paths         []string                  `config:"paths"`
	ScanFrequency time.Duration             `config:"scan_frequency"`
}

func (p *prospectorConfig) Validate() error { return nil }
