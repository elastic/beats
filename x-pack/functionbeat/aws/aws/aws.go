// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"github.com/elastic/beats/libbeat/feature"
	"github.com/elastic/beats/x-pack/functionbeat/function/provider"
)

// Bundle exposes the trigger supported by the AWS provider.
var Bundle = provider.MustCreate(
	"aws",
	provider.NewDefaultProvider("aws", NewCLI, NewTemplateBuilder),
	feature.NewDetails("AWS Lambda", "listen to events on AWS lambda", feature.Stable),
).MustAddFunction("cloudwatch_logs",
	NewCloudwatchLogs,
	feature.NewDetails(
		"Cloudwatch Logs trigger",
		"receive events from cloudwatch logs.",
		feature.Stable,
	),
).MustAddFunction("api_gateway_proxy",
	NewAPIGatewayProxy,
	feature.NewDetails(
		"API Gateway proxy trigger",
		"receive events from the api gateway proxy",
		feature.Experimental,
	),
).MustAddFunction("kinesis",
	NewKinesis,
	feature.NewDetails(
		"Kinesis trigger",
		"receive events from a Kinesis stream",
		feature.Stable,
	),
).MustAddFunction("sqs",
	NewSQS,
	feature.NewDetails(
		"SQS trigger",
		"receive events from a SQS queue",
		feature.Stable,
	),
).Bundle()
