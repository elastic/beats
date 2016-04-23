package sqs

import (
	"encoding/json"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/defaults"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
)

type sqsOutput struct {
	Index    string
	sqs      *sqs.SQS
	queueURL *string
}

func init() {
	outputs.RegisterOutputPlugin("sqs", New)
}

func New(cfg *common.Config, _ int) (outputs.Outputer, error) {
	config := defaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}

	output := &sqsOutput{}

	if err := output.Init(&config); err != nil {
		return nil, err
	}
	return output, nil
}

func (out *sqsOutput) Init(cfg *config) error {
	out.sqs = sqs.New(session.New(
		defaults.Config().
			WithRegion(cfg.Region).
			WithMaxRetries(3),
	))

	o, err := out.sqs.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: aws.String(cfg.QueueName),
	})

	if err != nil {
		return err
	}
	out.queueURL = o.QueueUrl

	return nil
}

// Implement Outputer
func (out *sqsOutput) Close() error {
	return nil
}

func (out *sqsOutput) PublishEvent(
	trans outputs.Signaler,
	opts outputs.Options,
	event common.MapStr,
) error {

	jsonEvent, err := json.Marshal(event)
	if err != nil {
		// mark as success so event is not sent again.
		outputs.SignalCompleted(trans)

		logp.Err("Fail to json encode event(%v): %#v", err, event)
		return err
	}

	_, err = out.sqs.SendMessage(&sqs.SendMessageInput{
		QueueUrl:    out.queueURL,
		MessageBody: aws.String(string(jsonEvent)),
	})

	if err != nil {
		logp.Err("Error when writing line to sqs: %s", err)
	}

	outputs.Signal(trans, err)
	return err
}
