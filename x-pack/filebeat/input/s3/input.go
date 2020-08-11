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
	"net/http"
	"net/url"
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

	"github.com/elastic/beats/v7/filebeat/channel"
	"github.com/elastic/beats/v7/filebeat/input"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/acker"
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/libbeat/logp"
	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
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

// s3Input is a input for s3
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

type s3BucketObject struct {
	bucket `json:"bucket"`
	object `json:"object"`
}

type sqsMessage struct {
	Records []struct {
		EventSource string         `json:"eventSource"`
		AwsRegion   string         `json:"awsRegion"`
		EventName   string         `json:"eventName"`
		S3          s3BucketObject `json:"s3"`
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
		ACKHandler: acker.ConnectionOnly(
			acker.EventPrivateReporter(func(_ int, privates []interface{}) {
				for _, private := range privates {
					if s3Context, ok := private.(*s3Context); ok {
						s3Context.done()
					}
				}
			}),
		),
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
		p.logger.Infof("visibility timeout is set to %v seconds", visibilityTimeout)
		p.logger.Infof("aws api timeout is set to %v", p.config.APITimeout)

		regionName, err := getRegionFromQueueURL(p.config.QueueURL)
		if err != nil {
			p.logger.Errorf("failed to get region name from queueURL: %v", p.config.QueueURL)
		}

		awsConfig := p.awsConfig.Copy()
		awsConfig.Region = regionName

		svcSQS := sqs.New(awscommon.EnrichAWSConfigWithEndpoint(p.config.AwsConfig.Endpoint, "sqs", regionName, awsConfig))
		svcS3 := s3.New(awscommon.EnrichAWSConfigWithEndpoint(p.config.AwsConfig.Endpoint, "s3", regionName, awsConfig))

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
			p.logger.Error("SQS ReceiveMessageRequest failed: ", err)
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
	p.logger.Debugf("Processing %v messages", numMessages)
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
		p.logger.Error(errors.Wrap(err, "handleSQSMessage failed"))
		return
	}
	p.logger.Debugf("handleSQSMessage succeed and returned %v sets of S3 log info", len(s3Infos))

	// read from s3 object and create event for each log line
	err = p.handleS3Objects(svcS3, s3Infos, errC)
	if err != nil {
		err = errors.Wrap(err, "handleS3Objects failed")
		p.logger.Error(err)
		return
	}
	p.logger.Debugf("handleS3Objects succeed")
}

