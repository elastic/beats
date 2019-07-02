// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package s3

import (
	"bufio"
	"encoding/json"
	"io"
	"strings"
	"sync"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/defaults"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/s3iface"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/pkg/errors"

	"github.com/elastic/beats/filebeat/channel"
	"github.com/elastic/beats/filebeat/harvester"
	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/filebeat/util"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"
)

var (
	// Filebeat input name
	inputName = "s3"

	// The maximum number of messages to return. Amazon SQS never returns more messages
	// than this value (however, fewer messages might be returned).
	maxNumberOfMessage int64 = 10

	// The duration (in seconds) for which the call waits for a message to arrive
	// in the queue before returning. If a message is available, the call returns
	// sooner than WaitTimeSeconds. If no messages are available and the wait time
	// expires, the call returns successfully with an empty list of messages.
	waitTimeSecond int64 = 10

	// The duration (in seconds) that the received messages are hidden from subsequent
	// retrieve requests after being retrieved by a ReceiveMessage request.
	// This value needs to be a lot bigger than filebeat collection frequency so
	// if it took too long to read the s3 log, this sqs message will not be reprocessed.
	// The default visibility timeout for a message is 30 seconds. The minimum
	// is 0 seconds. The maximum is 12 hours.
	visibilityTimeout int64 = 300
)

func init() {
	err := input.Register(inputName, NewInput)
	if err != nil {
		panic(err)
	}
}

// Input is a input for s3
type Input struct {
	outlet    channel.Outleter
	config    config
	awsConfig awssdk.Config
	logger    *logp.Logger
}

type s3Info struct {
	name   string
	key    string
	region string
}

// NewInput creates a new s3 input
func NewInput(cfg *common.Config, outletFactory channel.Connector, context input.Context) (input.Input, error) {
	cfgwarn.Beta("s3 input type is used")

	logger := logp.NewLogger(inputName)

	config := defaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return nil, errors.Wrap(err, "failed unpacking config")
	}

	outlet, err := outletFactory(cfg, context.DynamicFields)
	if err != nil {
		return nil, err
	}

	if len(config.QueueURLs) == 0 {
		return nil, errors.Wrap(err, "no sqs queueURLs configured")
	}

	awsConfig, err := getAWSCredentials(config)
	if err != nil {
		return nil, errors.Wrap(err, "getAWSCredentials failed")
	}

	p := &Input{
		outlet:    outlet,
		config:    config,
		awsConfig: awsConfig,
		logger:    logger,
	}

	return p, nil
}

func getAWSCredentials(config config) (awssdk.Config, error) {
	// Check if accessKeyID and secretAccessKey is given from configuration
	if config.AccessKeyID != "" && config.SecretAccessKey != "" {
		awsConfig := defaults.Config()
		awsCredentials := awssdk.Credentials{
			AccessKeyID:     config.AccessKeyID,
			SecretAccessKey: config.SecretAccessKey,
		}
		if config.SessionToken != "" {
			awsCredentials.SessionToken = config.SessionToken
		}

		awsConfig.Credentials = awssdk.StaticCredentialsProvider{
			Value: awsCredentials,
		}
		return awsConfig, nil
	}

	// If accessKeyID and secretAccessKey is not given, then load from default config
	// Please see https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-profiles.html
	// with more details.
	if config.ProfileName != "" {
		return external.LoadDefaultAWSConfig(
			external.WithSharedConfigProfile(config.ProfileName),
		)
	}
	return external.LoadDefaultAWSConfig()
}

