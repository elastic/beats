// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awscloudwatch

import (
	"time"

	"github.com/elastic/beats/v7/filebeat/harvester"
	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
)

type config struct {
	harvester.ForwarderConfig `config:",inline"`
	LogGroup                  string              `config:"log_group" validate:"nonzero,required"`
	LogStream                 string              `config:"log_stream" validate:"nonzero,required"`
	RegionName                string              `config:"region" validate:"nonzero,required"`
	Limit                     int                 `config:"limit"`
	APITimeout                time.Duration       `config:"api_timeout"`
	ScanFrequency             time.Duration       `config:"scan_frequency" validate:"min=0,nonzero"`
	AwsConfig                 awscommon.ConfigAWS `config:",inline"`
}

func defaultConfig() config {
	return config{
		ForwarderConfig: harvester.ForwarderConfig{
			Type: "awscloudwatch",
		},
		Limit:         100,
		APITimeout:    120 * time.Second,
		ScanFrequency: 10 * time.Second,
	}
}

func (c *config) Validate() error {
	return nil
}