func (p *s3Input) processorKeepAlive(svcSQS sqsiface.ClientAPI, message sqs.Message, queueURL string, visibilityTimeout int64, wg *sync.WaitGroup, errC chan error) {
	defer wg.Done()
	for {
		select {
		case <-p.close:
			return
		case err := <-errC:
			if err != nil {
				p.logger.Warn("Processing message failed, updating visibility timeout")
				err := p.changeVisibilityTimeout(queueURL, visibilityTimeout, svcSQS, message.ReceiptHandle)
				if err != nil {
					p.logger.Error(errors.Wrap(err, "SQS ChangeMessageVisibilityRequest failed"))
				}
				p.logger.Infof("Message visibility timeout updated to %v", visibilityTimeout)
			} else {
				// When ACK done, message will be deleted. Or when message is
				// not s3 ObjectCreated event related(handleSQSMessage function
				// failed), it will be removed as well.
				p.logger.Debug("Deleting message from SQS: ", message.MessageId)
				// only delete sqs message when errC is closed with no error
				err := p.deleteMessage(queueURL, *message.ReceiptHandle, svcSQS)
				if err != nil {
					p.logger.Error(errors.Wrap(err, "deleteMessages failed"))
				}
			}
			return
		case <-time.After(time.Duration(visibilityTimeout/2) * time.Second):
			p.logger.Warn("Half of the set visibilityTimeout passed, visibility timeout needs to be updated")
			// If half of the set visibilityTimeout passed and this is
			// still ongoing, then change visibility timeout.
			err := p.changeVisibilityTimeout(queueURL, visibilityTimeout, svcSQS, message.ReceiptHandle)
			if err != nil {
				p.logger.Error(errors.Wrap(err, "SQS ChangeMessageVisibilityRequest failed"))
			}
			p.logger.Infof("Message visibility timeout updated to %v seconds", visibilityTimeout)
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

	// The Context will interrupt the request if the timeout expires.
	ctx, cancelFn := context.WithTimeout(p.context, p.config.APITimeout)
	defer cancelFn()

	return req.Send(ctx)
}

func (p *s3Input) changeVisibilityTimeout(queueURL string, visibilityTimeout int64, svcSQS sqsiface.ClientAPI, receiptHandle *string) error {
	req := svcSQS.ChangeMessageVisibilityRequest(&sqs.ChangeMessageVisibilityInput{
		QueueUrl:          &queueURL,
		VisibilityTimeout: &visibilityTimeout,
		ReceiptHandle:     receiptHandle,
	})

	// The Context will interrupt the request if the timeout expires.
	ctx, cancelFn := context.WithTimeout(p.context, p.config.APITimeout)
	defer cancelFn()

	_, err := req.Send(ctx)
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
		if record.EventSource == "aws:s3" && strings.HasPrefix(record.EventName, "ObjectCreated:") {
			// Unescape substrings from s3 log name. For example, convert "%3D" back to "="
			filename, err := url.QueryUnescape(record.S3.object.Key)
			if err != nil {
				return nil, errors.Wrapf(err, "url.QueryUnescape failed for '%s'", record.S3.object.Key)
			}

			s3Infos = append(s3Infos, s3Info{
				region: record.AwsRegion,
				name:   record.S3.bucket.Name,
				key:    filename,
				arn:    record.S3.bucket.Arn,
			})
		} else {
			return nil, errors.New("this SQS queue should be dedicated to s3 ObjectCreated event notifications")
		}
	}
	return s3Infos, nil
}

func (p *s3Input) handleS3Objects(svc s3iface.ClientAPI, s3Infos []s3Info, errC chan error) error {
	s3Ctx := &s3Context{
		refs: 1,
		errC: errC,
	}
	defer s3Ctx.done()

	for _, info := range s3Infos {
		p.logger.Debugf("Processing file from s3 bucket \"%s\" with name \"%s\"", info.name, info.key)
		err := p.createEventsFromS3Info(svc, info, s3Ctx)
		if err != nil {
			err = errors.Wrapf(err, "createEventsFromS3Info failed processing file from s3 bucket \"%s\" with name \"%s\"", info.name, info.key)
			p.logger.Error(err)
			s3Ctx.setError(err)
		}
	}
	return nil
}

func (p *s3Input) createEventsFromS3Info(svc s3iface.ClientAPI, info s3Info, s3Ctx *s3Context) error {
	objectHash := s3ObjectHash(info)

	// Download the S3 object using GetObjectRequest.
	s3GetObjectInput := &s3.GetObjectInput{
		Bucket: awssdk.String(info.name),
		Key:    awssdk.String(info.key),
	}
	req := svc.GetObjectRequest(s3GetObjectInput)

	// The Context will interrupt the request if the timeout expires.
	ctx, cancelFn := context.WithTimeout(p.context, p.config.APITimeout)
	defer cancelFn()

	resp, err := req.Send(ctx)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			// If the SDK can determine the request or retry delay was canceled
			// by a context the ErrCodeRequestCanceled error will be returned.
			if awsErr.Code() == awssdk.ErrCodeRequestCanceled {
				err = errors.Wrapf(err, "S3 GetObjectRequest canceled for '%s' from S3 bucket '%s'", info.key, info.name)
				p.logger.Error(err)
				return err
			}

			if awsErr.Code() == "NoSuchKey" {
				p.logger.Warnf("Cannot find s3 file '%s' from S3 bucket '%s'", info.key, info.name)
				return nil
			}
		}
		return errors.Wrapf(err, "S3 GetObjectRequest failed for '%s' from S3 bucket '%s'", info.key, info.name)
	}

	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)

	isS3ObjGzipped, err := isStreamGzipped(reader)
	if err != nil {
		err = errors.Wrap(err, "could not determine if S3 object is gzipped")
		p.logger.Error(err)
		return err
	}

	if isS3ObjGzipped {
		gzipReader, err := gzip.NewReader(reader)
		if err != nil {
			err = errors.Wrapf(err, "gzip.NewReader failed for '%s' from S3 bucket '%s'", info.key, info.name)
			p.logger.Error(err)
			return err
		}
		reader = bufio.NewReader(gzipReader)
		gzipReader.Close()
	}

	// Decode JSON documents when content-type is "application/json" or expand_event_list_from_field is given in config
	if resp.ContentType != nil && *resp.ContentType == "application/json" || p.config.ExpandEventListFromField != "" {
		decoder := json.NewDecoder(reader)
		err := p.decodeJSON(decoder, objectHash, info, s3Ctx)
		if err != nil {
			err = errors.Wrapf(err, "decodeJSONWithKey failed for '%s' from S3 bucket '%s'", info.key, info.name)
			p.logger.Error(err)
			return err
		}
		return nil
	}

	// handle s3 objects that are not json content-type
	offset := 0
	for {
		log, err := readStringAndTrimDelimiter(reader)
		if err == io.EOF {
			// create event for last line
			offset += len([]byte(log))
			event := createEvent(log, offset, info, objectHash, s3Ctx)
			err = p.forwardEvent(event)
			if err != nil {
				err = errors.Wrap(err, "forwardEvent failed")
				p.logger.Error(err)
				return err
			}
			return nil
		} else if err != nil {
			err = errors.Wrap(err, "readStringAndTrimDelimiter failed")
			p.logger.Error(err)
			return err
		}

		if log == "" {
			break
		}

		// create event per log line
		offset += len([]byte(log))
		event := createEvent(log, offset, info, objectHash, s3Ctx)
		err = p.forwardEvent(event)
		if err != nil {
			err = errors.Wrap(err, "forwardEvent failed")
			p.logger.Error(err)
			return err
		}
	}
	return nil
}

