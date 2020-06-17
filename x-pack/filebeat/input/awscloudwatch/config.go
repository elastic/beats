// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awscloudwatch

import (
	"errors"
	"time"

	"github.com/elastic/beats/v7/filebeat/harvester"
	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
)

// TODO: add LogStreamPrefix and LogGroupPrefix config parameters
type config struct {
	harvester.ForwarderConfig `config:",inline"`
	RegionName                string              `config:"region" validate:"nonzero,required"`
	LogGroup                  string              `config:"log_group" validate:"nonzero,required"`
	LogStream                 string              `config:"log_stream"`
	StartPosition             string              `config:"start_position" default:"beginning"`
	APITimeout                time.Duration       `config:"api_timeout" validate:"min=0,nonzero"`
	Limit                     int64               `config:"limit" validate:"min=0,max=10000,nonzero"`
	WaitTime                  time.Duration       `config:"wait_time" validate:"min=0,nonzero"`
	AwsConfig                 awscommon.ConfigAWS `config:",inline"`
}

func defaultConfig() config {
	return config{
		ForwarderConfig: harvester.ForwarderConfig{
			Type: "awscloudwatch",
		},
		StartPosition: "beginning",
		APITimeout:    120 * time.Second,
		WaitTime:      1 * time.Minute,
		Limit:         10000,
	}
}

func (c *config) Validate() error {
	if c.StartPosition != "beginning" && c.StartPosition != "end" {
		return errors.New("start_position config parameter can only be either 'beginning' or 'end'")
	}
	return nil
}
