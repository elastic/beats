// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package s3

import "github.com/elastic/beats/filebeat/harvester"

var defaultConfig = config{
	ForwarderConfig: harvester.ForwarderConfig{
		Type: "s3",
	},
}

type config struct {
	harvester.ForwarderConfig `config:",inline"`
	QueueURLs                 []string `config:"queueURLs" validate:"nonzero,required"`
	BucketNames               []string `config:"bucketNames"`
	AccessKeyID               string   `config:"access_key_id"`
	SecretAccessKey           string   `config:"secret_access_key"`
	SessionToken              string   `config:"session_token"`
	SharedConfigProfile       string   `config:"shared_config_profile"`
}