func (p *s3Input) decodeJSON(decoder *json.Decoder, objectHash string, s3Info s3Info, s3Ctx *s3Context) error {
	offset := 0
	for {
		var jsonFields interface{}
		err := decoder.Decode(&jsonFields)
		if jsonFields == nil {
			return nil
		}

		if err == io.EOF {
			offset, err = p.jsonFieldsType(jsonFields, offset, objectHash, s3Info, s3Ctx)
			if err != nil {
				return err
			}
		} else if err != nil {
			// decode json failed, skip this log file
			err = errors.Wrapf(err, "decode json failed for '%s' from S3 bucket '%s', skipping this file", s3Info.key, s3Info.name)
			p.logger.Warn(err)
			return nil
		}

		offsetNew, err := p.jsonFieldsType(jsonFields, offset, objectHash, s3Info, s3Ctx)
		if err != nil {
			return err
		}
		offset = offsetNew
	}
}

func (p *s3Input) jsonFieldsType(jsonFields interface{}, offset int, objectHash string, s3Info s3Info, s3Ctx *s3Context) (int, error) {
	switch f := jsonFields.(type) {
	case map[string][]interface{}:
		if p.config.ExpandEventListFromField != "" {
			textValues, ok := f[p.config.ExpandEventListFromField]
			if !ok {
				err := errors.Errorf("key '%s' not found", p.config.ExpandEventListFromField)
				p.logger.Error(err)
				return offset, err
			}
			for _, v := range textValues {
				offset, err := p.convertJSONToEvent(v, offset, objectHash, s3Info, s3Ctx)
				if err != nil {
					err = errors.Wrapf(err, "convertJSONToEvent failed for '%s' from S3 bucket '%s'", s3Info.key, s3Info.name)
					p.logger.Error(err)
					return offset, err
				}
			}
			return offset, nil
		}
	case map[string]interface{}:
		if p.config.ExpandEventListFromField != "" {
			textValues, ok := f[p.config.ExpandEventListFromField]
			if !ok {
				err := errors.Errorf("key '%s' not found", p.config.ExpandEventListFromField)
				p.logger.Error(err)
				return offset, err
			}

			valuesConverted := textValues.([]interface{})
			for _, textValue := range valuesConverted {
				offsetNew, err := p.convertJSONToEvent(textValue, offset, objectHash, s3Info, s3Ctx)
				if err != nil {
					err = errors.Wrapf(err, "convertJSONToEvent failed for '%s' from S3 bucket '%s'", s3Info.key, s3Info.name)
					p.logger.Error(err)
					return offset, err
				}
				offset = offsetNew
			}
			return offset, nil
		}

		offset, err := p.convertJSONToEvent(f, offset, objectHash, s3Info, s3Ctx)
		if err != nil {
			err = errors.Wrapf(err, "convertJSONToEvent failed for '%s' from S3 bucket '%s'", s3Info.key, s3Info.name)
			p.logger.Error(err)
			return offset, err
		}
		return offset, nil
	}
	return offset, nil
}

