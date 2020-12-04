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

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/go-concert/unison"
)

type s3Collector struct {
	cancellation context.Context
	logger       *logp.Logger

	config            *config
	visibilityTimeout int64

	sqs       sqsiface.ClientAPI
	s3        s3iface.ClientAPI
	publisher beat.Client
}

type s3Info struct {
	name                     string
	key                      string
	region                   string
	arn                      string
	expandEventListFromField string
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

// The duration (in seconds) for which the call waits for a message to arrive
// in the queue before returning. If a message is available, the call returns
// sooner than WaitTimeSeconds. If no messages are available and the wait time
// expires, the call returns successfully with an empty list of messages.
var waitTimeSecond uint8 = 10

func (c *s3Collector) run() {
	defer c.logger.Info("s3 input worker has stopped.")
	c.logger.Info("s3 input worker has started.")
	for c.cancellation.Err() == nil {
		// receive messages from sqs
		output, err := c.receiveMessage(c.sqs, c.visibilityTimeout)
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == awssdk.ErrCodeRequestCanceled {
				continue
			}
			c.logger.Error("SQS ReceiveMessageRequest failed: ", err)
			continue
		}

		if output == nil || len(output.Messages) == 0 {
			c.logger.Debug("no message received from SQS")
			continue
		}

		// process messages received from sqs, get logs from s3 and create events
		c.processor(c.config.QueueURL, output.Messages, c.visibilityTimeout, c.s3, c.sqs)
	}
}

func (c *s3Collector) processor(queueURL string, messages []sqs.Message, visibilityTimeout int64, svcS3 s3iface.ClientAPI, svcSQS sqsiface.ClientAPI) {
	var grp unison.MultiErrGroup
	numMessages := len(messages)
	c.logger.Debugf("Processing %v messages", numMessages)

	// process messages received from sqs
	for i := range messages {
		i := i
		errC := make(chan error)
		grp.Go(func() (err error) {
			return c.processMessage(svcS3, messages[i], errC)
		})
		grp.Go(func() (err error) {
			return c.processorKeepAlive(svcSQS, messages[i], queueURL, visibilityTimeout, errC)
		})
	}
	grp.Wait()
}

func (c *s3Collector) processMessage(svcS3 s3iface.ClientAPI, message sqs.Message, errC chan error) error {
	s3Infos, err := c.handleSQSMessage(message)
	if err != nil {
		c.logger.Error(fmt.Errorf("handleSQSMessage failed: %w", err))
		return err
	}
	c.logger.Debugf("handleSQSMessage succeed and returned %v sets of S3 log info", len(s3Infos))

	// read from s3 object and create event for each log line
	err = c.handleS3Objects(svcS3, s3Infos, errC)
	if err != nil {
		c.logger.Error(fmt.Errorf("handleS3Objects failed: %w", err))
		return err
	}
	c.logger.Debugf("handleS3Objects succeed")
	return nil
}

func (c *s3Collector) processorKeepAlive(svcSQS sqsiface.ClientAPI, message sqs.Message, queueURL string, visibilityTimeout int64, errC chan error) error {
	for {
		select {
		case <-c.cancellation.Done():
			return nil
		case err := <-errC:
			if err != nil {
				if err == context.DeadlineExceeded {
					c.logger.Info("Context deadline exceeded, updating visibility timeout")
				} else {
					c.logger.Warnf("Processing message failed '%w', updating visibility timeout", err)
				}

				err := c.changeVisibilityTimeout(queueURL, visibilityTimeout, svcSQS, message.ReceiptHandle)
				if err != nil {
					c.logger.Error(fmt.Errorf("SQS ChangeMessageVisibilityRequest failed: %w", err))
				}
				c.logger.Infof("Message visibility timeout updated to %v", visibilityTimeout)
			} else {
				// When ACK done, message will be deleted. Or when message is
				// not s3 ObjectCreated event related(handleSQSMessage function
				// failed), it will be removed as well.
				c.logger.Debug("Deleting message from SQS: ", *message.MessageId)
				// only delete sqs message when errC is closed with no error
				err := c.deleteMessage(queueURL, *message.ReceiptHandle, svcSQS)
				if err != nil {
					c.logger.Error(fmt.Errorf("deleteMessages failed: %w", err))
				}
			}
			return err
		case <-time.After(time.Duration(visibilityTimeout/2) * time.Second):
			c.logger.Warn("Half of the set visibilityTimeout passed, visibility timeout needs to be updated")
			// If half of the set visibilityTimeout passed and this is
			// still ongoing, then change visibility timeout.
			err := c.changeVisibilityTimeout(queueURL, visibilityTimeout, svcSQS, message.ReceiptHandle)
			if err != nil {
				c.logger.Error(fmt.Errorf("SQS ChangeMessageVisibilityRequest failed: %w", err))
			}
			c.logger.Infof("Message visibility timeout updated to %v seconds", visibilityTimeout)
			return err
		}
	}
}

