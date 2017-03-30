package prospector

import (
	"fmt"
	"time"

	cfg "github.com/elastic/beats/filebeat/config"
	"github.com/elastic/beats/filebeat/harvester/reader"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/match"
	"github.com/elastic/beats/libbeat/processors"
)

var (
	defaultConfig = prospectorConfig{
		Enabled:        true,
		DocumentType:   "log",
		IgnoreOlder:    0,
		ScanFrequency:  10 * time.Second,
		InputType:      cfg.DefaultInputType,
		CleanInactive:  0,
		CleanRemoved:   true,
		HarvesterLimit: 0,
		Symlinks:       false,
		TailFiles:      false,
	}
)

type prospectorConfig struct {
	common.EventMetadata `config:",inline"`      // Fields and tags to add to events.
	Enabled              bool                    `config:"enabled"`
	DocumentType         string                  `config:"document_type"`
	ExcludeFiles         []match.Matcher         `config:"exclude_files"`
	IgnoreOlder          time.Duration           `config:"ignore_older"`
	Paths                []string                `config:"paths"`
	ScanFrequency        time.Duration           `config:"scan_frequency" validate:"min=0,nonzero"`
	InputType            string                  `config:"input_type"`
	CleanInactive        time.Duration           `config:"clean_inactive" validate:"min=0"`
	CleanRemoved         bool                    `config:"clean_removed"`
	HarvesterLimit       uint64                  `config:"harvester_limit" validate:"min=0"`
	Symlinks             bool                    `config:"symlinks"`
	TailFiles            bool                    `config:"tail_files"`
	JSON                 *reader.JSONConfig      `config:"json"`
	Pipeline             string                  `config:"pipeline"`
	Module               string                  `config:"_module_name"`  // hidden option to set the module name
	Fileset              string                  `config:"_fileset_name"` // hidden option to set the fileset name
	Processors           processors.PluginConfig `config:"processors"`
}

func (config *prospectorConfig) Validate() error {

	if config.InputType == cfg.LogInputType && len(config.Paths) == 0 {
		return fmt.Errorf("No paths were defined for prospector")
	}

	if config.CleanInactive != 0 && config.IgnoreOlder == 0 {
		return fmt.Errorf("ignore_older must be enabled when clean_inactive is used")
	}

	if config.CleanInactive != 0 && config.CleanInactive <= config.IgnoreOlder+config.ScanFrequency {
		return fmt.Errorf("clean_inactive must be > ignore_older + scan_frequency to make sure only files which are not monitored anymore are removed")
	}

	return nil
}
