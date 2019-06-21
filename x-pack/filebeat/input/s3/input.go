// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package s3

import (
	"bytes"
	"encoding/json"
	"strings"
	"sync"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/defaults"
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
	// input name
	inputName = "s3"
	// MaxNumberOfMessage at one poll
	MaxNumberOfMessage int64 = 10
	// WaitTimeSecond for each poll
	WaitTimeSecond int64 = 20
)

type s3Info struct {
	name string
	key  string
}

func init() {
	err := input.Register(inputName, NewInput)
	if err != nil {
		panic(err)
	}
}

// Input is a input for s3
type Input struct {
	started  bool
	outlet   channel.Outleter
	config   config
	cfg      *common.Config
	registry *harvester.Registry
	logger   *logp.Logger
}

// NewInput creates a new s3 input
func NewInput(cfg *common.Config, outletFactory channel.Connector, context input.Context) (input.Input, error) {
	cfgwarn.Beta("s3 input type is used")

	logger := logp.NewLogger(inputName)

	config := defaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return nil, errors.Wrap(err, "failed unpacking config")
	}

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

	outlet, err := outletFactory(cfg, context.DynamicFields)
	if err != nil {
		return nil, err
	}

	p := &Input{
		started:  false,
		outlet:   outlet,
		cfg:      cfg,
		config:   config,
		logger:   logger,
		registry: harvester.NewRegistry(),
	}

	return p, nil
}

// Run runs the input
func (p *Input) Run() {
	p.logger.Debugf("s3", "Run s3 input with queueURLs: %+v", p.config.QueueURLs)
	if len(p.config.QueueURLs) == 0 {
		p.logger.Error("No sqs queueURLs configured")
		return
	}

	awsConfig := defaults.Config()
	awsCredentials := awssdk.Credentials{
		AccessKeyID:     p.config.AccessKeyID,
		SecretAccessKey: p.config.SecretAccessKey,
	}
	if p.config.SessionToken != "" {
		awsCredentials.SessionToken = p.config.SessionToken
	}

	awsConfig.Credentials = awssdk.StaticCredentialsProvider{
		Value: awsCredentials,
	}

	forwarder := harvester.NewForwarder(p.outlet)
	for _, queueURL := range p.config.QueueURLs {
		var d *util.Data
		regionName, err := getRegionFromQueueURL(queueURL)
		if err != nil {
			p.logger.Errorf("failed to get region name from queueURL: %s", queueURL)
			continue
		}

		awsConfig.Region = regionName
		svcSQS := sqs.New(awsConfig)
		svcS3 := s3.New(awsConfig)

		// RECEIVE
		receiveMessageInput := &sqs.ReceiveMessageInput{
			QueueUrl:              &queueURL,
			MessageAttributeNames: []string{"All"},
			MaxNumberOfMessages:   &MaxNumberOfMessage,
			VisibilityTimeout:     awssdk.Int64(20), // 20 seconds
			WaitTimeSeconds:       &WaitTimeSecond,
		}

		req := svcSQS.ReceiveMessageRequest(receiveMessageInput)
		output, errR := req.Send()
		if errR != nil {
			return
		}

		if len(output.Messages) > 0 {
			events, messagesReceiptHandles, err := p.receiveMessages(queueURL, output.Messages, svcS3, svcSQS)
			if err != nil {
				p.logger.Error(errors.Wrap(err, "receiveMessages failed"))
			}

			for _, event := range events {
				d = &util.Data{Event: *event}
				err = forwarder.Send(d)
				if err != nil {
					p.logger.Error(errors.Wrap(err, "forwarder send failed"))
				}
			}

			// TODO: When log message collection takes longer than 30s(default filebeat freq?),
			//  sqs messages got read twice or more because it didn't get deleted fast enough.
			// delete message after events are sent
			err = deleteMessages(queueURL, messagesReceiptHandles, svcSQS)
			if err != nil {
				p.logger.Error(errors.Wrap(err, "deleteMessages failed"))
			}
		}
	}
}

