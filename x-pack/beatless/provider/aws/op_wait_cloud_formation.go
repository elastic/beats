package aws

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"

	"github.com/elastic/beats/libbeat/logp"
)

var periodicCheck = 10 * time.Second

type opCloudWaitCloudFormation struct {
	log       *logp.Logger
	svc       *cloudformation.CloudFormation
	stackName string
}

func newOpWaitCloudFormation(
	log *logp.Logger,
	cfg aws.Config,
	stackName string,
) *opCloudWaitCloudFormation {
	return &opCloudWaitCloudFormation{
		log:       log,
		svc:       cloudformation.New(cfg),
		stackName: stackName,
	}
}

func (o *opCloudWaitCloudFormation) query() (*cloudformation.StackStatus, string, error) {
	input := &cloudformation.DescribeStacksInput{StackName: aws.String(o.stackName)}
	req := o.svc.DescribeStacksRequest(input)
	resp, err := req.Send()
	if err != nil {
		return nil, "", err
	}

	if len(resp.Stacks) == 0 {
		return nil, "", fmt.Errorf("no stack found with the name %s", o.stackName)
	}

	stack := resp.Stacks[0]
	return &stack.StackStatus, "", nil
}

func (o *opCloudWaitCloudFormation) Execute(ctx *executorContext) error {
	o.log.Debug("waiting for cloudformation confirmation")
	status, reason, err := o.query()
	for *status == cloudformation.StackStatusCreateInProgress && err == nil {
		select {
		case <-time.After(periodicCheck):
			status, reason, err = o.query()
		}
	}

	if *status != cloudformation.StackStatusCreateComplete {
		return fmt.Errorf("could not create the stack, status: %s, reason: %s", *status, reason)
	}
	return nil
}
