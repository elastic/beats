// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dustin/go-humanize"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/common/match"
	"github.com/elastic/beats/v7/libbeat/reader/parser"
	"github.com/elastic/beats/v7/libbeat/reader/readfile"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestConfig(t *testing.T) {
	const queueURL = "https://example.com"
	const s3Bucket = "arn:aws:s3:::aBucket"
	const nonAWSS3Bucket = "minio-bucket"
	makeConfig := func(quequeURL, s3Bucket string, nonAWSS3Bucket string) config {
		// Have a separate copy of defaults in the test to make it clear when
		// anyone changes the defaults.
		parserConf := parser.Config{}
		require.NoError(t, parserConf.Unpack(conf.MustNewConfigFrom("")))
		return config{
			QueueURL:            quequeURL,
			BucketARN:           s3Bucket,
			NonAWSBucketName:    nonAWSS3Bucket,
			APITimeout:          120 * time.Second,
			VisibilityTimeout:   300 * time.Second,
			SQSMaxReceiveCount:  5,
			SQSWaitTime:         20 * time.Second,
			BucketListInterval:  120 * time.Second,
			BucketListPrefix:    "",
			PathStyle:           false,
			MaxNumberOfMessages: 5,
			ReaderConfig: readerConfig{
				BufferSize:     16 * humanize.KiByte,
				MaxBytes:       10 * humanize.MiByte,
				LineTerminator: readfile.AutoLineTerminator,
				Parsers:        parserConf,
			},
		}
	}

	testCases := []struct {
		name           string
		queueURL       string
		s3Bucket       string
		nonAWSS3Bucket string
		config         mapstr.M
		expectedErr    string
		expectedCfg    func(queueURL, s3Bucket, nonAWSS3Bucket string) config
	}{
		{
			name:           "input with defaults for queueURL",
			queueURL:       queueURL,
			s3Bucket:       "",
			nonAWSS3Bucket: "",
			config: mapstr.M{
				"queue_url": queueURL,
			},
			expectedErr: "",
			expectedCfg: makeConfig,
		},
		{
			name:           "input with defaults for s3Bucket",
			queueURL:       "",
			s3Bucket:       s3Bucket,
			nonAWSS3Bucket: "",
			config: mapstr.M{
				"bucket_arn":        s3Bucket,
				"number_of_workers": 5,
			},
			expectedErr: "",
			expectedCfg: func(queueURL, s3Bucket, nonAWSS3Bucket string) config {
				c := makeConfig("", s3Bucket, "")
				c.NumberOfWorkers = 5
				return c
			},
		},
		{
			name:           "input with file_selectors",
			queueURL:       queueURL,
			s3Bucket:       "",
			nonAWSS3Bucket: "",
			config: mapstr.M{
				"queue_url": queueURL,
				"file_selectors": []mapstr.M{
					{
						"regex": "/CloudTrail/",
					},
				},
			},
			expectedErr: "",
			expectedCfg: func(queueURL, s3Bucket, nonAWSS3Bucket string) config {
				c := makeConfig(queueURL, "", "")
				regex := match.MustCompile("/CloudTrail/")
				c.FileSelectors = []fileSelectorConfig{
					{
						Regex:        &regex,
						ReaderConfig: c.ReaderConfig,
					},
				}
				return c
			},
		},
		{
			name:           "non-AWS_endpoint_with_explicit_region",
			queueURL:       queueURL,
			s3Bucket:       "",
			nonAWSS3Bucket: "",
			config: mapstr.M{
				"queue_url": queueURL,
				"region":    "region",
				"endpoint":  "ep",
			},
			expectedErr: "",
			expectedCfg: func(queueURL, s3Bucket, nonAWSS3Bucket string) config {
				c := makeConfig(queueURL, "", "")
				c.RegionName = "region"
				c.AWSConfig.Endpoint = "ep"
				return c
			},
		},
		{
			name:           "explicit_AWS_endpoint_with_explicit_region",
			queueURL:       queueURL,
			s3Bucket:       "",
			nonAWSS3Bucket: "",
			config: mapstr.M{
				"queue_url": "https://sqs.us-east-1.amazonaws.com/627959692251/test-s3-logs",
				"region":    "region",
				"endpoint":  "amazonaws.com",
			},
			expectedErr: "",
			expectedCfg: func(queueURL, s3Bucket, nonAWSS3Bucket string) config {
				c := makeConfig(queueURL, "", "")
				c.QueueURL = "https://sqs.us-east-1.amazonaws.com/627959692251/test-s3-logs"
				c.AWSConfig.Endpoint = "amazonaws.com"
				c.RegionName = "region"
				return c
			},
		},
		{
			name:           "inferred_AWS_endpoint_with_explicit_region",
			queueURL:       queueURL,
			s3Bucket:       "",
			nonAWSS3Bucket: "",
			config: mapstr.M{
				"queue_url": "https://sqs.us-east-1.amazonaws.com/627959692251/test-s3-logs",
				"region":    "region",
			},
			expectedErr: "",
			expectedCfg: func(queueURL, s3Bucket, nonAWSS3Bucket string) config {
				c := makeConfig(queueURL, "", "")
				c.QueueURL = "https://sqs.us-east-1.amazonaws.com/627959692251/test-s3-logs"
				c.RegionName = "region"
				return c
			},
		},
		{
			name:           "localstack_with_region_name",
			queueURL:       "http://localhost:4566/000000000000/sample-queue",
			s3Bucket:       "",
			nonAWSS3Bucket: "",
			config: mapstr.M{
				"queue_url": "http://localhost:4566/000000000000/sample-queue",
				"region":    "myregion",
			},
			expectedErr: "",
			expectedCfg: func(queueURL, s3Bucket, nonAWSS3Bucket string) config {
				c := makeConfig(queueURL, "", "")
				c.RegionName = "myregion"
				return c
			},
		},
		{
			name:           "error on no queueURL and s3Bucket and nonAWSS3Bucket",
			queueURL:       "",
			s3Bucket:       "",
			nonAWSS3Bucket: "",
			config: mapstr.M{
				"queue_url":           "",
				"bucket_arn":          "",
				"non_aws_bucket_name": "",
			},
			expectedErr: "neither queue_url, bucket_arn nor non_aws_bucket_name were provided",
			expectedCfg: nil,
		},
		{
			name:           "error on both queueURL and s3Bucket",
			queueURL:       queueURL,
			s3Bucket:       s3Bucket,
			nonAWSS3Bucket: "",
			config: mapstr.M{
				"queue_url":  queueURL,
				"bucket_arn": s3Bucket,
			},
			expectedErr: "queue_url <https://example.com>, bucket_arn <arn:aws:s3:::aBucket>, non_aws_bucket_name <> cannot be set at the same time",
			expectedCfg: nil,
		},
		{
			name:           "error on both queueURL and NonAWSS3Bucket",
			queueURL:       queueURL,
			s3Bucket:       "",
			nonAWSS3Bucket: nonAWSS3Bucket,
			config: mapstr.M{
				"queue_url":           queueURL,
				"non_aws_bucket_name": nonAWSS3Bucket,
			},
			expectedErr: "queue_url <https://example.com>, bucket_arn <>, non_aws_bucket_name <minio-bucket> cannot be set at the same time",
			expectedCfg: nil,
		},
		{
			name:           "error on both s3Bucket and NonAWSS3Bucket",
			queueURL:       "",
			s3Bucket:       s3Bucket,
			nonAWSS3Bucket: nonAWSS3Bucket,
			config: mapstr.M{
				"bucket_arn":          s3Bucket,
				"non_aws_bucket_name": nonAWSS3Bucket,
			},
			expectedErr: "queue_url <>, bucket_arn <arn:aws:s3:::aBucket>, non_aws_bucket_name <minio-bucket> cannot be set at the same time",
			expectedCfg: nil,
		},
		{
			name:           "error on queueURL, s3Bucket, and NonAWSS3Bucket",
			queueURL:       queueURL,
			s3Bucket:       s3Bucket,
			nonAWSS3Bucket: nonAWSS3Bucket,
			config: mapstr.M{
				"queue_url":           queueURL,
				"bucket_arn":          s3Bucket,
				"non_aws_bucket_name": nonAWSS3Bucket,
			},
			expectedErr: "queue_url <https://example.com>, bucket_arn <arn:aws:s3:::aBucket>, non_aws_bucket_name <minio-bucket> cannot be set at the same time",
			expectedCfg: nil,
		},
		{
			name:           "error on api_timeout == 0",
			queueURL:       queueURL,
			s3Bucket:       "",
			nonAWSS3Bucket: "",
			config: mapstr.M{
				"queue_url":   queueURL,
				"api_timeout": "0",
			},
			expectedErr: "api_timeout <0s> must be greater than the sqs.wait_time <20s",
			expectedCfg: nil,
		},
		{
			name:           "error on visibility_timeout == 0",
			queueURL:       queueURL,
			s3Bucket:       "",
			nonAWSS3Bucket: "",
			config: mapstr.M{
				"queue_url":          queueURL,
				"visibility_timeout": "0",
			},
			expectedErr: "visibility_timeout <0s> must be greater than 0 and less than or equal to 12h",
			expectedCfg: nil,
		},
		{
			name:           "error on visibility_timeout > 12h",
			queueURL:       queueURL,
			s3Bucket:       "",
			nonAWSS3Bucket: "",
			config: mapstr.M{
				"queue_url":          queueURL,
				"visibility_timeout": "12h1ns",
			},
			expectedErr: "visibility_timeout <12h0m0.000000001s> must be greater than 0 and less than or equal to 12h",
			expectedCfg: nil,
		},
		{
			name:           "error on bucket_list_interval == 0",
			queueURL:       "",
			s3Bucket:       s3Bucket,
			nonAWSS3Bucket: "",
			config: mapstr.M{
				"bucket_arn":           s3Bucket,
				"bucket_list_interval": "0",
			},
			expectedErr: "bucket_list_interval <0s> must be greater than 0",
			expectedCfg: nil,
		},
		{
			name:           "error on number_of_workers == 0",
			queueURL:       "",
			s3Bucket:       s3Bucket,
			nonAWSS3Bucket: "",
			config: mapstr.M{
				"bucket_arn":        s3Bucket,
				"number_of_workers": "0",
			},
			expectedErr: "number_of_workers <0> must be greater than 0",
			expectedCfg: nil,
		},
		{
			name:           "error on max_number_of_messages == 0",
			queueURL:       queueURL,
			s3Bucket:       "",
			nonAWSS3Bucket: "",
			config: mapstr.M{
				"queue_url":              queueURL,
				"max_number_of_messages": "0",
			},
			expectedErr: "max_number_of_messages <0> must be greater than 0",
			expectedCfg: nil,
		},
		{
			name:           "error on buffer_size == 0 ",
			queueURL:       queueURL,
			s3Bucket:       "",
			nonAWSS3Bucket: "",
			config: mapstr.M{
				"queue_url":   queueURL,
				"buffer_size": "0",
			},
			expectedErr: "buffer_size <0> must be greater than 0",
			expectedCfg: nil,
		},
		{
			name:           "error on max_bytes == 0 ",
			queueURL:       queueURL,
			s3Bucket:       "",
			nonAWSS3Bucket: "",
			config: mapstr.M{
				"queue_url": queueURL,
				"max_bytes": "0",
			},
			expectedErr: "max_bytes <0> must be greater than 0",
			expectedCfg: nil,
		},
		{
			name:           "error on expand_event_list_from_field and content_type != application/json ",
			queueURL:       queueURL,
			s3Bucket:       "",
			nonAWSS3Bucket: "",
			config: mapstr.M{
				"queue_url":                    queueURL,
				"expand_event_list_from_field": "Records",
				"content_type":                 "text/plain",
			},
			expectedErr: "content_type must be `application/json` when expand_event_list_from_field is used",
			expectedCfg: nil,
		},
		{
			name:           "error on expand_event_list_from_field and content_type != application/json ",
			queueURL:       "",
			s3Bucket:       s3Bucket,
			nonAWSS3Bucket: "",
			config: mapstr.M{
				"bucket_arn":                   s3Bucket,
				"expand_event_list_from_field": "Records",
				"content_type":                 "text/plain",
			},
			expectedErr: "content_type must be `application/json` when expand_event_list_from_field is used",
			expectedCfg: nil,
		},
		{
			name:           "input with defaults for non-AWS S3 Bucket",
			queueURL:       "",
			s3Bucket:       "",
			nonAWSS3Bucket: nonAWSS3Bucket,
			config: mapstr.M{
				"non_aws_bucket_name": nonAWSS3Bucket,
				"number_of_workers":   5,
			},
			expectedErr: "",
			expectedCfg: func(queueURL, s3Bucket, nonAWSS3Bucket string) config {
				c := makeConfig("", "", nonAWSS3Bucket)
				c.NumberOfWorkers = 5
				return c
			},
		},
		{
			name:           "error on FIPS with non-AWS S3 Bucket",
			queueURL:       "",
			s3Bucket:       "",
			nonAWSS3Bucket: nonAWSS3Bucket,
			config: mapstr.M{
				"non_aws_bucket_name": nonAWSS3Bucket,
				"number_of_workers":   5,
				"fips_enabled":        true,
			},
			expectedErr: "fips_enabled cannot be used with a non-AWS S3 bucket",
			expectedCfg: nil,
		},
		{
			name:           "error on path_style with AWS native S3 Bucket",
			queueURL:       "",
			s3Bucket:       s3Bucket,
			nonAWSS3Bucket: "",
			config: mapstr.M{
				"bucket_arn":        s3Bucket,
				"number_of_workers": 5,
				"path_style":        true,
			},
			expectedErr: "path_style can only be used when polling non-AWS S3 services",
			expectedCfg: nil,
		},
		{
			name:           "error on provider with AWS native S3 Bucket",
			queueURL:       "",
			s3Bucket:       s3Bucket,
			nonAWSS3Bucket: "",
			config: mapstr.M{
				"bucket_arn":        s3Bucket,
				"number_of_workers": 5,
				"provider":          "asdf",
			},
			expectedErr: "provider can only be overridden when polling non-AWS S3 services",
			expectedCfg: nil,
		},
		{
			name:           "error on provider with AWS SQS Queue",
			queueURL:       queueURL,
			s3Bucket:       "",
			nonAWSS3Bucket: "",
			config: mapstr.M{
				"queue_url":         queueURL,
				"number_of_workers": 5,
				"provider":          "asdf",
			},
			expectedErr: "provider can only be overridden when polling non-AWS S3 services",
			expectedCfg: nil,
		},
		{
			name:           "backup_to_bucket with AWS",
			queueURL:       "",
			s3Bucket:       s3Bucket,
			nonAWSS3Bucket: "",
			config: mapstr.M{
				"bucket_arn":              s3Bucket,
				"backup_to_bucket_arn":    "arn:aws:s3:::bBucket",
				"backup_to_bucket_prefix": "backup",
				"number_of_workers":       5,
			},
			expectedErr: "",
			expectedCfg: func(queueURL, s3Bucket, nonAWSS3Bucket string) config {
				c := makeConfig("", s3Bucket, "")
				c.BackupConfig.BackupToBucketArn = "arn:aws:s3:::bBucket"
				c.BackupConfig.BackupToBucketPrefix = "backup"
				c.NumberOfWorkers = 5
				return c
			},
		},
		{
			name:           "backup_to_bucket with non-AWS",
			queueURL:       "",
			s3Bucket:       "",
			nonAWSS3Bucket: nonAWSS3Bucket,
			config: mapstr.M{
				"non_aws_bucket_name":           nonAWSS3Bucket,
				"non_aws_backup_to_bucket_name": "bBucket",
				"backup_to_bucket_prefix":       "backup",
				"number_of_workers":             5,
			},
			expectedErr: "",
			expectedCfg: func(queueURL, s3Bucket, nonAWSS3Bucket string) config {
				c := makeConfig("", "", nonAWSS3Bucket)
				c.NonAWSBucketName = nonAWSS3Bucket
				c.BackupConfig.NonAWSBackupToBucketName = "bBucket"
				c.BackupConfig.BackupToBucketPrefix = "backup"
				c.NumberOfWorkers = 5
				return c
			},
		},
		{
			name:           "error with non-AWS backup and AWS source",
			queueURL:       "",
			s3Bucket:       s3Bucket,
			nonAWSS3Bucket: "",
			config: mapstr.M{
				"bucket_arn":                    s3Bucket,
				"non_aws_backup_to_bucket_name": "bBucket",
				"number_of_workers":             5,
			},
			expectedErr: "backup to non-AWS bucket can only be used for non-AWS sources",
			expectedCfg: nil,
		},
		{
			name:           "error with AWS backup and non-AWS source",
			queueURL:       "",
			s3Bucket:       "",
			nonAWSS3Bucket: nonAWSS3Bucket,
			config: mapstr.M{
				"non_aws_bucket_name":  nonAWSS3Bucket,
				"backup_to_bucket_arn": "arn:aws:s3:::bBucket",
				"number_of_workers":    5,
			},
			expectedErr: "backup to AWS bucket can only be used for AWS sources",
			expectedCfg: nil,
		},
		{
			name:           "error with same bucket backup and empty backup prefix",
			queueURL:       "",
			s3Bucket:       s3Bucket,
			nonAWSS3Bucket: "",
			config: mapstr.M{
				"bucket_arn":           s3Bucket,
				"backup_to_bucket_arn": s3Bucket,
				"number_of_workers":    5,
			},
			expectedErr: "backup_to_bucket_prefix is a required property when source and backup bucket are the same",
			expectedCfg: nil,
		},
		{
			name:           "error with same bucket backup (non-AWS) and empty backup prefix",
			queueURL:       "",
			s3Bucket:       "",
			nonAWSS3Bucket: nonAWSS3Bucket,
			config: mapstr.M{
				"non_aws_bucket_name":           nonAWSS3Bucket,
				"non_aws_backup_to_bucket_name": nonAWSS3Bucket,
				"number_of_workers":             5,
			},
			expectedErr: "backup_to_bucket_prefix is a required property when source and backup bucket are the same",
			expectedCfg: nil,
		},
		{
			name:           "error with same bucket backup and backup prefix equal to list prefix",
			queueURL:       "",
			s3Bucket:       s3Bucket,
			nonAWSS3Bucket: "",
			config: mapstr.M{
				"bucket_arn":              s3Bucket,
				"backup_to_bucket_arn":    s3Bucket,
				"number_of_workers":       5,
				"backup_to_bucket_prefix": "processed_",
				"bucket_list_prefix":      "processed_",
			},
			expectedErr: "backup_to_bucket_prefix cannot be the same as bucket_list_prefix, this will create an infinite loop",
			expectedCfg: nil,
		},
		{
			name:           "error with same bucket backup (non-AWS) and backup prefix equal to list prefix",
			queueURL:       "",
			s3Bucket:       "",
			nonAWSS3Bucket: nonAWSS3Bucket,
			config: mapstr.M{
				"non_aws_bucket_name":           nonAWSS3Bucket,
				"non_aws_backup_to_bucket_name": nonAWSS3Bucket,
				"number_of_workers":             5,
				"backup_to_bucket_prefix":       "processed_",
				"bucket_list_prefix":            "processed_",
			},
			expectedErr: "backup_to_bucket_prefix cannot be the same as bucket_list_prefix, this will create an infinite loop",
			expectedCfg: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			in := conf.MustNewConfigFrom(tc.config)

			c := defaultConfig()
			if err := in.Unpack(&c); err != nil {
				if tc.expectedErr != "" {
					assert.Contains(t, err.Error(), tc.expectedErr)
					return
				}
				t.Fatal(err)
			}

			if tc.expectedCfg == nil {
				t.Fatal("missing expected config in test case")
			}
			assert.EqualValues(t, tc.expectedCfg(tc.queueURL, tc.s3Bucket, tc.nonAWSS3Bucket), c)
		})
	}
}