// Run runs the input
func (p *Input) Run() {
	p.logger.Debugf("s3", "Run s3 input with queueURLs: %+v", p.config.QueueURLs)
	for _, queueURL := range p.config.QueueURLs {
		regionName, err := getRegionFromQueueURL(queueURL)
		if err != nil {
			p.logger.Errorf("failed to get region name from queueURL: %s", queueURL)
			continue
		}

		awsConfig := p.awsConfig.Copy()
		awsConfig.Region = regionName
		svcSQS := sqs.New(awsConfig)
		svcS3 := s3.New(awsConfig)

		// update message visibility timeout if it's taking longer than 1/2 of
		// visibilityTimeout to make sure filebeat can finish reading
		changeMessageVisibilityInput := &sqs.ChangeMessageVisibilityInput{
			QueueUrl:          &queueURL,
			VisibilityTimeout: &visibilityTimeout,
		}

		// receive messages
		req := svcSQS.ReceiveMessageRequest(
			&sqs.ReceiveMessageInput{
				QueueUrl:              &queueURL,
				MessageAttributeNames: []string{"All"},
				MaxNumberOfMessages:   &maxNumberOfMessage,
				VisibilityTimeout:     &visibilityTimeout,
				WaitTimeSeconds:       &waitTimeSecond,
			})
		output, err := req.Send()
		if err != nil {
			p.logger.Error("failed to receive message from SQS:", err)
			continue
		}

		// process messages received from sqs
		if len(output.Messages) > 0 {
			var wg sync.WaitGroup
			numMessages := len(output.Messages)
			wg.Add(numMessages)

			for i := range output.Messages {
				done := make(chan struct{})
				// launch goroutine to handle each message from sqs
				go func(message sqs.Message) {
					defer wg.Done()
					defer close(done)

					s3Infos, err := handleMessage(message, p.config.BucketNames)
					if err != nil {
						p.logger.Error(err.Error())
					}

					// read from s3 object and create event for each log line
					p.readS3Object(svcS3, s3Infos)

					// delete message after events are sent
					err = deleteMessage(queueURL, *message.ReceiptHandle, svcSQS)
					if err != nil {
						p.logger.Error(errors.Wrap(err, "deleteMessages failed"))
					}
				}(output.Messages[i])

				select {
				case <-done:
				case <-time.After(time.Duration(visibilityTimeout/2) * time.Second):
					// if half of the set visibilityTimeout passed and this is
					// still ongoing, then change visibility timeout.
					changeMessageVisibilityInput.ReceiptHandle = output.Messages[i].ReceiptHandle
					req := svcSQS.ChangeMessageVisibilityRequest(changeMessageVisibilityInput)
					_, err = req.Send()
					if err != nil {
						p.logger.Error(errors.Wrap(err, "change message visibility failed"))
					}
				}
			}
			wg.Wait()
		}
	}
}

// Stop stops the input and all its harvesters
func (p *Input) Stop() {
	p.outlet.Close()
}

// Wait stops the s3 input.
func (p *Input) Wait() {
	p.Stop()
}

func getRegionFromQueueURL(queueURL string) (string, error) {
	// get region from queueURL
	// Example: https://sqs.us-east-1.amazonaws.com/627959692251/test-s3-logs
	queueURLSplit := strings.Split(queueURL, ".")
	if queueURLSplit[0] == "https://sqs" && queueURLSplit[2] == "amazonaws" {
		return queueURLSplit[1], nil
	}
	return "", errors.New("queueURL is not in format: https://sqs.{REGION_ENDPOINT}.amazonaws.com/{ACCOUNT_NUMBER}/{QUEUE_NAME}")
}

// handle message
func handleMessage(m sqs.Message, bucketNames []string) ([]s3Info, error) {
	msg := map[string]interface{}{}
	err := json.Unmarshal([]byte(*m.Body), &msg)
	if err != nil {
		return nil, errors.Wrap(err, "json unmarshal sqs message body failed")
	}

	var s3Infos []s3Info
	records := msg["Records"].([]interface{})
	for _, record := range records {
		recordMap := record.(map[string]interface{})
		if recordMap["eventSource"] == "aws:s3" && recordMap["eventName"] == "ObjectCreated:Put" {
			s3Info := s3Info{}
			if !stringInSlice(recordMap["awsRegion"].(string), bucketNames) {
				continue
			}

			s3Info.region = recordMap["awsRegion"].(string)
			s3Record := recordMap["s3"].(map[string]interface{})

			bucketInfo := s3Record["bucket"].(map[string]interface{})
			s3Info.name = bucketInfo["name"].(string)

			objectInfo := s3Record["object"].(map[string]interface{})
			s3Info.key = objectInfo["key"].(string)
			s3Infos = append(s3Infos, s3Info)
		}
	}
	return s3Infos, nil
}

