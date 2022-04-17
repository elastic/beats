// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package include

import (
	"github.com/menderesk/beats/v7/libbeat/feature"
	"github.com/menderesk/beats/v7/x-pack/functionbeat/function/provider"
	"github.com/menderesk/beats/v7/x-pack/functionbeat/provider/aws/aws"
)

// Bundle exposes the trigger supported by the AWS provider.
var bundle = provider.MustCreate(
	"aws",
	provider.NewDefaultProvider("aws", provider.NewNullCli, provider.NewNullTemplateBuilder),
	feature.MakeDetails("AWS Lambda", "listen to events on AWS lambda", feature.Stable),
).MustAddFunction("cloudwatch_logs",
	aws.NewCloudwatchLogs,
	aws.CloudwatchLogsDetails(),
).MustAddFunction("api_gateway_proxy",
	aws.NewAPIGatewayProxy,
	aws.APIGatewayProxyDetails(),
).MustAddFunction("kinesis",
	aws.NewKinesis,
	aws.KinesisDetails(),
).MustAddFunction("sqs",
	aws.NewSQS,
	aws.SQSDetails(),
).MustAddFunction("cloudwatch_logs_kinesis",
	aws.NewCloudwatchKinesis,
	aws.CloudwatchKinesisDetails(),
).Bundle()

func init() {
	feature.MustRegisterBundle(bundle)
}
