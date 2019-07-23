// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package s3

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
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
	awscommon "github.com/elastic/beats/x-pack/libbeat/common/aws"
)

const inputName = "s3"

var (
	// The maximum number of messages to return. Amazon SQS never returns more messages
	// than this value (however, fewer messages might be returned).
	maxNumberOfMessage int64 = 10

	// The duration (in seconds) for which the call waits for a message to arrive
	// in the queue before returning. If a message is available, the call returns
	// sooner than WaitTimeSeconds. If no messages are available and the wait time
	// expires, the call returns successfully with an empty list of messages.
	waitTimeSecond int64 = 10
)

func init() {
	err := input.Register(inputName, NewInput)
	if err != nil {
		panic(err)
	}
}

// Input is a input for s3
type Input struct {
	outlet     channel.Outleter // Output of received s3 logs.
	config     config
	awsConfig  awssdk.Config
	logger     *logp.Logger
	close      chan struct{}
	workerOnce sync.Once // Guarantees that the worker goroutine is only started once.
	context    *channelContext
}

type s3Info struct {
	name   string
	key    string
	region string
	arn    string
}

type bucket struct {
	Name string `json:"name"`
	Arn  string `json:"arn"`
}

type object struct {
	Key string `json:"key"`
}

type s3BucketOjbect struct {
	bucket `json:"bucket"`
	object `json:"object"`
}

type s3Context struct {
	mux  sync.Mutex
	refs int
	err  error // first error witnessed or multi error
	errC chan error
}

type sqsMessage struct {
	Records []struct {
		EventSource string         `json:"eventSource"`
		AwsRegion   string         `json:"awsRegion"`
		EventName   string         `json:"eventName"`
		S3          s3BucketOjbect `json:"s3"`
	} `json:"Records"`
}

// channelContext implements context.Context by wrapping a channel
type channelContext struct {
	done <-chan struct{}
}

func (r *channelContext) Deadline() (time.Time, bool) { return time.Time{}, false }
func (r *channelContext) Done() <-chan struct{}       { return r.done }
func (r *channelContext) Err() error {
	select {
	case <-r.done:
		return context.Canceled
	default:
		return nil
	}
}
func (r *channelContext) Value(key interface{}) interface{} { return nil }

// NewInput creates a new s3 input
func NewInput(cfg *common.Config, outletFactory channel.Connector, context input.Context) (input.Input, error) {
	cfgwarn.Beta("s3 input type is used")
	logger := logp.NewLogger(inputName)

	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, errors.Wrap(err, "failed unpacking config")
	}

	outlet, err := outletFactory(cfg, context.DynamicFields)
	if err != nil {
		return nil, err
	}

	awsConfig, err := awscommon.GetAWSCredentials(config.AwsConfig)
	if err != nil {
		return nil, errors.Wrap(err, "getAWSCredentials failed")
	}

	closeChannel := make(chan struct{})
	p := &Input{
		outlet:    outlet,
		config:    config,
		awsConfig: awsConfig,
		logger:    logger,
		close:     closeChannel,
		context:   &channelContext{closeChannel},
	}
	return p, nil
}

// Run runs the input
func (p *Input) Run() {
	p.workerOnce.Do(func() {
		visibilityTimeout := int64(p.config.VisibilityTimeout.Seconds())
		regionName, err := getRegionFromQueueURL(p.config.QueueURL)
		if err != nil {
			p.logger.Errorf("failed to get region name from queueURL: %v", p.config.QueueURL)
		}

		awsConfig := p.awsConfig.Copy()
		awsConfig.Region = regionName
		svcSQS := sqs.New(awsConfig)
		svcS3 := s3.New(awsConfig)

		go p.run(svcSQS, svcS3, visibilityTimeout)
	})
}

func (p *Input) run(svcSQS *sqs.Client, svcS3 *s3.Client, visibilityTimeout int64) {
	defer p.logger.Infof("S3 input worker for '%v' has stopped.", p.config.QueueURL)
	p.logger.Infof("s3 input worker has started. with queueURL: %v", p.config.QueueURL)
	for {
		select {
		case <-p.close:
			return
		default:
		}
		// receive messages from sqs
		req := svcSQS.ReceiveMessageRequest(
			&sqs.ReceiveMessageInput{
				QueueUrl:              &p.config.QueueURL,
				MessageAttributeNames: []string{"All"},
				MaxNumberOfMessages:   &maxNumberOfMessage,
				VisibilityTimeout:     &visibilityTimeout,
				WaitTimeSeconds:       &waitTimeSecond,
			})

		output, err := req.Send(p.context)
		if err != nil {
			p.logger.Error("failed to receive message from SQS:", err)
			time.Sleep(time.Duration(waitTimeSecond) * time.Second)
			continue
		}

		if output == nil || len(output.Messages) == 0 {
			p.logger.Debug("no message received from SQS:", p.config.QueueURL)
			continue
		}

		// process messages received from sqs, get logs from s3 and create events
		p.processor(p.config.QueueURL, output.Messages, visibilityTimeout, svcS3, svcSQS)
	}
}

