// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package s3

import (
	"time"

	"github.com/elastic/beats/filebeat/harvester"
	awscommon "github.com/elastic/beats/x-pack/libbeat/common/aws"
)

type config struct {
	harvester.ForwarderConfig `config:",inline"`
	QueueURL                  string        `config:"queue_url" validate:"nonzero,required"`
	VisibilityTimeout         time.Duration `config:"visibility_timeout"`
	AwsConfig                 awscommon.ConfigAWS
}

func defaultConfig() config {
	return config{
		ForwarderConfig: harvester.ForwarderConfig{
			Type: "s3",
		},
		VisibilityTimeout: 300 * time.Second,
	}
}
