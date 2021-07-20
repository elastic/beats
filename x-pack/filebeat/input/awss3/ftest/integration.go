// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package ftest

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/common"

	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
)

const (
	visibilityTimeout = 300 * time.Second
)

// GetConfigForTestSQSCollector function gets aws credentials for integration tests.
func GetConfigForTestSQSCollector(t *testing.T) *common.Config {
	t.Helper()

	awsConfig := awscommon.ConfigAWS{}
	queueURL := os.Getenv("QUEUE_URL")
	profileName := os.Getenv("AWS_PROFILE_NAME")
	accessKeyID := os.Getenv("AWS_ACCESS_KEY_ID")
	secretAccessKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	sessionToken := os.Getenv("AWS_SESSION_TOKEN")

	config := common.MapStr{
		"VisibilityTimeout": visibilityTimeout,
	}
	switch {
	case queueURL == "":
		t.Fatal("$QUEUE_URL is not set in environment")
	case profileName == "" && accessKeyID == "":
		t.Fatal("$AWS_ACCESS_KEY_ID or $AWS_PROFILE_NAME not set or set to empty")
	case profileName != "":
		awsConfig.ProfileName = profileName
		config["queue_url"] = queueURL

		updateConfigFromAwsConfig(t, config, awsConfig)

		return common.MustNewConfigFrom(config)
	case secretAccessKey == "":
		t.Fatal("$AWS_SECRET_ACCESS_KEY not set or set to empty")
	}

	awsConfig.AccessKeyID = accessKeyID
	awsConfig.SecretAccessKey = secretAccessKey
	if sessionToken != "" {
		awsConfig.SessionToken = sessionToken
	}
	config["aws_config"] = awsConfig
	updateConfigFromAwsConfig(t, config, awsConfig)

	return common.MustNewConfigFrom(config)
}

// GetConfigForTestS3BucketCollector function gets aws credentials for integration tests.
func GetConfigForTestS3BucketCollector(t *testing.T) *common.Config {
	t.Helper()

	awsConfig := awscommon.ConfigAWS{}
	s3Bucket := os.Getenv("S3_BUCKET_NAME")
	profileName := os.Getenv("AWS_PROFILE_NAME")
	accessKeyID := os.Getenv("AWS_ACCESS_KEY_ID")
	secretAccessKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	sessionToken := os.Getenv("AWS_SESSION_TOKEN")

	config := common.MapStr{
		"visibility_timeout": visibilityTimeout,
	}
	switch {
	case s3Bucket == "":
		t.Fatal("S3_BUCKET_NAME is not set in environment")
	case profileName == "" && accessKeyID == "":
		t.Fatal("$AWS_ACCESS_KEY_ID or $AWS_PROFILE_NAME not set or set to empty")
	case profileName != "":
		awsConfig.ProfileName = profileName
		config["s3_bucket"] = s3Bucket

		updateConfigFromAwsConfig(t, config, awsConfig)

		return common.MustNewConfigFrom(config)
	case secretAccessKey == "":
		t.Fatal("$AWS_SECRET_ACCESS_KEY not set or set to empty")
	}

	awsConfig.AccessKeyID = accessKeyID
	awsConfig.SecretAccessKey = secretAccessKey
	if sessionToken != "" {
		awsConfig.SessionToken = sessionToken
	}
	config["aws_config"] = awsConfig
	updateConfigFromAwsConfig(t, config, awsConfig)

	return common.MustNewConfigFrom(config)
}

func updateConfigFromAwsConfig(t *testing.T, config common.MapStr, awsConfig awscommon.ConfigAWS) {
	awsConfigJson, err := json.Marshal(awsConfig)
	if err != nil {
		t.Fatal("cannot generate config", err)
	}
	var awsConfigMapStr common.MapStr
	err = json.Unmarshal(awsConfigJson, &awsConfigMapStr)
	if err != nil {
		t.Fatal("cannot generate config", err)
	}

	config.Update(awsConfigMapStr)

}
