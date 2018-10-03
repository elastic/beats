package aws

import (
	"fmt"
	"strings"
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

func (o *opCloudWaitCloudFormation) Execute(ctx *executorContext) error {
	o.log.Debug("waiting for cloudformation confirmation")
	status, reason, err := queryStackStatus(o.svc, o.stackName)

	for strings.Index(string(*status), "FAILED") == -1 && *status != cloudformation.StackStatusUpdateComplete && *status != cloudformation.StackStatusCreateComplete && err == nil {
		select {
		case <-time.After(periodicCheck):
			status, reason, err = queryStackStatus(o.svc, o.stackName)
		}
	}

	// Multiple status, setup a catch all for all errors.
	if strings.Index(string(*status), "FAILED") != -1 {
		return fmt.Errorf("could not create the stack, status: %s, reason: %s", *status, reason)
	}

	if err != nil {
		return err
	}

	return nil
}
