// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/defaults"
	"github.com/aws/aws-sdk-go-v2/aws/external"
)

// ConfigAWS is a structure defined for AWS credentials
type ConfigAWS struct {
	AccessKeyID               string   `config:"access_key_id"`
	SecretAccessKey           string   `config:"secret_access_key"`
	SessionToken              string   `config:"session_token"`
	ProfileName               string   `config:"credential_profile_name"`
}

// GetAWSCredentials function gets aws credentials from the config.
// If access_key_id and secret_access_key are given, then use them as credentials.
// If not, then load from aws config file. If credential_profile_name is not
// given, then load default profile from the aws config file.
func GetAWSCredentials(config ConfigAWS) (awssdk.Config, error) {
	// Check if accessKeyID or secretAccessKey or sessionToken is given from configuration
	if config.AccessKeyID != "" || config.SecretAccessKey != "" || config.SessionToken != "" {
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
		return awsConfig, nil
	}

	// If accessKeyID, secretAccessKey or sessionToken is not given, then load from default config
	// Please see https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-profiles.html
	// with more details.
	if config.ProfileName != "" {
		return external.LoadDefaultAWSConfig(
			external.WithSharedConfigProfile(config.ProfileName),
		)
	}
	return external.LoadDefaultAWSConfig()
}
