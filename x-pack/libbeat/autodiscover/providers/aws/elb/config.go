// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package elb

import (
	"time"

	"github.com/elastic/beats/x-pack/libbeat/common/aws"

	"github.com/elastic/beats/libbeat/autodiscover/template"
	"github.com/elastic/beats/libbeat/common"
)

// Config for the aws_elb autodiscover provider.
type Config struct {
	Type string `config:"type"`

	// Standard autodiscover fields.

	// Hints are currently not supported, but may be implemented in a later release
	HintsEnabled bool                    `config:"hints.enabled"`
	Builders     []*common.Config        `config:"builders"`
	Appenders    []*common.Config        `config:"appenders"`
	Templates    template.MapperSettings `config:"templates"`

	// Period defines how often to poll the AWS API.
	Period time.Duration `config:"period" validate:"nonzero,required"`

	// AWS Specific autodiscover fields

	Regions []string `config:"regions" validate:"required"`
	aws.ConfigAWS
}

func defaultConfig() *Config {
	return &Config{
		Period: time.Minute,
	}
}
