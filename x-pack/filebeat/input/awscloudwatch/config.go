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
	LogGroupNamePrefix        string              `config:"log_group_name_prefix"`
	RegionName                string              `config:"region_name"`
	LogStreams                []*string            `config:"log_streams"`
	LogStreamPrefix           string              `config:"log_stream_prefix"`
	StartPosition             string              `config:"start_position" default:"beginning"`
	ScanFrequency             time.Duration       `config:"scan_frequency" validate:"min=0,nonzero"`
	APITimeout                time.Duration       `config:"api_timeout" validate:"min=0,nonzero"`
	APISleep                  time.Duration       `config:"api_sleep" validate:"min=0,nonzero"`
	Latency                   time.Duration       `config:"latency"`
	NumberOfWorkers           int                 `config:"number_of_workers"`
	AWSConfig                 awscommon.ConfigAWS `config:",inline"`
}

func defaultConfig() config {
	return config{
		ForwarderConfig: harvester.ForwarderConfig{
			Type: "aws-cloudwatch",
		},
		StartPosition:   "beginning",
		ScanFrequency:   10 * time.Second,
		APITimeout:      120 * time.Second,
		APISleep:        200 * time.Millisecond, // FilterLogEvents has a limit of 5 transactions per second (TPS)/account/Region: 1s / 5 = 200 ms
		NumberOfWorkers: 1,
	}
}

func (c *config) Validate() error {
	if c.StartPosition != "beginning" && c.StartPosition != "end" {
		return errors.New("start_position config parameter can only be " +
			"either 'beginning' or 'end'")
	}

	if c.LogGroupARN == "" && c.LogGroupName == "" && c.LogGroupNamePrefix == "" {
		return errors.New("log_group_arn, log_group_name and log_group_name_prefix config parameter" +
			"cannot all be empty")
	}

	if c.LogGroupName != "" && c.LogGroupNamePrefix != "" {
		return errors.New("log_group_name and log_group_name_prefix cannot be given at the same time")
	}

	if (c.LogGroupName != "" || c.LogGroupNamePrefix != "") && c.RegionName == "" {
		return errors.New("region_name is required when log_group_name or log_group_name_prefix " +
			"config parameter is given")
	}
	return nil
}