// Stop stops the s3 input
func (p *Input) Stop() {
	defer p.outlet.Close()
	close(p.close)
	p.logger.Info("Stopping s3 input")
}

// Wait stops the s3 input.
func (p *Input) Wait() {
	p.Stop()
}

func (p *Input) processor(queueURL string, messages []sqs.Message, visibilityTimeout int64, svcS3 *s3.Client, svcSQS *sqs.Client) {
	var wg sync.WaitGroup
	numMessages := len(messages)
	wg.Add(numMessages)

	// process messages received from sqs
	for i := range messages {
		errC := make(chan error)
		go p.processMessage(svcS3, messages[i], &wg, errC)
		go p.processorKeepAlive(svcSQS, messages[i], queueURL, visibilityTimeout, errC)
	}
	wg.Wait()
}

func (p *Input) processMessage(svcS3 *s3.Client, message sqs.Message, wg *sync.WaitGroup, errC chan error) {
	defer wg.Done()

	s3Infos, err := handleSQSMessage(message)
	if err != nil {
		p.logger.Error(errors.Wrap(err, "handelMessage failed"))
		return
	}

	// read from s3 object and create event for each log line
	p.handleS3Objects(svcS3, s3Infos, errC)
}

func (p *Input) processorKeepAlive(svcSQS *sqs.Client, message sqs.Message, queueURL string, visibilityTimeout int64, errC chan error) {
	for {
		select {
		case <-p.close:
			return
		case err := <-errC:
			if err != nil {
				p.logger.Warnf("Processing message failed: %v", err)
				err := p.changeVisibilityTimeout(queueURL, visibilityTimeout, svcSQS, message.ReceiptHandle)
				if err != nil {
					p.logger.Error(errors.Wrap(err, "change message visibility failed"))
				}
				p.logger.Warnf("Message visibility timeout updated to %v", visibilityTimeout)
			} else {
				err := p.deleteMessage(queueURL, *message.ReceiptHandle, svcSQS)
				if err != nil {
					p.logger.Error(errors.Wrap(err, "deleteMessages failed"))
				}
			}
			return
		case <-time.After(time.Duration(visibilityTimeout/2) * time.Second):
			// If half of the set visibilityTimeout passed and this is
			// still ongoing, then change visibility timeout.
			err := p.changeVisibilityTimeout(queueURL, visibilityTimeout, svcSQS, message.ReceiptHandle)
			if err != nil {
				p.logger.Error(errors.Wrap(err, "change message visibility failed"))
			}
			p.logger.Infof("Message visibility timeout updated to %v", visibilityTimeout)
		}
	}
}

