// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !agentbeat

package include

import (
	// register autodiscover providers
	_ "github.com/elastic/beats/v7/x-pack/libbeat/autodiscover/providers/aws/ec2"
	_ "github.com/elastic/beats/v7/x-pack/libbeat/autodiscover/providers/aws/elb"
	_ "github.com/elastic/beats/v7/x-pack/libbeat/autodiscover/providers/nomad"
)