func (c *s3Collector) receiveMessage(svcSQS sqsiface.ClientAPI, visibilityTimeout int64) (*sqs.ReceiveMessageResponse, error) {
	// receive messages from sqs
	req := svcSQS.ReceiveMessageRequest(
		&sqs.ReceiveMessageInput{
			QueueUrl:              &c.config.QueueURL,
			MessageAttributeNames: []string{"All"},
			MaxNumberOfMessages:   awssdk.Int64(int64(c.config.MaxNumberOfMessages)),
			VisibilityTimeout:     &visibilityTimeout,
			WaitTimeSeconds:       awssdk.Int64(int64(waitTimeSecond)),
		})

	// The Context will interrupt the request if the timeout expires.
	sendCtx, cancelFn := context.WithTimeout(c.cancellation, c.config.APITimeout)
	defer cancelFn()

	return req.Send(sendCtx)
}

func (c *s3Collector) changeVisibilityTimeout(queueURL string, visibilityTimeout int64, svcSQS sqsiface.ClientAPI, receiptHandle *string) error {
	req := svcSQS.ChangeMessageVisibilityRequest(&sqs.ChangeMessageVisibilityInput{
		QueueUrl:          &queueURL,
		VisibilityTimeout: &visibilityTimeout,
		ReceiptHandle:     receiptHandle,
	})

	// The Context will interrupt the request if the timeout expires.
	sendCtx, cancelFn := context.WithTimeout(c.cancellation, c.config.APITimeout)
	defer cancelFn()

	_, err := req.Send(sendCtx)
	return err
}

func getRegionFromQueueURL(queueURL string) (string, error) {
	// get region from queueURL
	// Example: https://sqs.us-east-1.amazonaws.com/627959692251/test-s3-logs
	queueURLSplit := strings.Split(queueURL, ".")
	if queueURLSplit[0] == "https://sqs" && queueURLSplit[2] == "amazonaws" {
		return queueURLSplit[1], nil
	}
	return "", fmt.Errorf("queueURL is not in format: https://sqs.{REGION_ENDPOINT}.amazonaws.com/{ACCOUNT_NUMBER}/{QUEUE_NAME}")
}

// handle message
func (c *s3Collector) handleSQSMessage(m sqs.Message) ([]s3Info, error) {
	msg := sqsMessage{}
	err := json.Unmarshal([]byte(*m.Body), &msg)
	if err != nil {
		return nil, fmt.Errorf("json unmarshal sqs message body failed: %w", err)
	}

	var s3Infos []s3Info
	for _, record := range msg.Records {
		if record.EventSource != "aws:s3" || !strings.HasPrefix(record.EventName, "ObjectCreated:") {
			return nil, fmt.Errorf("this SQS queue should be dedicated to s3 ObjectCreated event notifications")
		}
		// Unescape substrings from s3 log name. For example, convert "%3D" back to "="
		filename, err := url.QueryUnescape(record.S3.object.Key)
		if err != nil {
			return nil, fmt.Errorf("url.QueryUnescape failed for '%s': %w", record.S3.object.Key, err)
		}

		if len(c.config.FileSelectors) == 0 {
			s3Infos = append(s3Infos, s3Info{
				region:                   record.AwsRegion,
				name:                     record.S3.bucket.Name,
				key:                      filename,
				arn:                      record.S3.bucket.Arn,
				expandEventListFromField: c.config.ExpandEventListFromField,
			})
			continue
		}

		for _, fs := range c.config.FileSelectors {
			if fs.Regex == nil {
				continue
			}
			if fs.Regex.MatchString(filename) {
				s3Infos = append(s3Infos, s3Info{
					region:                   record.AwsRegion,
					name:                     record.S3.bucket.Name,
					key:                      filename,
					arn:                      record.S3.bucket.Arn,
					expandEventListFromField: fs.ExpandEventListFromField,
				})
				break
			}
		}
	}
	return s3Infos, nil
}

