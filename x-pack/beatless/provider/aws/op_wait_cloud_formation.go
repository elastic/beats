package aws

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"

	"github.com/elastic/beats/libbeat/logp"
)

type opCloudWaitCloudFormation struct {
	log *logp.Logger
	svc *cloudformation.CloudFormation
}

func newOpWaitCloudFormation(log *logp.Logger, cfg aws.Config) *opCloudWaitCloudFormation {
	return &opCloudWaitCloudFormation{log: log, svc: cloudformation.New(cfg)}
}

func (o *opCloudWaitCloudFormation) Execute(ctx *executorContext) error {
	return nil
}
