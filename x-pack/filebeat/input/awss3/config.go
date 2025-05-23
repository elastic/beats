// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/dustin/go-humanize"

	"github.com/elastic/beats/v7/libbeat/common/cfgtype"
	"github.com/elastic/beats/v7/libbeat/common/match"
	"github.com/elastic/beats/v7/libbeat/reader/parser"
	"github.com/elastic/beats/v7/libbeat/reader/readfile"
	"github.com/elastic/beats/v7/libbeat/reader/readfile/encoding"
	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
)

type config struct {
	APITimeout         time.Duration        `config:"api_timeout"`
	AWSConfig          awscommon.ConfigAWS  `config:",inline"`
	AccessPointARN     string               `config:"access_point_arn"`
	BackupConfig       backupConfig         `config:",inline"`
	BucketARN          string               `config:"bucket_arn"`
	BucketListInterval time.Duration        `config:"bucket_list_interval"`
	BucketListPrefix   string               `config:"bucket_list_prefix"`
	FileSelectors      []fileSelectorConfig `config:"file_selectors"`
	IgnoreOlder        time.Duration        `config:"ignore_older"`
	NonAWSBucketName   string               `config:"non_aws_bucket_name"`
	NumberOfWorkers    int                  `config:"number_of_workers"`
	PathStyle          bool                 `config:"path_style"`
	ProviderOverride   string               `config:"provider"`
	QueueURL           string               `config:"queue_url"`
	ReaderConfig       readerConfig         `config:",inline"` // Reader options to apply when no file_selectors are used.
	RegionName         string               `config:"region"`
	SQSMaxReceiveCount int                  `config:"sqs.max_receive_count"` // The max number of times a message should be received (retried) before deleting it.
	SQSScript          *scriptConfig        `config:"sqs.notification_parsing_script"`
	SQSWaitTime        time.Duration        `config:"sqs.wait_time"`           // The max duration for which the SQS ReceiveMessage call waits for a message to arrive in the queue before returning.
	SQSGraceTime       time.Duration        `config:"sqs.shutdown_grace_time"` // The time that the processing loop will wait for messages before shutting down.
	StartTimestamp     string               `config:"start_timestamp"`
	VisibilityTimeout  time.Duration        `config:"visibility_timeout"`
}

func defaultConfig() config {
	c := config{
		APITimeout:         120 * time.Second,
		VisibilityTimeout:  300 * time.Second,
		BucketListInterval: 120 * time.Second,
		BucketListPrefix:   "",
		SQSWaitTime:        20 * time.Second,
		SQSGraceTime:       20 * time.Second,
		SQSMaxReceiveCount: 5,
		NumberOfWorkers:    5,
		PathStyle:          false,
	}
	c.ReaderConfig.InitDefaults()
	return c
}

func (c *config) Validate() error {
	configs := []bool{c.QueueURL != "", c.BucketARN != "", c.AccessPointARN != "", c.NonAWSBucketName != ""}
	enabled := []bool{}
	for i := range configs {
		if configs[i] {
			enabled = append(enabled, configs[i])
		}
	}
	if len(enabled) == 0 {
		return errors.New("neither queue_url, bucket_arn, access_point_arn, nor non_aws_bucket_name were provided")
	} else if len(enabled) > 1 {
		return fmt.Errorf("queue_url <%v>, bucket_arn <%v>, access_point_arn <%v>, non_aws_bucket_name <%v> "+
			"cannot be set at the same time", c.QueueURL, c.BucketARN, c.AccessPointARN, c.NonAWSBucketName)
	}

	if (c.BucketARN != "" || c.AccessPointARN != "" || c.NonAWSBucketName != "") && c.BucketListInterval <= 0 {
		return fmt.Errorf("bucket_list_interval <%v> must be greater than 0", c.BucketListInterval)
	}

	if (c.BucketARN != "" || c.AccessPointARN != "" || c.NonAWSBucketName != "") && c.NumberOfWorkers <= 0 {
		return fmt.Errorf("number_of_workers <%v> must be greater than 0", c.NumberOfWorkers)
	}

	if c.AccessPointARN != "" && !isValidAccessPointARN(c.AccessPointARN) {
		return fmt.Errorf("invalid format for access_point_arn <%v>", c.AccessPointARN)
	}

	if c.QueueURL != "" && (c.VisibilityTimeout <= 0 || c.VisibilityTimeout.Hours() > 12) {
		return fmt.Errorf("visibility_timeout <%v> must be greater than 0 and "+
			"less than or equal to 12h", c.VisibilityTimeout)
	}

	if c.QueueURL != "" && (c.SQSWaitTime <= 0 || c.SQSWaitTime.Seconds() > 20) {
		return fmt.Errorf("wait_time <%v> must be greater than 0 and "+
			"less than or equal to 20s", c.SQSWaitTime)
	}

	if c.QueueURL != "" && c.SQSGraceTime < 0 {
		return fmt.Errorf("shutdown_grace_time <%v> must not be negative", c.SQSGraceTime)
	}

	if c.QueueURL != "" && c.APITimeout < c.SQSWaitTime {
		return fmt.Errorf("api_timeout <%v> must be greater than the sqs.wait_time <%v",
			c.APITimeout, c.SQSWaitTime)
	}

	if c.AWSConfig.FIPSEnabled && c.NonAWSBucketName != "" {
		return errors.New("fips_enabled cannot be used with a non-AWS S3 bucket")
	}
	if c.PathStyle && c.NonAWSBucketName == "" && c.QueueURL == "" {
		return errors.New("path_style can only be used when polling non-AWS S3 services or SQS/SNS QueueURL")
	}
	if c.ProviderOverride != "" && c.NonAWSBucketName == "" {
		return errors.New("provider can only be overridden when polling non-AWS S3 services")
	}
	if c.AWSConfig.Endpoint != "" {
		// Make sure the given endpoint can be parsed
		_, err := url.Parse(c.AWSConfig.Endpoint)
		if err != nil {
			return fmt.Errorf("failed to parse endpoint: %w", err)
		}
	}
	if c.BackupConfig.NonAWSBackupToBucketName != "" && c.NonAWSBucketName == "" {
		return errors.New("backup to non-AWS bucket can only be used for non-AWS sources")
	}
	if c.BackupConfig.BackupToBucketArn != "" && c.BucketARN == "" && c.AccessPointARN == "" {
		return errors.New("backup to AWS bucket can only be used for AWS sources")
	}
	if c.BackupConfig.BackupToBucketArn != "" && c.BackupConfig.NonAWSBackupToBucketName != "" {
		return errors.New("backup_to_bucket_arn and non_aws_backup_to_bucket_name cannot be used together")
	}
	if c.BackupConfig.GetBucketName() != "" && c.QueueURL == "" {
		if (c.BackupConfig.BackupToBucketArn != "" &&
			(c.BackupConfig.BackupToBucketArn == c.BucketARN || c.BackupConfig.BackupToBucketArn == c.AccessPointARN)) ||
			(c.BackupConfig.NonAWSBackupToBucketName != "" && c.BackupConfig.NonAWSBackupToBucketName == c.NonAWSBucketName) {
			if c.BackupConfig.BackupToBucketPrefix == "" {
				return errors.New("backup_to_bucket_prefix is a required property when source and backup bucket are the same")
			}
			if c.BackupConfig.BackupToBucketPrefix == c.BucketListPrefix {
				return errors.New("backup_to_bucket_prefix cannot be the same as bucket_list_prefix, this will create an infinite loop")
			}
		}
	}

	if c.StartTimestamp != "" {
		_, err := time.Parse(time.RFC3339, c.StartTimestamp)
		if err != nil {
			return fmt.Errorf("invalid input for start_timestamp: %w", err)
		}
	}

	return nil
}

