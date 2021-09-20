// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"fmt"
	"time"

	"github.com/dustin/go-humanize"

	"github.com/elastic/beats/v7/libbeat/common/cfgtype"
	"github.com/elastic/beats/v7/libbeat/common/match"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/reader/parser"
	"github.com/elastic/beats/v7/libbeat/reader/readfile"
	"github.com/elastic/beats/v7/libbeat/reader/readfile/encoding"
	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
)

type config struct {
	APITimeout          time.Duration        `config:"api_timeout"`
	VisibilityTimeout   time.Duration        `config:"visibility_timeout"`
	SQSWaitTime         time.Duration        `config:"sqs.wait_time"`         // The max duration for which the SQS ReceiveMessage call waits for a message to arrive in the queue before returning.
	SQSMaxReceiveCount  int                  `config:"sqs.max_receive_count"` // The max number of times a message should be received (retried) before deleting it.
	FIPSEnabled         bool                 `config:"fips_enabled"`
	MaxNumberOfMessages int                  `config:"max_number_of_messages"`
	QueueURL            string               `config:"queue_url"`
	BucketARN           string               `config:"bucket_arn"`
	BucketListInterval  time.Duration        `config:"bucket_list_interval"`
	NumberOfWorkers     int                  `config:"number_of_workers"`
	AWSConfig           awscommon.ConfigAWS  `config:",inline"`
	FileSelectors       []fileSelectorConfig `config:"file_selectors"`
	ReaderConfig        readerConfig         `config:",inline"` // Reader options to apply when no file_selectors are used.
}

func defaultConfig() config {
	c := config{
		APITimeout:          120 * time.Second,
		VisibilityTimeout:   300 * time.Second,
		BucketListInterval:  120 * time.Second,
		SQSWaitTime:         20 * time.Second,
		SQSMaxReceiveCount:  5,
		FIPSEnabled:         false,
		MaxNumberOfMessages: 5,
	}
	c.ReaderConfig.InitDefaults()
	return c
}

func (c *config) Validate() error {
	if c.QueueURL == "" && c.BucketARN == "" {
		logp.NewLogger(inputName).Warnf("neither queue_url nor bucket_arn were provided, input %s will stop", inputName)
		return nil
	}

	if c.QueueURL != "" && c.BucketARN != "" {
		return fmt.Errorf("queue_url <%v> and bucket_arn <%v> "+
			"cannot be set at the same time", c.QueueURL, c.BucketARN)
	}

	if c.BucketARN != "" && c.BucketListInterval <= 0 {
		return fmt.Errorf("bucket_list_interval <%v> must be greater than 0", c.BucketListInterval)
	}

	if c.BucketARN != "" && c.NumberOfWorkers <= 0 {
		return fmt.Errorf("number_of_workers <%v> must be greater than 0", c.NumberOfWorkers)
	}

	if c.QueueURL != "" && (c.VisibilityTimeout <= 0 || c.VisibilityTimeout.Hours() > 12) {
		return fmt.Errorf("visibility_timeout <%v> must be greater than 0 and "+
			"less than or equal to 12h", c.VisibilityTimeout)
	}

	if c.QueueURL != "" && (c.SQSWaitTime <= 0 || c.SQSWaitTime.Seconds() > 20) {
		return fmt.Errorf("wait_time <%v> must be greater than 0 and "+
			"less than or equal to 20s", c.SQSWaitTime)
	}

	if c.QueueURL != "" && c.MaxNumberOfMessages <= 0 {
		return fmt.Errorf("max_number_of_messages <%v> must be greater than 0",
			c.MaxNumberOfMessages)
	}

	if c.QueueURL != "" && c.APITimeout < c.SQSWaitTime {
		return fmt.Errorf("api_timeout <%v> must be greater than the sqs.wait_time <%v",
			c.APITimeout, c.SQSWaitTime)
	}

	return nil
}

// fileSelectorConfig defines reader configuration that applies to a subset
// of S3 objects whose URL matches the given regex.
type fileSelectorConfig struct {
	Regex        *match.Matcher `config:"regex" validate:"required"`
	ReaderConfig readerConfig   `config:",inline"`
}

// readerConfig defines the options for reading the content of an S3 object.
type readerConfig struct {
	BufferSize               cfgtype.ByteSize        `config:"buffer_size"`
	ContentType              string                  `config:"content_type"`
	Encoding                 string                  `config:"encoding"`
	ExpandEventListFromField string                  `config:"expand_event_list_from_field"`
	IncludeS3Metadata        []string                `config:"include_s3_metadata"`
	LineTerminator           readfile.LineTerminator `config:"line_terminator"`
	MaxBytes                 cfgtype.ByteSize        `config:"max_bytes"`
	Parsers                  parser.Config           `config:",inline"`
}

func (rc *readerConfig) Validate() error {
	if rc.BufferSize <= 0 {
		return fmt.Errorf("buffer_size <%v> must be greater than 0", rc.BufferSize)
	}

	if rc.MaxBytes <= 0 {
		return fmt.Errorf("max_bytes <%v> must be greater than 0", rc.MaxBytes)
	}

	if rc.ExpandEventListFromField != "" && rc.ContentType != "" && rc.ContentType != "application/json" {
		return fmt.Errorf("content_type must be `application/json` when expand_event_list_from_field is used")
	}

	_, found := encoding.FindEncoding(rc.Encoding)
	if !found {
		return fmt.Errorf("encoding type <%v> not found", rc.Encoding)
	}

	return nil
}

func (rc *readerConfig) InitDefaults() {
	rc.BufferSize = 16 * humanize.KiByte
	rc.MaxBytes = 10 * humanize.MiByte
	rc.LineTerminator = readfile.AutoLineTerminator
}
