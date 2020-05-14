// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/defaults"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/aws/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/logp"
)

// ConfigAWS is a structure defined for AWS credentials
type ConfigAWS struct {
	AccessKeyID          string `config:"access_key_id"`
	SecretAccessKey      string `config:"secret_access_key"`
	SessionToken         string `config:"session_token"`
	ProfileName          string `config:"credential_profile_name"`
	SharedCredentialFile string `config:"shared_credential_file"`
	Endpoint             string `config:"endpoint"`
	RoleArn              string `config:"role_arn"`
}

// GetAWSCredentials function gets aws credentials from the config.
// If access_key_id and secret_access_key are given, then use them as credentials.
// If role_arn is given, assume the IAM role instead.
// If none of the above is given, then load from aws config file. If credential_profile_name is not
// given, then load default profile from the aws config file.
func GetAWSCredentials(config ConfigAWS) (awssdk.Config, error) {
	logger := logp.NewLogger("get_aws_credentials")

	// Check if accessKeyID or secretAccessKey or sessionToken is given from configuration
	if config.AccessKeyID != "" || config.SecretAccessKey != "" || config.SessionToken != "" {
		logger.Debug("Using access_key_id, secret_access_key and/or session_token for AWS credential")
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

	// Assume IAM role if iam_role config parameter is given
	if config.RoleArn != "" {
		logger.Debug("Using role_arn for AWS credential")
		awsConfig, err := external.LoadDefaultAWSConfig()
		if err != nil {
			return awsConfig, errors.Wrap(err, "external.LoadDefaultAWSConfig failed when using role_arn")
		}
		stsSvc := sts.New(awsConfig)
		stsCredProvider := stscreds.NewAssumeRoleProvider(stsSvc, config.RoleArn)
		awsConfig.Credentials = stsCredProvider
		return awsConfig, nil
	}

	// If accessKeyID, secretAccessKey or sessionToken is not given, iam_role is not given, then load from default config
	// Please see https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-profiles.html
	// with more details.
	// If credential_profile_name is empty, then default profile is used.
	logger.Debug("Using shared credential profile for AWS credential")
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
		return awsConfig, errors.Wrap(err, "external.LoadDefaultAWSConfig failed when using shared credential profile")
	}
	return awsConfig, nil
}

// EnrichAWSConfigWithEndpoint function enabled endpoint resolver for AWS
// service clients when endpoint is given in config.
func EnrichAWSConfigWithEndpoint(endpoint string, serviceName string, regionName string, awsConfig awssdk.Config) awssdk.Config {
	if endpoint != "" {
		if regionName == "" {
			awsConfig.EndpointResolver = awssdk.ResolveWithEndpointURL("https://" + serviceName + "." + endpoint)
		} else {
			awsConfig.EndpointResolver = awssdk.ResolveWithEndpointURL("https://" + serviceName + "." + regionName + "." + endpoint)
		}
	}
	return awsConfig
}
