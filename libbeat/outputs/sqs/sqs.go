package sqs

import (
	"encoding/json"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/op"
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
	sqsConfig := defaultConfig
	if err := cfg.Unpack(&sqsConfig); err != nil {
		return nil, err
	}

	awsSqs := sqs.New(awsSession(sqsConfig))

	o, err := awsSqs.GetQueueUrl(&sqs.GetQueueUrlInput{QueueName: aws.String(sqsConfig.QueueName)})
	if err != nil {
		return nil, err
	}

	output := &sqsOutput{
		sqs:      awsSqs,
		queueURL: o.QueueUrl,
	}

	return output, nil
}

func awsSession(cfg config) *session.Session {
	sess := aws.NewConfig().
		WithRegion(cfg.Region).
		WithMaxRetries(5).
		WithCredentialsChainVerboseErrors(true)

	prods := make([]credentials.Provider, 0)
	if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
		prods = append(prods, &credentials.StaticProvider{
			Value: credentials.Value{
				AccessKeyID:     cfg.AccessKeyID,
				SecretAccessKey: cfg.SecretAccessKey,
				ProviderName:    credentials.StaticProviderName,
			}})
	}
	prods = append(prods, &credentials.EnvProvider{})
	prods = append(prods, &ec2rolecreds.EC2RoleProvider{
		Client: ec2metadata.New(session.New(sess), &aws.Config{Endpoint: aws.String("http://169.254.169.254/latest")}),
	})

	creds := credentials.NewChainCredentials(prods)
	sess.WithCredentials(creds)
	return session.New(sess)

}

// Implement Outputer
func (out *sqsOutput) Close() error {
	return nil
}

func (out *sqsOutput) PublishEvent(
	s op.Signaler,
	opts outputs.Options,
	event common.MapStr,
) error {

	jsonEvent, err := json.Marshal(event)
	if err != nil {
		op.SigCompleted(s)
		logp.Err("Fail to json encode event(%v): %#v", err, event)
		return err
	}

	_, err = out.sqs.SendMessage(&sqs.SendMessageInput{
		QueueUrl:    out.queueURL,
		MessageBody: aws.String(string(jsonEvent)),
	})

	if err != nil {
		logp.Critical("Error when writing line to sqs: %s", err)
		op.SigFailed(s, err)
		return err
	}

	op.SigCompleted(s)
	return err
}
