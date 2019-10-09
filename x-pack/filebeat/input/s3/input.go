// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package s3

import (
	"bufio"
	"compress/gzip"
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
	"github.com/aws/aws-sdk-go-v2/aws/awserr"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/s3iface"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/sqsiface"
	"github.com/pkg/errors"

	"github.com/elastic/beats/filebeat/channel"
	"github.com/elastic/beats/filebeat/input"
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

	errOutletClosed = errors.New("input outlet closed")
)

func init() {
	err := input.Register(inputName, NewInput)
	if err != nil {
		panic(err)
	}
}

// Input is a input for s3
type s3Input struct {
	outlet     channel.Outleter // Output of received s3 logs.
	config     config
	awsConfig  awssdk.Config
	logger     *logp.Logger
	close      chan struct{}
	workerOnce sync.Once // Guarantees that the worker goroutine is only started once.
	context    *channelContext
	workerWg   sync.WaitGroup // Waits on s3 worker goroutine.
	stopOnce   sync.Once
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

type sqsMessage struct {
	Records []struct {
		EventSource string         `json:"eventSource"`
		AwsRegion   string         `json:"awsRegion"`
		EventName   string         `json:"eventName"`
		S3          s3BucketOjbect `json:"s3"`
	} `json:"Records"`
}

type s3Context struct {
	mux  sync.Mutex
	refs int
	err  error // first error witnessed or multi error
	errC chan error
}

// channelContext implements context.Context by wrapping a channel
type channelContext struct {
	done <-chan struct{}
}

func (c *channelContext) Deadline() (time.Time, bool) { return time.Time{}, false }
func (c *channelContext) Done() <-chan struct{}       { return c.done }
func (c *channelContext) Err() error {
	select {
	case <-c.done:
		return context.Canceled
	default:
		return nil
	}
}
func (c *channelContext) Value(key interface{}) interface{} { return nil }

// NewInput creates a new s3 input
func NewInput(cfg *common.Config, connector channel.Connector, context input.Context) (input.Input, error) {
	cfgwarn.Beta("s3 input type is used")
	logger := logp.NewLogger(inputName)

	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, errors.Wrap(err, "failed unpacking config")
	}

	out, err := connector.ConnectWith(cfg, beat.ClientConfig{
		Processing: beat.ProcessingConfig{
			DynamicFields: context.DynamicFields,
		},
		ACKEvents: func(privates []interface{}) {
			for _, private := range privates {
				if s3Context, ok := private.(*s3Context); ok {
					s3Context.done()
				}
			}
		},
	})
	if err != nil {
		return nil, err
	}

	awsConfig, err := awscommon.GetAWSCredentials(config.AwsConfig)
	if err != nil {
		return nil, errors.Wrap(err, "getAWSCredentials failed")
	}

	closeChannel := make(chan struct{})
	p := &s3Input{
		outlet:    out,
		config:    config,
		awsConfig: awsConfig,
		logger:    logger,
		close:     closeChannel,
		context:   &channelContext{closeChannel},
	}
	return p, nil
}

// Run runs the input
func (p *s3Input) Run() {
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

		p.workerWg.Add(1)
		go p.run(svcSQS, svcS3, visibilityTimeout)
		p.workerWg.Done()
	})
}

