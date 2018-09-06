// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"github.com/elastic/beats/libbeat/feature"
	"github.com/elastic/beats/x-pack/beatless/provider"
)

// Bundle exposes the trigger supported by the AWS provider.
var Bundle = provider.MustCreate(
	"aws",
	provider.NewDefaultProvider("aws"),
	feature.NewDetails("AWS Lambda", "listen to events on AWS lambda", feature.Experimental),
).MustAddFunction("cloudwatchlogs",
	NewCloudwatchLogs,
	feature.NewDetails(
		"Cloudwatch Logs",
		"receive events from cloudwatch logs.",
		feature.Experimental,
	)).Bundle()
