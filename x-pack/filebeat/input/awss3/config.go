// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"errors"
	"fmt"
	"time"

	"github.com/dustin/go-humanize"

	"github.com/elastic/beats/v7/libbeat/common/cfgtype"
	"github.com/elastic/beats/v7/libbeat/common/match"
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
	SQSScript           *scriptConfig        `config:"sqs.notification_parsing_script"`
	MaxNumberOfMessages int                  `config:"max_number_of_messages"`
	QueueURL            string               `config:"queue_url"`
	BucketARN           string               `config:"bucket_arn"`
	NonAWSBucketName    string               `config:"non_aws_bucket_name"`
	BucketListInterval  time.Duration        `config:"bucket_list_interval"`
	BucketListPrefix    string               `config:"bucket_list_prefix"`
	NumberOfWorkers     int                  `config:"number_of_workers"`
	AWSConfig           awscommon.ConfigAWS  `config:",inline"`
	FileSelectors       []fileSelectorConfig `config:"file_selectors"`
	ReaderConfig        readerConfig         `config:",inline"` // Reader options to apply when no file_selectors are used.
	PathStyle           bool                 `config:"path_style"`
	ProviderOverride    string               `config:"provider"`
}

func defaultConfig() config {
	c := config{
		APITimeout:          120 * time.Second,
		VisibilityTimeout:   300 * time.Second,
		BucketListInterval:  120 * time.Second,
		BucketListPrefix:    "",
		SQSWaitTime:         20 * time.Second,
		SQSMaxReceiveCount:  5,
		MaxNumberOfMessages: 5,
		PathStyle:           false,
	}
	c.ReaderConfig.InitDefaults()
	return c
}

func (c *config) Validate() error {
	configs := []bool{c.QueueURL != "", c.BucketARN != "", c.NonAWSBucketName != ""}
	enabled := []bool{}
	for i := range configs {
		if configs[i] {
			enabled = append(enabled, configs[i])
		}
	}
	if len(enabled) == 0 {
		return errors.New("neither queue_url, bucket_arn nor non_aws_bucket_name were provided")
	} else if len(enabled) > 1 {
		return fmt.Errorf("queue_url <%v>, bucket_arn <%v>, non_aws_bucket_name <%v> "+
			"cannot be set at the same time", c.QueueURL, c.BucketARN, c.NonAWSBucketName)
	}

	if (c.BucketARN != "" || c.NonAWSBucketName != "") && c.BucketListInterval <= 0 {
		return fmt.Errorf("bucket_list_interval <%v> must be greater than 0", c.BucketListInterval)
	}

	if (c.BucketARN != "" || c.NonAWSBucketName != "") && c.NumberOfWorkers <= 0 {
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

	if c.AWSConfig.FIPSEnabled && c.NonAWSBucketName != "" {
		return errors.New("fips_enabled cannot be used with a non-AWS S3 bucket.")
	}
	if c.PathStyle && c.NonAWSBucketName == "" {
		return errors.New("path_style can only be used when polling non-AWS S3 services")
	}
	if c.ProviderOverride != "" && c.NonAWSBucketName == "" {
		return errors.New("provider can only be overriden when polling non-AWS S3 services")
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

type scriptConfig struct {
	Source            string                 `config:"source"`                               // Inline script to execute.
	File              string                 `config:"file"`                                 // Source file.
	Files             []string               `config:"files"`                                // Multiple source files.
	Params            map[string]interface{} `config:"params"`                               // Parameters to pass to script.
	Timeout           time.Duration          `config:"timeout" validate:"min=0"`             // Execution timeout.
	MaxCachedSessions int                    `config:"max_cached_sessions" validate:"min=0"` // Max. number of cached VM sessions.
}

// Validate returns an error if one (and only one) option is not set.
func (c scriptConfig) Validate() error {
	numConfigured := 0
	for _, set := range []bool{c.Source != "", c.File != "", len(c.Files) > 0} {
		if set {
			numConfigured++
		}
	}

	switch {
	case numConfigured == 0:
		return errors.New("javascript must be defined via 'file', " +
			"'files', or inline as 'source'")
	case numConfigured > 1:
		return errors.New("javascript can be defined in only one of " +
			"'file', 'files', or inline as 'source'")
	}

	return nil
}

func (rc *readerConfig) InitDefaults() {
	rc.BufferSize = 16 * humanize.KiByte
	rc.MaxBytes = 10 * humanize.MiByte
	rc.LineTerminator = readfile.AutoLineTerminator
}
