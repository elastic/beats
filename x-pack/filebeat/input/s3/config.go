// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package s3

import (
	"fmt"
	"time"

	"github.com/elastic/beats/filebeat/harvester"
	awscommon "github.com/elastic/beats/x-pack/libbeat/common/aws"
)

type config struct {
	harvester.ForwarderConfig `config:",inline"`
	QueueURL                  string              `config:"queue_url" validate:"nonzero,required"`
	VisibilityTimeout         time.Duration       `config:"visibility_timeout"`
	AwsConfig                 awscommon.ConfigAWS `config:",inline"`
	ExpandEventListFromField  string              `config:"expand_event_list_from_field"`
	AwsApiTimeout             time.Duration       `config:"aws_api_timeout"`
}

func defaultConfig() config {
	return config{
		ForwarderConfig: harvester.ForwarderConfig{
			Type: "s3",
		},
		VisibilityTimeout: 300 * time.Second,
		AwsApiTimeout:     120 * time.Second,
	}
}

func (c *config) Validate() error {
	if c.VisibilityTimeout < 0 || c.VisibilityTimeout.Hours() > 12 {
		return fmt.Errorf("visibility timeout %v is not within the "+
			"required range 0s to 12h", c.VisibilityTimeout)
	}
	if c.AwsApiTimeout < 0 || c.AwsApiTimeout > c.VisibilityTimeout/2 {
		return fmt.Errorf("aws api timeout %v needs to be larger than"+
			" 0s and smaller than half of the visibility timeout", c.AwsApiTimeout)
	}
	return nil
}