func (c *s3Collector) handleS3Objects(svc s3iface.ClientAPI, s3Infos []s3Info, errC chan error) error {
	s3Ctx := &s3Context{
		refs: 1,
		errC: errC,
	}
	defer s3Ctx.done()

	for _, info := range s3Infos {
		c.logger.Debugf("Processing file from s3 bucket \"%s\" with name \"%s\"", info.name, info.key)
		err := c.createEventsFromS3Info(svc, info, s3Ctx)
		if err != nil {
			c.logger.Error(fmt.Errorf("createEventsFromS3Info failed processing file from s3 bucket \"%s\" with name \"%s\": %w", info.name, info.key, err))
			s3Ctx.setError(err)
		}
	}
	return nil
}

func (c *s3Collector) createEventsFromS3Info(svc s3iface.ClientAPI, info s3Info, s3Ctx *s3Context) error {
	objectHash := s3ObjectHash(info)

	// Download the S3 object using GetObjectRequest.
	s3GetObjectInput := &s3.GetObjectInput{
		Bucket: awssdk.String(info.name),
		Key:    awssdk.String(info.key),
	}
	req := svc.GetObjectRequest(s3GetObjectInput)

	// The Context will interrupt the request if the timeout expires.
	ctx, cancelFn := context.WithTimeout(c.cancellation, c.config.APITimeout)
	defer cancelFn()

	resp, err := req.Send(ctx)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			// If the SDK can determine the request or retry delay was canceled
			// by a context the ErrCodeRequestCanceled error will be returned.
			if awsErr.Code() == awssdk.ErrCodeRequestCanceled {
				c.logger.Error(fmt.Errorf("s3 GetObjectRequest canceled for '%s' from S3 bucket '%s': %w", info.key, info.name, err))
				return err
			}

			if awsErr.Code() == "NoSuchKey" {
				c.logger.Warnf("Cannot find s3 file '%s' from S3 bucket '%s'", info.key, info.name)
				return nil
			}
		}
		return fmt.Errorf("s3 GetObjectRequest failed for '%s' from S3 bucket '%s': %w", info.key, info.name, err)
	}

	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)

	isS3ObjGzipped, err := isStreamGzipped(reader)
	if err != nil {
		c.logger.Error(fmt.Errorf("could not determine if S3 object is gzipped: %w", err))
		return err
	}

	if isS3ObjGzipped {
		gzipReader, err := gzip.NewReader(reader)
		if err != nil {
			c.logger.Error(fmt.Errorf("gzip.NewReader failed for '%s' from S3 bucket '%s': %w", info.key, info.name, err))
			return err
		}
		reader = bufio.NewReader(gzipReader)
		gzipReader.Close()
	}

	// Decode JSON documents when content-type is "application/json" or expand_event_list_from_field is given in config
	if resp.ContentType != nil && *resp.ContentType == "application/json" || info.expandEventListFromField != "" {
		decoder := json.NewDecoder(reader)
		err := c.decodeJSON(decoder, objectHash, info, s3Ctx)
		if err != nil {
			c.logger.Error(fmt.Errorf("decodeJSONWithKey failed for '%s' from S3 bucket '%s': %w", info.key, info.name, err))
			return err
		}
		return nil
	}

	// handle s3 objects that are not json content-type
	var offset int64
	for {
		log, err := readStringAndTrimDelimiter(reader)
		if err == io.EOF {
			// create event for last line
			offset += int64(len(log))
			event := createEvent(log, offset, info, objectHash, s3Ctx)
			err = c.forwardEvent(event)
			if err != nil {
				c.logger.Error(fmt.Errorf("forwardEvent failed: %w", err))
				return err
			}
			return nil
		} else if err != nil {
			c.logger.Error(fmt.Errorf("readStringAndTrimDelimiter failed: %w", err))
			return err
		}

		if log == "" {
			break
		}

		// create event per log line
		offset += int64(len(log))
		event := createEvent(log, offset, info, objectHash, s3Ctx)
		err = c.forwardEvent(event)
		if err != nil {
			c.logger.Error(fmt.Errorf("forwardEvent failed: %w", err))
			return err
		}
	}
	return nil
}

func (c *s3Collector) decodeJSON(decoder *json.Decoder, objectHash string, s3Info s3Info, s3Ctx *s3Context) error {
	var offset int64
	for {
		var jsonFields interface{}
		err := decoder.Decode(&jsonFields)
		if jsonFields == nil {
			return nil
		}

		if err == io.EOF {
			offsetNew, err := c.jsonFieldsType(jsonFields, offset, objectHash, s3Info, s3Ctx)
			if err != nil {
				return err
			}
			offset = offsetNew
		} else if err != nil {
			// decode json failed, skip this log file
			err = fmt.Errorf("decode json failed for '%s' from S3 bucket '%s', skipping this file: %w", s3Info.key, s3Info.name, err)
			c.logger.Warn(err)
			return nil
		}

		offset, err = c.jsonFieldsType(jsonFields, offset, objectHash, s3Info, s3Ctx)
		if err != nil {
			return err
		}
	}
}

