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
	LogGroupARN               string              `config:"log_group_arn"`
	LogGroupName              string              `config:"log_group_name"`
	RegionName                string              `config:"region_name"`
	LogStreams                []string            `config:"log_streams"`
	LogStreamPrefix           string              `config:"log_stream_prefix"`
	StartPosition             string              `config:"start_position" default:"beginning"`
	ScanFrequency             time.Duration       `config:"scan_frequency" validate:"min=0,nonzero"`
	APITimeout                time.Duration       `config:"api_timeout" validate:"min=0,nonzero"`
	APISleep                  time.Duration       `config:"api_sleep" validate:"min=0,nonzero"`
	AwsConfig                 awscommon.ConfigAWS `config:",inline"`
}

func defaultConfig() config {
	return config{
		ForwarderConfig: harvester.ForwarderConfig{
			Type: "aws-cloudwatch",
		},
		StartPosition: "beginning",
		ScanFrequency: 10 * time.Second,
		APITimeout:    120 * time.Second,
		APISleep:      200 * time.Millisecond, // FilterLogEvents has a limit of 5 transactions per second (TPS)/account/Region: 1s / 5 = 200 ms
	}
}

func (c *config) Validate() error {
	if c.StartPosition != "beginning" && c.StartPosition != "end" {
		return errors.New("start_position config parameter can only be " +
			"either 'beginning' or 'end'")
	}

	if c.LogGroupARN == "" && c.LogGroupName == "" {
		return errors.New("log_group_arn and log_group_name config parameter" +
			"cannot be both empty")
	}

	if c.LogGroupName != "" && c.RegionName == "" {
		return errors.New("region_name is required when log_group_name " +
			"config parameter is given")
	}
	return nil
}