// stringInSlice checks if a string is already exists in list
// If there is no bucketNames configured, then collect all.
func stringInSlice(name string, bucketNames []string) bool {
	if bucketNames == nil || len(bucketNames) == 0 {
		return true
	}

	for _, v := range bucketNames {
		if v == name {
			return true
		}
	}
	return false
}

func (p *Input) readS3Object(svc s3iface.S3API, s3Infos []s3Info) {
	if len(s3Infos) > 0 {
		var wg sync.WaitGroup
		numS3Infos := len(s3Infos)
		wg.Add(numS3Infos)

		for i := range s3Infos {
			go func(s3Info s3Info) {
				// launch goroutine to handle each message
				defer wg.Done()

				// read from s3 object
				reader, err := bufferedIORead(svc, s3Info)
				if err != nil {
					p.logger.Error(errors.Wrap(err, "s3 get object request failed"))
					return
				}

				offset := 0
				for {
					log, err := reader.ReadString('\n')
					if log == "" {
						break
					}

					if err != nil {
						if err == io.EOF {
							offset += len([]byte(log))
							p.forwardEvent(createEvent(log, offset, s3Info))
							break
						} else {
							p.logger.Error(errors.Wrap(err, "ReadString failed"))
						}
					}

					// create event per log line
					offset += len([]byte(log))
					p.forwardEvent(createEvent(log, offset, s3Info))
				}
			}(s3Infos[i])
		}
		wg.Wait()
	}
}

func bufferedIORead(svc s3iface.S3API, s3Info s3Info) (*bufio.Reader, error) {
	s3GetObjectInput := &s3.GetObjectInput{
		Bucket: awssdk.String(s3Info.name),
		Key:    awssdk.String(s3Info.key),
	}
	req := svc.GetObjectRequest(s3GetObjectInput)

	resp, err := req.Send()
	if err != nil {
		return nil, errors.Wrap(err, "s3 get object request failed")
	}

	return bufio.NewReader(resp.Body), nil
}

func (p *Input) forwardEvent(event *beat.Event) {
	forwarder := harvester.NewForwarder(p.outlet)
	d := &util.Data{Event: *event}
	err := forwarder.Send(d)
	if err != nil {
		p.logger.Error(errors.Wrap(err, "forwarder send failed"))
	}
}

func deleteMessage(queueURL string, messagesReceiptHandle string, svcSQS *sqs.SQS) error {
	deleteMessageInput := &sqs.DeleteMessageInput{
		QueueUrl:      awssdk.String(queueURL),
		ReceiptHandle: awssdk.String(messagesReceiptHandle),
	}

	req := svcSQS.DeleteMessageRequest(deleteMessageInput)
	_, err := req.Send()
	if err != nil {
		return errors.Wrap(err, "DeleteMessageRequest failed")
	}
	return nil
}

func createEvent(log string, offset int, s3Info s3Info) *beat.Event {
	f := common.MapStr{
		"message": log,
		"log": common.MapStr{
			"offset":    int64(offset),
			"file.path": constructObjectURL(s3Info),
		},
		"aws": common.MapStr{
			"s3": common.MapStr{
				"bucket_name": s3Info.name,
				"object_key":  s3Info.key,
			},
		},
		"cloud": common.MapStr{
			"provider": "aws",
			"region":   s3Info.region,
		},
	}
	return &beat.Event{
		Timestamp: time.Now(),
		Fields:    f,
	}
}

func constructObjectURL(info s3Info) string {
	return "https://" + info.name + ".s3-" + info.region + ".amazonaws.com/" + info.key
}
