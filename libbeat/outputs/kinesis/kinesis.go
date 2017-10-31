package kinesisout

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/codec"
	"github.com/elastic/beats/libbeat/outputs/codec/json"
	"github.com/elastic/beats/libbeat/publisher"
)

func init() {
	outputs.RegisterType("kinesis", makeKinesisout)
}

type Kinesis struct {
	beat   beat.Info
	stats  *outputs.Stats
	codec  codec.Codec
	stream *kinesis.Kinesis
	config
}

func makeKinesisout(beat beat.Info, stats *outputs.Stats, cfg *common.Config) (outputs.Group, error) {
	ko := &Kinesis{
		beat:   beat,
		stats:  stats,
		config: defaultConfig,
		codec:  json.New(false, beat.Version),
	}
	err := cfg.Unpack(&ko.config)
	if err != nil {
		return outputs.Fail(err)
	}

	var awsCreds *credentials.Credentials
	if ko.config.AccessKeyID != "" {
		awsCreds = credentials.NewStaticCredentials(ko.config.AccessKeyID, ko.config.SecretAccessKey, "")
	} else {
		awsCreds = credentials.NewEnvCredentials()
	}

	session := session.New(&aws.Config{
		Region: &ko.config.Region,
		DisableSSL: &ko.config.DisableSSL,
		LogLevel: &ko.config.LogLevel,
		MaxRetries: &ko.config.MaxRetries,
		Credentials: awsCreds,
	})
	ko.stream = kinesis.New(session)
	return outputs.Success(-1, 0, ko)
}

func (k *Kinesis) Close() error {
	return nil
}

func (k *Kinesis) Publish(batch publisher.Batch) error {
	events := batch.Events()
	k.stats.NewBatch(len(events))
	request := kinesis.PutRecordsInput{StreamName: aws.String(k.config.StreamName)}
	for _, event := range events {
		content, err := k.codec.Encode(k.beat.Beat, &event.Content)
		if err == nil {
			request.Records = append(
				request.Records,
				&kinesis.PutRecordsRequestEntry{
					Data:         content,
					PartitionKey: &k.config.PartitionKey,
				},
			)
		} else if event.Guaranteed() {
			logp.Critical("Failed to serialize the event: %v", err)
		} else {
			logp.Warn("Failed to serialize the event: %v", err)
		}
	}
	fmt.Printf("wrote %d records", len(request.Records))
	response, err := k.stream.PutRecords(&request)
	if err != nil {
		logp.Critical("Failed to write to Kinesis stream: %v", err)
	}
	batch.ACK()
	var failed int
	if response.FailedRecordCount == nil {
		failed = 0
	} else {
		failed = int(*response.FailedRecordCount)
	}
	k.stats.Dropped(len(events) - len(request.Records) + failed)
	k.stats.Acked(len(request.Records) - failed)
	return nil
}
