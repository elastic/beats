// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"time"

	"github.com/menderesk/beats/v7/x-pack/libbeat/common/aws"

	"github.com/menderesk/beats/v7/libbeat/autodiscover/template"
)

// Config for all aws autodiscover providers.
type Config struct {
	Type      string                  `config:"type"`
	Templates template.MapperSettings `config:"templates"`

	// Period defines how often to poll the AWS API.
	Period time.Duration `config:"period" validate:"nonzero,required"`

	// AWS Specific autodiscover fields
	Regions   []string      `config:"regions"`
	AWSConfig aws.ConfigAWS `config:",inline"`
}

// DefaultConfig for all aws autodiscover providers.
func DefaultConfig() *Config {
	return &Config{
		Period: time.Minute,
	}
}