func (p *s3Input) convertJSONToEvent(jsonFields interface{}, offset int, objectHash string, s3Info s3Info, s3Ctx *s3Context) (int, error) {
	vJSON, err := json.Marshal(jsonFields)
	logOriginal := string(vJSON)
	log := trimLogDelimiter(logOriginal)
	offset += len([]byte(log))
	event := createEvent(log, offset, s3Info, objectHash, s3Ctx)

	err = p.forwardEvent(event)
	if err != nil {
		err = errors.Wrap(err, "forwardEvent failed")
		p.logger.Error(err)
		return offset, err
	}
	return offset, nil
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

	// The Context will interrupt the request if the timeout expires.
	ctx, cancelFn := context.WithTimeout(p.context, p.config.APITimeout)
	defer cancelFn()

	_, err := req.Send(ctx)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == awssdk.ErrCodeRequestCanceled {
			return nil
		}
		return errors.Wrapf(err, "SQS DeleteMessageRequest failed in queue %s", queueURL)
	}
	return nil
}

func trimLogDelimiter(log string) string {
	return strings.TrimSuffix(log, "\n")
}

func readStringAndTrimDelimiter(reader *bufio.Reader) (string, error) {
	logOriginal, err := reader.ReadString('\n')
	if err != nil {
		return logOriginal, err
	}
	return trimLogDelimiter(logOriginal), nil
}

func createEvent(log string, offset int, info s3Info, objectHash string, s3Ctx *s3Context) beat.Event {
	s3Ctx.Inc()

	event := beat.Event{
		Timestamp: time.Now().UTC(),
		Fields: common.MapStr{
			"message": log,
			"log": common.MapStr{
				"offset":    int64(offset),
				"file.path": constructObjectURL(info),
			},
			"aws": common.MapStr{
				"s3": common.MapStr{
					"bucket": common.MapStr{
						"name": info.name,
						"arn":  info.arn},
					"object.key": info.key,
				},
			},
			"cloud": common.MapStr{
				"provider": "aws",
				"region":   info.region,
			},
		},
		Private: s3Ctx,
	}
	event.SetID(objectHash + "-" + fmt.Sprintf("%012d", offset))

	return event
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

func (c *s3Context) setError(err error) {
	// only care about the last error for now
	// TODO: add "Typed" error to error for context
	c.mux.Lock()
	defer c.mux.Unlock()
	c.err = err
}

func (c *s3Context) done() {
	c.mux.Lock()
	defer c.mux.Unlock()
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

// isStreamGzipped determines whether the given stream of bytes (encapsulated in a buffered reader)
// represents gzipped content or not. A buffered reader is used so the function can peek into the byte
// stream without consuming it. This makes it convenient for code executed after this function call
// to consume the stream if it wants.
func isStreamGzipped(r *bufio.Reader) (bool, error) {
	// Why 512? See https://godoc.org/net/http#DetectContentType
	buf, err := r.Peek(512)
	if err != nil && err != io.EOF {
		return false, err
	}

	switch http.DetectContentType(buf) {
	case "application/x-gzip", "application/zip":
		return true, nil
	default:
		return false, nil
	}
}
