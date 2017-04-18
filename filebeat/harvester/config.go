package harvester

import (
	"fmt"
	"time"

	cfg "github.com/elastic/beats/filebeat/config"
	"github.com/elastic/beats/filebeat/harvester/reader"

	"github.com/dustin/go-humanize"
	"github.com/elastic/beats/libbeat/common/match"
)

var (
	defaultConfig = harvesterConfig{
		BufferSize:    16 * humanize.KiByte,
		InputType:     cfg.DefaultInputType,
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

type harvesterConfig struct {
	BufferSize    int                     `config:"harvester_buffer_size"`
	Encoding      string                  `config:"encoding"`
	InputType     string                  `config:"input_type"`
	Backoff       time.Duration           `config:"backoff" validate:"min=0,nonzero"`
	BackoffFactor int                     `config:"backoff_factor" validate:"min=1"`
	MaxBackoff    time.Duration           `config:"max_backoff" validate:"min=0,nonzero"`
	CloseInactive time.Duration           `config:"close_inactive"`
	CloseRemoved  bool                    `config:"close_removed"`
	CloseRenamed  bool                    `config:"close_renamed"`
	CloseEOF      bool                    `config:"close_eof"`
	CloseTimeout  time.Duration           `config:"close_timeout" validate:"min=0"`
	ExcludeLines  []match.Matcher         `config:"exclude_lines"`
	IncludeLines  []match.Matcher         `config:"include_lines"`
	MaxBytes      int                     `config:"max_bytes" validate:"min=0,nonzero"`
	Multiline     *reader.MultilineConfig `config:"multiline"`
	JSON          *reader.JSONConfig      `config:"json"`
}

func (config *harvesterConfig) Validate() error {

	// Check input type
	if _, ok := cfg.ValidInputType[config.InputType]; !ok {
		return fmt.Errorf("Invalid input type: %v", config.InputType)
	}

	if config.JSON != nil && len(config.JSON.MessageKey) == 0 &&
		config.Multiline != nil {
		return fmt.Errorf("When using the JSON decoder and multiline together, you need to specify a message_key value")
	}

	if config.JSON != nil && len(config.JSON.MessageKey) == 0 &&
		(len(config.IncludeLines) > 0 || len(config.ExcludeLines) > 0) {
		return fmt.Errorf("When using the JSON decoder and line filtering together, you need to specify a message_key value")
	}

	return nil
}
