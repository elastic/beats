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
	AccessKeyID          string `config:"access_key_id"`
	SecretAccessKey      string `config:"secret_access_key"`
	SessionToken         string `config:"session_token"`
	ProfileName          string `config:"credential_profile_name"`
	SharedCredentialFile string `config:"shared_credential_file"`
	Endpoint             string `config:"endpoint"`
	EndpointRegion       string `config:"endpoint_region"`
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

		// check if endpoint is given from configuration
		if config.Endpoint != "" {
			awsConfig.EndpointResolver = awssdk.ResolveWithEndpointURL(config.Endpoint)
		}
		return awsConfig, nil
	}

	// If accessKeyID, secretAccessKey or sessionToken is not given, then load from default config
	// Please see https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-profiles.html
	// with more details.
	// If credential_profile_name is empty, then default profile is used.
	var options []external.Config
	if config.ProfileName != "" {
		options = append(options, external.WithSharedConfigProfile(config.ProfileName))
	}
	// If shared_credential_file is empty, then external.LoadDefaultAWSConfig
	// function will load AWS config from current user's home directory.
	// Linux/OSX: "$HOME/.aws/credentials"
	// Windows:   "%USERPROFILE%\.aws\credentials"
	if config.SharedCredentialFile != "" {
		options = append(options, external.WithSharedConfigFiles([]string{config.SharedCredentialFile}))
	}

	awsConfig, err := external.LoadDefaultAWSConfig(options...)
	if err != nil {
		return awsConfig, err
	}

	// check if endpoint is given from configuration
	if config.Endpoint != "" {
		awsConfig.EndpointResolver = awssdk.ResolveWithEndpointURL(config.Endpoint)
	}
	return awsConfig, nil
}
