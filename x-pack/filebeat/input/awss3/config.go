// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"fmt"
	"regexp"
	"time"

	"github.com/dustin/go-humanize"

	"github.com/elastic/beats/v7/libbeat/reader/multiline"
	"github.com/elastic/beats/v7/libbeat/reader/readfile"
	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
)

type config struct {
	APITimeout               time.Duration           `config:"api_timeout"`
	ExpandEventListFromField string                  `config:"expand_event_list_from_field"`
	FileSelectors            []FileSelectorCfg       `config:"file_selectors"`
	FipsEnabled              bool                    `config:"fips_enabled"`
	MaxNumberOfMessages      int                     `config:"max_number_of_messages"`
	QueueURL                 string                  `config:"queue_url" validate:"nonzero,required"`
	VisibilityTimeout        time.Duration           `config:"visibility_timeout"`
	AwsConfig                awscommon.ConfigAWS     `config:",inline"`
	MaxBytes                 int                     `config:"max_bytes" validate:"min=0,nonzero"`
	Multiline                *multiline.Config       `config:"multiline"`
	LineTerminator           readfile.LineTerminator `config:"line_terminator"`
	Encoding                 string                  `config:"encoding"`
	BufferSize               int                     `config:"buffer_size"`
}

// FileSelectorCfg defines type and configuration of FileSelectors
type FileSelectorCfg struct {
	RegexString              string                  `config:"regex"`
	Regex                    *regexp.Regexp          `config:",ignore"`
	ExpandEventListFromField string                  `config:"expand_event_list_from_field"`
	MaxBytes                 int                     `config:"max_bytes" validate:"min=0,nonzero"`
	Multiline                *multiline.Config       `config:"multiline"`
	LineTerminator           readfile.LineTerminator `config:"line_terminator"`
	Encoding                 string                  `config:"encoding"`
	BufferSize               int                     `config:"buffer_size"`
}

func defaultConfig() config {
	return config{
		APITimeout:          120 * time.Second,
		FipsEnabled:         false,
		MaxNumberOfMessages: 5,
		VisibilityTimeout:   300 * time.Second,
		LineTerminator:      readfile.AutoLineTerminator,
		MaxBytes:            10 * humanize.MiByte,
		BufferSize:          16 * humanize.KiByte,
	}
}

func (c *config) Validate() error {
	if c.VisibilityTimeout < 0 || c.VisibilityTimeout.Hours() > 12 {
		return fmt.Errorf("visibility timeout %v is not within the "+
			"required range 0s to 12h", c.VisibilityTimeout)
	}

	if c.APITimeout < 0 || c.APITimeout > c.VisibilityTimeout/2 {
		return fmt.Errorf("api timeout %v needs to be larger than"+
			" 0s and smaller than half of the visibility timeout", c.APITimeout)
	}

	for i := range c.FileSelectors {
		r, err := regexp.Compile(c.FileSelectors[i].RegexString)
		if err != nil {
			return err
		}
		c.FileSelectors[i].Regex = r
	}

	if c.MaxNumberOfMessages > 10 || c.MaxNumberOfMessages < 1 {
		return fmt.Errorf(" max_number_of_messages %v needs to be between 1 and 10", c.MaxNumberOfMessages)
	}
	return nil
}