func (p *s3Input) run(svcSQS sqsiface.ClientAPI, svcS3 s3iface.ClientAPI, visibilityTimeout int64) {
	defer p.logger.Infof("s3 input worker for '%v' has stopped.", p.config.QueueURL)

	p.logger.Infof("s3 input worker has started. with queueURL: %v", p.config.QueueURL)
	for p.context.Err() == nil {
		// receive messages from sqs
		output, err := p.receiveMessage(svcSQS, visibilityTimeout)
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == awssdk.ErrCodeRequestCanceled {
				continue
			}
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
func (p *s3Input) Stop() {
	p.stopOnce.Do(func() {
		defer p.outlet.Close()
		close(p.close)
		p.context.Done()
		p.logger.Info("Stopping s3 input")
	})
}

// Wait stops the s3 input.
func (p *s3Input) Wait() {
	p.Stop()
	p.workerWg.Wait()
}

func (p *s3Input) processor(queueURL string, messages []sqs.Message, visibilityTimeout int64, svcS3 s3iface.ClientAPI, svcSQS sqsiface.ClientAPI) {
	var wg sync.WaitGroup
	numMessages := len(messages)
	wg.Add(numMessages * 2)

	// process messages received from sqs
	for i := range messages {
		errC := make(chan error)
		go p.processMessage(svcS3, messages[i], &wg, errC)
		go p.processorKeepAlive(svcSQS, messages[i], queueURL, visibilityTimeout, &wg, errC)
	}
	wg.Wait()
}

func (p *s3Input) processMessage(svcS3 s3iface.ClientAPI, message sqs.Message, wg *sync.WaitGroup, errC chan error) {
	defer wg.Done()

	s3Infos, err := handleSQSMessage(message)
	if err != nil {
		p.logger.Error(errors.Wrap(err, "handleMessage failed"))
		return
	}

	// read from s3 object and create event for each log line
	err = p.handleS3Objects(svcS3, s3Infos, errC)
	if err != nil {
		err = errors.Wrap(err, "handleS3Objects failed")
		errC <- err
		p.logger.Error(err)
	}
}

func (p *s3Input) processorKeepAlive(svcSQS sqsiface.ClientAPI, message sqs.Message, queueURL string, visibilityTimeout int64, wg *sync.WaitGroup, errC chan error) {
	defer wg.Done()
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
				p.logger.Debug("ACK done, deleting message from SQS")
				// only delete sqs message when errC is closed with no error
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

func (p *s3Input) receiveMessage(svcSQS sqsiface.ClientAPI, visibilityTimeout int64) (*sqs.ReceiveMessageResponse, error) {
	// receive messages from sqs
	req := svcSQS.ReceiveMessageRequest(
		&sqs.ReceiveMessageInput{
			QueueUrl:              &p.config.QueueURL,
			MessageAttributeNames: []string{"All"},
			MaxNumberOfMessages:   &maxNumberOfMessage,
			VisibilityTimeout:     &visibilityTimeout,
			WaitTimeSeconds:       &waitTimeSecond,
		})

	return req.Send(p.context)
}

func (p *s3Input) changeVisibilityTimeout(queueURL string, visibilityTimeout int64, svcSQS sqsiface.ClientAPI, receiptHandle *string) error {
	req := svcSQS.ChangeMessageVisibilityRequest(&sqs.ChangeMessageVisibilityInput{
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

func (p *s3Input) handleS3Objects(svc s3iface.ClientAPI, s3Infos []s3Info, errC chan error) error {
	s3Context := &s3Context{
		refs: 1,
		errC: errC,
	}
	defer s3Context.done()

	for _, s3Info := range s3Infos {
		objectHash := s3ObjectHash(s3Info)

		// read from s3 object
		reader, err := p.newS3BucketReader(svc, s3Info)
		if err != nil {
			return errors.Wrap(err, "newS3BucketReader failed")
		}
		if reader == nil {
			continue
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
					event := createEvent(log, offset, s3Info, objectHash, s3Context)
					err = p.forwardEvent(event)
					if err != nil {
						err = errors.Wrapf(err, "forwardEvent failed for %v", s3Info.key)
						s3Context.Fail(err)
						return err
					}
					return nil
				}
				return errors.Wrapf(err, "ReadString failed for %v", s3Info.key)
			}

			// create event per log line
			offset += len([]byte(log))
			event := createEvent(log, offset, s3Info, objectHash, s3Context)
			err = p.forwardEvent(event)
			if err != nil {
				err = errors.Wrapf(err, "forwardEvent failed for %v", s3Info.key)
				s3Context.Fail(err)
				return err
			}
		}
	}

	return nil
}

func (p *s3Input) newS3BucketReader(svc s3iface.ClientAPI, s3Info s3Info) (*bufio.Reader, error) {
	s3GetObjectInput := &s3.GetObjectInput{
		Bucket: awssdk.String(s3Info.name),
		Key:    awssdk.String(s3Info.key),
	}
	req := svc.GetObjectRequest(s3GetObjectInput)

	resp, err := req.Send(p.context)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == awssdk.ErrCodeRequestCanceled {
				return nil, nil
			}

			if awsErr.Code() == "NoSuchKey" {
				p.logger.Warn("Cannot find s3 file with key ", s3Info.key)
				return nil, nil
			}
		}
		return nil, errors.Wrapf(err, "s3 get object request failed %v", s3Info.key)
	}

	if resp.Body == nil {
		return nil, errors.New("s3 get object response body is empty")
	}

	if p.config.GZip && strings.HasSuffix(s3Info.key, ".gz") {
		gzipReader, err := gzip.NewReader(resp.Body)

		if err != nil {
			return nil, errors.New("Failed to decompress gzipped file")
		}

		return bufio.NewReader(gzipReader), nil
	}

	return bufio.NewReader(resp.Body), nil
}

func (p *s3Input) forwardEvent(event beat.Event) error {
	ok := p.outlet.OnEvent(event)
	if !ok {
		return errOutletClosed
	}
	return nil
}

func (p *s3Input) deleteMessage(queueURL string, messagesReceiptHandle string, svcSQS sqsiface.ClientAPI) error {
	deleteMessageInput := &sqs.DeleteMessageInput{
		QueueUrl:      awssdk.String(queueURL),
		ReceiptHandle: awssdk.String(messagesReceiptHandle),
	}

	req := svcSQS.DeleteMessageRequest(deleteMessageInput)
	_, err := req.Send(p.context)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == awssdk.ErrCodeRequestCanceled {
			return nil
		}
		return errors.Wrap(err, "DeleteMessageRequest failed")
	}
	return nil
}

func createEvent(log string, offset int, s3Info s3Info, objectHash string, s3Context *s3Context) beat.Event {
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

	s3Context.Inc()
	return beat.Event{
		Timestamp: time.Now(),
		Fields:    f,
		Meta:      common.MapStr{"id": objectHash + "-" + fmt.Sprintf("%012d", offset)},
		Private:   s3Context,
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

func (c *s3Context) Inc() {
	c.mux.Lock()
	defer c.mux.Unlock()
	c.refs++
}
