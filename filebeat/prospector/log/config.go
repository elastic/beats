package log

import (
	"fmt"
	"time"

	"github.com/dustin/go-humanize"
	cfg "github.com/elastic/beats/filebeat/config"
	"github.com/elastic/beats/filebeat/harvester/reader"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/match"
	"github.com/elastic/beats/libbeat/processors"
)

var (
	defaultConfig = config{
		// Common
		InputType:     cfg.DefaultInputType,
		CleanInactive: 0,

		// Prospector
		Enabled:        true,
		IgnoreOlder:    0,
		ScanFrequency:  10 * time.Second,
		CleanRemoved:   true,
		HarvesterLimit: 0,
		Symlinks:       false,
		TailFiles:      false,

		// Harvester
		BufferSize:    16 * humanize.KiByte,
		Backoff:       1 * time.Second,
		BackoffFactor: 2,
		MaxBackoff:    10 * time.Second,
		CloseInactive: 5 * time.Minute,
		MaxBytes:      10 * humanize.MiByte,
		CloseRemoved:  true,
		CloseRenamed:  false,
		CloseEOF:      false,
		CloseTimeout:  0,
	}
)

type config struct {

	// Common
	InputType     string        `config:"input_type"`
	CleanInactive time.Duration `config:"clean_inactive" validate:"min=0"`

	// Prospector
	Enabled        bool            `config:"enabled"`
	ExcludeFiles   []match.Matcher `config:"exclude_files"`
	IgnoreOlder    time.Duration   `config:"ignore_older"`
	Paths          []string        `config:"paths"`
	ScanFrequency  time.Duration   `config:"scan_frequency" validate:"min=0,nonzero"`
	CleanRemoved   bool            `config:"clean_removed"`
	HarvesterLimit uint64          `config:"harvester_limit" validate:"min=0"`
	Symlinks       bool            `config:"symlinks"`
	TailFiles      bool            `config:"tail_files"`
	recursiveGlob  bool            `config:"recursive_glob.enabled"`

	// Harvester
	common.EventMetadata `config:",inline"`      // Fields and tags to add to events.
	BufferSize           int                     `config:"harvester_buffer_size"`
	Encoding             string                  `config:"encoding"`
	Backoff              time.Duration           `config:"backoff" validate:"min=0,nonzero"`
	BackoffFactor        int                     `config:"backoff_factor" validate:"min=1"`
	MaxBackoff           time.Duration           `config:"max_backoff" validate:"min=0,nonzero"`
	CloseInactive        time.Duration           `config:"close_inactive"`
	CloseRemoved         bool                    `config:"close_removed"`
	CloseRenamed         bool                    `config:"close_renamed"`
	CloseEOF             bool                    `config:"close_eof"`
	CloseTimeout         time.Duration           `config:"close_timeout" validate:"min=0"`
	ExcludeLines         []match.Matcher         `config:"exclude_lines"`
	IncludeLines         []match.Matcher         `config:"include_lines"`
	MaxBytes             int                     `config:"max_bytes" validate:"min=0,nonzero"`
	Multiline            *reader.MultilineConfig `config:"multiline"`
	JSON                 *reader.JSONConfig      `config:"json"`
	Pipeline             string                  `config:"pipeline"`
	Module               string                  `config:"_module_name"`  // hidden option to set the module name
	Fileset              string                  `config:"_fileset_name"` // hidden option to set the fileset name
	Processors           processors.PluginConfig `config:"processors"`
}

func (c *config) Validate() error {

	// Prospector
	if c.InputType == cfg.LogInputType && len(c.Paths) == 0 {
		return fmt.Errorf("No paths were defined for prospector")
	}

	if c.CleanInactive != 0 && c.IgnoreOlder == 0 {
		return fmt.Errorf("ignore_older must be enabled when clean_inactive is used")
	}

	if c.CleanInactive != 0 && c.CleanInactive <= c.IgnoreOlder+c.ScanFrequency {
		return fmt.Errorf("clean_inactive must be > ignore_older + scan_frequency to make sure only files which are not monitored anymore are removed")
	}

	// Harvester
	// Check input type
	if _, ok := cfg.ValidInputType[c.InputType]; !ok {
		return fmt.Errorf("Invalid input type: %v", c.InputType)
	}

	if c.JSON != nil && len(c.JSON.MessageKey) == 0 &&
		c.Multiline != nil {
		return fmt.Errorf("When using the JSON decoder and multiline together, you need to specify a message_key value")
	}

	if c.JSON != nil && len(c.JSON.MessageKey) == 0 &&
		(len(c.IncludeLines) > 0 || len(c.ExcludeLines) > 0) {
		return fmt.Errorf("When using the JSON decoder and line filtering together, you need to specify a message_key value")
	}

	return nil
}
