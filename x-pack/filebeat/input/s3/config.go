// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package s3

import (
	"time"

	"github.com/elastic/beats/filebeat/harvester"
)

var defaultConfig = config{
	ForwarderConfig: harvester.ForwarderConfig{
		Type: "s3",
	},
	ProfileName:       "default",
	VisibilityTimeout: 300 * time.Second,
}

type config struct {
	harvester.ForwarderConfig `config:",inline"`
	QueueURL                  string        `config:"queue_url" validate:"nonzero,required"`
	AccessKeyID               string        `config:"access_key_id"`
	SecretAccessKey           string        `config:"secret_access_key"`
	SessionToken              string        `config:"session_token"`
	ProfileName               string        `config:"credential_profile_name"`
	VisibilityTimeout         time.Duration `config:"visibility_timeout"`
}
