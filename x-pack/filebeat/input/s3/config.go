// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package s3

import (
	"fmt"
	"regexp"
	"time"

	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
)

type config struct {
	QueueURL                 string              `config:"queue_url" validate:"nonzero,required"`
	VisibilityTimeout        time.Duration       `config:"visibility_timeout"`
	FipsEnabled              bool                `config:"fips_enabled"`
	AwsConfig                awscommon.ConfigAWS `config:",inline"`
	ExpandEventListFromField string              `config:"expand_event_list_from_field"`
	APITimeout               time.Duration       `config:"api_timeout"`
	FileSelectors            []FileSelectorCfg   `config:"file_selectors"`
}

// FileSelectorCfg defines type and configuration of FileSelectors
type FileSelectorCfg struct {
	RegexString              string         `config:"regex"`
	Regex                    *regexp.Regexp `config:",ignore"`
	ExpandEventListFromField string         `config:"expand_event_list_from_field"`
}

func defaultConfig() config {
	return config{
		VisibilityTimeout: 300 * time.Second,
		APITimeout:        120 * time.Second,
		FipsEnabled:       false,
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
	return nil
}