func (p *Input) changeVisibilityTimeout(queueURL string, visibilityTimeout int64, svc *sqs.Client, receiptHandle *string) error {
	req := svc.ChangeMessageVisibilityRequest(&sqs.ChangeMessageVisibilityInput{
		QueueUrl:          &queueURL,
		VisibilityTimeout: &visibilityTimeout,
		ReceiptHandle:     receiptHandle,
	})
	_, err := req.Send(p.context)
	return err
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
func handleSQSMessage(m sqs.Message) ([]s3Info, error) {
	msg := sqsMessage{}
	err := json.Unmarshal([]byte(*m.Body), &msg)
	if err != nil {
		return nil, errors.Wrap(err, "json unmarshal sqs message body failed")
	}

	var s3Infos []s3Info
	for _, record := range msg.Records {
		if record.EventSource == "aws:s3" && record.EventName == "ObjectCreated:Put" {
			s3Infos = append(s3Infos, s3Info{
				region: record.AwsRegion,
				name:   record.S3.bucket.Name,
				key:    record.S3.object.Key,
				arn:    record.S3.bucket.Arn,
			})
		}
	}
	return s3Infos, nil
}

func (c *s3Context) Done() {
	c.mux.Lock()
	defer c.mux.Unlock()
	c.done()
}

func (c *s3Context) Fail(err error) {
	c.mux.Lock()
	defer c.mux.Unlock()

	// only care about the last error for now
	// TODO: add "Typed" error to error for context
	c.err = err
	c.done()
}

func (c *s3Context) done() {
	c.refs--
	if c.refs == 0 {
		c.errC <- c.err
		close(c.errC)
	}
}

func (p *Input) handleS3Objects(svc s3iface.ClientAPI, s3Infos []s3Info, errC chan error) {
	if len(s3Infos) > 0 {
		var wg sync.WaitGroup
		wg.Add(len(s3Infos))

		s3Context := &s3Context{
			refs: len(s3Infos),
			errC: errC,
		}

		for i := range s3Infos {
			go func(s3Info s3Info) {
				defer wg.Done()
				defer s3Context.Done()
				objectHash := s3ObjectHash(s3Info)

				// read from s3 object
				reader, err := p.bufferedIORead(svc, s3Info)
				if err != nil {
					s3Context.Fail(errors.Wrap(err, "bufferedIORead failed"))
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
							// create event for last line
							offset += len([]byte(log))
							err = p.forwardEvent(createEvent(log, offset, s3Info, objectHash))
							if err != nil {
								s3Context.Fail(errors.Wrapf(err, "forwardEvent failed for %v", s3Info.key))
							}
							return
						}

						s3Context.Fail(errors.Wrapf(err, "ReadString failed for %v", s3Info.key))
						return
					}

					// create event per log line
					offset += len([]byte(log))
					err = p.forwardEvent(createEvent(log, offset, s3Info, objectHash))
					if err != nil {
						s3Context.Fail(errors.Wrapf(err, "forwardEvent failed for %v", s3Info.key))
						return
					}
				}
			}(s3Infos[i])
		}
		wg.Wait()
	}
}

func (p *Input) bufferedIORead(svc s3iface.ClientAPI, s3Info s3Info) (*bufio.Reader, error) {
	s3GetObjectInput := &s3.GetObjectInput{
		Bucket: awssdk.String(s3Info.name),
		Key:    awssdk.String(s3Info.key),
	}
	req := svc.GetObjectRequest(s3GetObjectInput)

	resp, err := req.Send(p.context)
	if err != nil {
		return nil, errors.Wrapf(err, "s3 get object request failed %v", s3Info.key)
	}

	return bufio.NewReader(resp.Body), nil
}

func (p *Input) forwardEvent(event *beat.Event) error {
	forwarder := harvester.NewForwarder(p.outlet)
	d := &util.Data{Event: *event}
	err := forwarder.Send(d)
	if err != nil {
		return errors.Wrap(err, "forwarder send failed")
	}
	return nil
}

func (p *Input) deleteMessage(queueURL string, messagesReceiptHandle string, svcSQS *sqs.Client) error {
	deleteMessageInput := &sqs.DeleteMessageInput{
		QueueUrl:      awssdk.String(queueURL),
		ReceiptHandle: awssdk.String(messagesReceiptHandle),
	}

	req := svcSQS.DeleteMessageRequest(deleteMessageInput)
	_, err := req.Send(p.context)
	if err != nil {
		return errors.Wrap(err, "DeleteMessageRequest failed")
	}
	return nil
}

func createEvent(log string, offset int, s3Info s3Info, objectHash string) *beat.Event {
	f := common.MapStr{
		"message": log,
		"log": common.MapStr{
			"offset":    int64(offset),
			"file.path": constructObjectURL(s3Info),
		},
		"aws": common.MapStr{
			"s3": common.MapStr{
				"bucket": common.MapStr{
					"name": s3Info.name,
					"arn":  s3Info.arn},
				"object.key": s3Info.key,
			},
		},
		"cloud": common.MapStr{
			"provider": "aws",
			"region":   s3Info.region,
		},
	}
	return &beat.Event{
		Meta:      common.MapStr{"id": objectHash + "-" + fmt.Sprintf("%012d", offset)},
		Timestamp: time.Now(),
		Fields:    f,
	}
}

func constructObjectURL(info s3Info) string {
	return "https://" + info.name + ".s3-" + info.region + ".amazonaws.com/" + info.key
}

// s3ObjectHash returns a short sha256 hash of the bucket arn + object key name.
func s3ObjectHash(s3Info s3Info) string {
	h := sha256.New()
	h.Write([]byte(s3Info.arn + s3Info.key))
	prefix := hex.EncodeToString(h.Sum(nil))
	return prefix[:10]
}
