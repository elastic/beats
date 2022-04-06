// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
<<<<<<< HEAD
=======
	"crypto/tls"
	"fmt"
>>>>>>> cb5a95180f ([libbeat][aws] Fix AWS config initialization issue when using a role (#31014))
	"net/http"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/defaults"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/aws/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/libbeat/common/transport/httpcommon"
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
	AWSPartition         string `config:"aws_partition"` // Deprecated.
	ProxyUrl             string `config:"proxy_url"`
}

// InitializeAWSConfig function creates the awssdk.Config object from the provided config
func InitializeAWSConfig(config ConfigAWS) (awssdk.Config, error) {
	AWSConfig, _ := GetAWSCredentials(config)
<<<<<<< HEAD
=======
	if AWSConfig.Region == "" {
		if config.DefaultRegion != "" {
			AWSConfig.Region = config.DefaultRegion
		} else {
			AWSConfig.Region = "us-east-1"
		}
	}

	// Assume IAM role if iam_role config parameter is given
	if config.RoleArn != "" {
		AWSConfig = switchToAssumeRoleProvider(config, AWSConfig)
	}

	var proxy func(*http.Request) (*url.URL, error)
>>>>>>> cb5a95180f ([libbeat][aws] Fix AWS config initialization issue when using a role (#31014))
	if config.ProxyUrl != "" {
		proxyUrl, err := httpcommon.NewProxyURIFromString(config.ProxyUrl)
		if err != nil {
			return AWSConfig, err
		}

		httpClient := &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(proxyUrl.URI()),
			},
		}
		AWSConfig.HTTPClient = httpClient
	}
	return AWSConfig, nil
}

// GetAWSCredentials function gets aws credentials from the config.
// If access keys given, use them as credentials.
// If access keys are not given, then load from AWS config file. If credential_profile_name is not
// given, default profile will be used.
// If role_arn is given, assume the IAM role either with access keys or default profile.
func GetAWSCredentials(config ConfigAWS) (awssdk.Config, error) {
	// Check if accessKeyID or secretAccessKey or sessionToken is given from configuration
	if config.AccessKeyID != "" || config.SecretAccessKey != "" || config.SessionToken != "" {
		return getAccessKeys(config), nil
	}

	return getSharedCredentialProfile(config)
}

func getAccessKeys(config ConfigAWS) awssdk.Config {
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

<<<<<<< HEAD
	// Set default region if empty to make initial aws api call
	if awsConfig.Region == "" {
		awsConfig.Region = "us-east-1"
	}

	// Assume IAM role if iam_role config parameter is given
	if config.RoleArn != "" {
		logger.Debug("Using role arn and access keys for AWS credential")
		return getRoleArn(config, awsConfig)
	}

=======
>>>>>>> cb5a95180f ([libbeat][aws] Fix AWS config initialization issue when using a role (#31014))
	return awsConfig
}

func getSharedCredentialProfile(config ConfigAWS) (awssdk.Config, error) {
	// If accessKeyID, secretAccessKey or sessionToken is not given, iam_role is not given, then load from default config
	// Please see https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-profiles.html
	// with more details.
	// If credential_profile_name is empty, then default profile is used.
	logger := logp.NewLogger("getSharedCredentialProfile")
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
<<<<<<< HEAD
		return awsConfig, errors.Wrap(err, "external.LoadDefaultAWSConfig failed with shared credential profile given")
	}

	// Set default region if empty to make initial aws api call
	if awsConfig.Region == "" {
		awsConfig.Region = "us-east-1"
	}

	// Assume IAM role if iam_role config parameter is given
	if config.RoleArn != "" {
		logger.Debug("Using role arn and shared credential profile for AWS credential")
		return getRoleArn(config, awsConfig), nil
=======
		return awsConfig, fmt.Errorf("external.LoadDefaultAWSConfig failed with shared credential profile given: %w", err)
>>>>>>> cb5a95180f ([libbeat][aws] Fix AWS config initialization issue when using a role (#31014))
	}

	logger.Debug("Using shared credential profile for AWS credential")
	return awsConfig, nil
}

// switchToAssumeRoleProvider switches the credentials provider in the awsConfig to the `AssumeRoleProvider`.
func switchToAssumeRoleProvider(config ConfigAWS, awsConfig awssdk.Config) awssdk.Config {
	logger := logp.NewLogger("switchToAssumeRoleProvider")
	logger.Debug("Switching credentials provider to AssumeRoleProvider")
	stsSvc := sts.New(awsConfig)
	stsCredProvider := stscreds.NewAssumeRoleProvider(stsSvc, config.RoleArn)
	awsConfig.Credentials = stsCredProvider
	return awsConfig
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

// Validate checks for deprecated config option
func (c ConfigAWS) Validate() error {
	if c.AWSPartition != "" {
		cfgwarn.Deprecate("8.0.0", "aws_partition is deprecated. Please use endpoint instead.")
	}
	return nil
}
