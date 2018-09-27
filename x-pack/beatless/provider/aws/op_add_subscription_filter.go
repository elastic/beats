package aws

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"

	"github.com/elastic/beats/libbeat/logp"
)

type subscriptionFilter struct {
	LogGroupName  string
	FilterName    string
	FilterPattern string
}
type opAddSubscriptionFilter struct {
	log          *logp.Logger
	subscription subscriptionFilter
	svc          *cloudwatchlogs.CloudWatchLogs
}

func (o *opAddSubscriptionFilter) Execute(ctx *executerContext) error {
	req := &cloudwatchlogs.PutSubscriptionFilterInput{
		DestinationArn: aws.String(ctx.FunctionArn),
		LogGroupName:   aws.String(o.subscription.LogGroupName),
		FilterName:     aws.String(o.subscription.FilterName),
		FilterPattern:  aws.String(o.subscription.FilterPattern),
	}

	api := o.svc.PutSubscriptionFilterRequest(req)
	resp, err := api.Send()
	if err != nil {
		o.log.Debugf("could not subscription to lambda, error: %s, response: %s", err, resp)
		return err
	}
	return nil
}

func (o *opAddSubscriptionFilter) Rollback(ctx *executerContext) error {
	return nil
}

func newOpAddSubscriptionFilter(log *logp.Logger, awsCfg aws.Config, subscription subscriptionFilter) *opAddSubscriptionFilter {
	if log == nil {
		log = logp.NewLogger("opAddSubscriptionFilter")
	}
	return &opAddSubscriptionFilter{log: log, subscription: subscription, svc: cloudwatchlogs.New(awsCfg)}
}