func (c *s3Collector) jsonFieldsType(jsonFields interface{}, offset int64, objectHash string, s3Info s3Info, s3Ctx *s3Context) (int64, error) {
	switch f := jsonFields.(type) {
	case map[string][]interface{}:
		if s3Info.expandEventListFromField != "" {
			textValues, ok := f[s3Info.expandEventListFromField]
			if !ok {
				err := fmt.Errorf("key '%s' not found", s3Info.expandEventListFromField)
				c.logger.Error(err)
				return offset, err
			}
			for _, v := range textValues {
				offset, err := c.convertJSONToEvent(v, offset, objectHash, s3Info, s3Ctx)
				if err != nil {
					err = fmt.Errorf("convertJSONToEvent failed for '%s' from S3 bucket '%s': %w", s3Info.key, s3Info.name, err)
					c.logger.Error(err)
					return offset, err
				}
			}
			return offset, nil
		}
	case map[string]interface{}:
		if s3Info.expandEventListFromField != "" {
			textValues, ok := f[s3Info.expandEventListFromField]
			if !ok {
				err := fmt.Errorf("key '%s' not found", s3Info.expandEventListFromField)
				c.logger.Error(err)
				return offset, err
			}

			valuesConverted := textValues.([]interface{})
			for _, textValue := range valuesConverted {
				offsetNew, err := c.convertJSONToEvent(textValue, offset, objectHash, s3Info, s3Ctx)
				if err != nil {
					err = fmt.Errorf("convertJSONToEvent failed for '%s' from S3 bucket '%s': %w", s3Info.key, s3Info.name, err)
					c.logger.Error(err)
					return offset, err
				}
				offset = offsetNew
			}
			return offset, nil
		}

		offset, err := c.convertJSONToEvent(f, offset, objectHash, s3Info, s3Ctx)
		if err != nil {
			err = fmt.Errorf("convertJSONToEvent failed for '%s' from S3 bucket '%s': %w", s3Info.key, s3Info.name, err)
			c.logger.Error(err)
			return offset, err
		}
		return offset, nil
	}
	return offset, nil
}

func (c *s3Collector) convertJSONToEvent(jsonFields interface{}, offset int64, objectHash string, s3Info s3Info, s3Ctx *s3Context) (int64, error) {
	vJSON, _ := json.Marshal(jsonFields)
	logOriginal := string(vJSON)
	log := trimLogDelimiter(logOriginal)
	offset += int64(len(log))
	event := createEvent(log, offset, s3Info, objectHash, s3Ctx)

	err := c.forwardEvent(event)
	if err != nil {
		err = fmt.Errorf("forwardEvent failed: %w", err)
		c.logger.Error(err)
		return offset, err
	}
	return offset, nil
}

func (c *s3Collector) forwardEvent(event beat.Event) error {
	c.publisher.Publish(event)
	return c.cancellation.Err()
}

func (c *s3Collector) deleteMessage(queueURL string, messagesReceiptHandle string, svcSQS sqsiface.ClientAPI) error {
	deleteMessageInput := &sqs.DeleteMessageInput{
		QueueUrl:      awssdk.String(queueURL),
		ReceiptHandle: awssdk.String(messagesReceiptHandle),
	}

	req := svcSQS.DeleteMessageRequest(deleteMessageInput)

	// The Context will interrupt the request if the timeout expires.
	ctx, cancelFn := context.WithTimeout(c.cancellation, c.config.APITimeout)
	defer cancelFn()

	_, err := req.Send(ctx)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == awssdk.ErrCodeRequestCanceled {
			return nil
		}
		return fmt.Errorf("SQS DeleteMessageRequest failed: %w", err)
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

func createEvent(log string, offset int64, info s3Info, objectHash string, s3Ctx *s3Context) beat.Event {
	s3Ctx.Inc()

	event := beat.Event{
		Timestamp: time.Now().UTC(),
		Fields: common.MapStr{
			"message": log,
			"log": common.MapStr{
				"offset": int64(offset),
				"file": common.MapStr{
					"path": constructObjectURL(info),
				},
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
	event.SetID(objectID(objectHash, offset))

	return event
}

func objectID(objectHash string, offset int64) string {
	return fmt.Sprintf("%s-%012d", objectHash, offset)
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
