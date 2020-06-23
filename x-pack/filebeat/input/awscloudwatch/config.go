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

type config struct {
	harvester.ForwarderConfig `config:",inline"`
	LogGroupARN               string              `config:"log_group_arn" validate:"nonzero,required"`
	LogStreams                []string            `config:"log_streams"`
	LogStreamPrefix           string              `config:"log_stream_prefix"`
	StartPosition             string              `config:"start_position" default:"beginning"`
	ScanFrequency             time.Duration       `config:"scan_frequency" validate:"min=0,nonzero"`
	APITimeout                time.Duration       `config:"api_timeout" validate:"min=0,nonzero"`
	AwsConfig                 awscommon.ConfigAWS `config:",inline"`
	LogGroup                  string
	RegionName                string
}

func defaultConfig() config {
	return config{
		ForwarderConfig: harvester.ForwarderConfig{
			Type: "awscloudwatch",
		},
		StartPosition: "beginning",
		APITimeout:    120 * time.Second,
		ScanFrequency: 1 * time.Minute,
	}
}

func (c *config) Validate() error {
	if c.StartPosition != "beginning" && c.StartPosition != "end" {
		return errors.New("start_position config parameter can only be either 'beginning' or 'end'")
	}
	return nil
}