type backupConfig struct {
	BackupToBucketArn        string `config:"backup_to_bucket_arn"`
	NonAWSBackupToBucketName string `config:"non_aws_backup_to_bucket_name"`
	BackupToBucketPrefix     string `config:"backup_to_bucket_prefix"`
	Delete                   bool   `config:"delete_after_backup"`
}

func (c *backupConfig) GetBucketName() string {
	if c.BackupToBucketArn != "" {
		return getBucketNameFromARN(c.BackupToBucketArn)
	}
	return c.NonAWSBackupToBucketName
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
	Decoding                 decoderConfig           `config:"decoding"`
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

func (c config) getBucketName() string {
	if c.NonAWSBucketName != "" {
		return c.NonAWSBucketName
	}
	if c.AccessPointARN != "" {
		return c.AccessPointARN
	}
	if c.BucketARN != "" {
		return getBucketNameFromARN(c.BucketARN)
	}
	return ""
}

func (c config) getBucketARN() string {
	if c.NonAWSBucketName != "" {
		return c.NonAWSBucketName
	}
	if c.BucketARN != "" {
		return c.BucketARN
	}
	if c.AccessPointARN != "" {
		return c.AccessPointARN
	}
	return ""
}

// An AWS SDK callback to apply the input configuration's settings to an S3
// options struct.
// Should be provided as a parameter to s3.NewFromConfig.
func (c config) s3ConfigModifier(o *s3.Options) {
	if c.AWSConfig.FIPSEnabled {
		o.EndpointOptions.UseFIPSEndpoint = awssdk.FIPSEndpointStateEnabled
	}
	// Apply slightly different endpoint resolvers depending on whether we're in S3 or SQS mode.
	if c.AWSConfig.Endpoint != "" {
		//nolint:staticcheck // haven't migrated to the new interface yet
		o.EndpointResolver = s3.EndpointResolverFromURL(c.AWSConfig.Endpoint,
			func(e *awssdk.Endpoint) {
				// The S3 hostname is immutable in bucket polling mode, mutable otherwise.
				e.HostnameImmutable = (c.getBucketARN() != "")
			})
	}
	o.UsePathStyle = c.PathStyle

	o.Retryer = retry.NewStandard(func(so *retry.StandardOptions) {
		so.MaxAttempts = 5
		// Recover quickly when requests start working again
		so.NoRetryIncrement = 100
	})
}

// An AWS SDK callback to apply the input configuration's settings to an SQS
// options struct.
// Should be provided as a parameter to sqs.NewFromConfig.
func (c config) sqsConfigModifier(o *sqs.Options) {
	if c.AWSConfig.FIPSEnabled {
		o.EndpointOptions.UseFIPSEndpoint = awssdk.FIPSEndpointStateEnabled
	}
	if c.AWSConfig.Endpoint != "" {
		//nolint:staticcheck // not changing through this PR
		o.EndpointResolver = sqs.EndpointResolverFromURL(c.AWSConfig.Endpoint)
	}
}

func (c config) getFileSelectors() []fileSelectorConfig {
	if len(c.FileSelectors) > 0 {
		return c.FileSelectors
	}
	return []fileSelectorConfig{{ReaderConfig: c.ReaderConfig}}
}

// Helper function to detect if an ARN is an Access Point
func isValidAccessPointARN(arn string) bool {
	parts := strings.Split(arn, ":")
	return len(parts) >= 6 &&
		strings.HasPrefix(parts[5], "accesspoint/") &&
		len(strings.TrimPrefix(parts[5], "accesspoint/")) > 0
}