// Stop stops the input and all its harvesters
func (p *Input) Stop() {
	p.registry.Stop()
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

// launches goroutine per received message and wait for all message to be processed
func (p *Input) receiveMessages(queueURL string, messages []sqs.Message, svcS3 s3iface.S3API, svcSQS *sqs.SQS) ([]*beat.Event, []string, error) {
	var eventsTotal []*beat.Event
	var messagesReceiptHandles []string
	var wg sync.WaitGroup

	// TODO: Check goroutine cleanup
	numMessages := len(messages)
	wg.Add(numMessages)
	for i := range messages {
		go func(m sqs.Message) {
			// launch goroutine to handle each message
			defer wg.Done()

			s3Infos, err := handleMessage(m)
			if err != nil {
				p.logger.Error(err.Error())
			}

			if err != nil {
				p.logger.Error(err.Error())
			}

			// read from s3
			events, err := readS3Object(svcS3, s3Infos)
			if err != nil {
				p.logger.Error(err.Error())
			}

			eventsTotal = append(eventsTotal, events...)
			messagesReceiptHandles = append(messagesReceiptHandles, *m.ReceiptHandle)
		}(messages[i])
	}

	wg.Wait()
	return eventsTotal, messagesReceiptHandles, nil
}

// handle message
func handleMessage(m sqs.Message) (s3Infos []s3Info, err error) {
	msg := map[string]interface{}{}
	err = json.Unmarshal([]byte(*m.Body), &msg)
	if err != nil {
		err = errors.Wrap(err, "json unmarshal sqs message body failed")
		return
	}

	records := msg["Records"].([]interface{})
	s3Info := s3Info{}
	for _, record := range records {
		recordMap := record.(map[string]interface{})
		if recordMap["eventSource"] == "aws:s3" && recordMap["eventName"] == "ObjectCreated:Put" {
			s3Record := recordMap["s3"].(map[string]interface{})
			bucketInfo := s3Record["bucket"].(map[string]interface{})
			objectInfo := s3Record["object"].(map[string]interface{})
			s3Info.name = bucketInfo["name"].(string)
			s3Info.key = objectInfo["key"].(string)
			s3Infos = append(s3Infos, s3Info)
		}
	}
	return
}

func readS3Object(svc s3iface.S3API, s3Infos []s3Info) ([]*beat.Event, error) {
	var events []*beat.Event
	for _, s3Info := range s3Infos {
		s3GetObjectInput := &s3.GetObjectInput{
			Bucket: awssdk.String(s3Info.name),
			Key:    awssdk.String(s3Info.key),
		}
		objReq := svc.GetObjectRequest(s3GetObjectInput)

		objResp, err := objReq.Send()
		if err != nil {
			return nil, errors.Wrap(err, "s3 get object request failed")
		}

		// TODO: check way to stream
		buf := new(bytes.Buffer)
		_, err = buf.ReadFrom(objResp.Body)
		if err != nil {
			return nil, errors.Wrap(err, "buf.ReadFrom failed")
		}

		s := buf.String() // Does a complete copy of the bytes in the buffer.
		logLines := strings.Split(s, "\n")
		for _, log := range logLines {
			// create event
			event := createEvent(log, s3Info)
			events = append(events, event)
		}
	}
	return events, nil
}

func deleteMessages(queueURL string, messagesReceiptHandles []string, svcSQS *sqs.SQS) error {
	for _, receiptHandle := range messagesReceiptHandles {
		deleteMessageInput := &sqs.DeleteMessageInput{
			QueueUrl:      awssdk.String(queueURL),
			ReceiptHandle: awssdk.String(receiptHandle),
		}

		req := svcSQS.DeleteMessageRequest(deleteMessageInput)
		_, err := req.Send()
		if err != nil {
			return errors.Wrap(err, "DeleteMessageRequest failed")
		}
	}
	return nil
}

func createEvent(log string, s3Info s3Info) *beat.Event {
	f := common.MapStr{
		"message": log,
		"log": common.MapStr{
			"source": common.MapStr{
				"bucketName": s3Info.name,
				"objectKey":  s3Info.key,
			},
		},
	}
	return &beat.Event{
		Timestamp: time.Now(),
		Fields:    f,
	}
}
