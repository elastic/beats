// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/defaults"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/aws/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	"github.com/elastic/beats/v7/libbeat/common/transport/httpcommon"
	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/v7/libbeat/logp"
)

// OptionalGovCloudFIPS is a list of services on AWS GovCloud that is not FIPS by default.
// These services follow the standard <service name>-fips.<region>.amazonaws.com format.
var OptionalGovCloudFIPS = map[string]bool{
	"s3": true,
}

// ConfigAWS is a structure defined for AWS credentials
type ConfigAWS struct {
	AccessKeyID          string            `config:"access_key_id"`
	SecretAccessKey      string            `config:"secret_access_key"`
	SessionToken         string            `config:"session_token"`
	ProfileName          string            `config:"credential_profile_name"`
	SharedCredentialFile string            `config:"shared_credential_file"`
	Endpoint             string            `config:"endpoint"`
	RoleArn              string            `config:"role_arn"`
	ProxyUrl             string            `config:"proxy_url"`
	FIPSEnabled          bool              `config:"fips_enabled"`
	TLS                  *tlscommon.Config `config:"ssl" yaml:"ssl,omitempty" json:"ssl,omitempty"`
	DefaultRegion        string            `config:"default_region"`
}

// InitializeAWSConfig function creates the awssdk.Config object from the provided config
func InitializeAWSConfig(config ConfigAWS) (awssdk.Config, error) {
	AWSConfig, _ := GetAWSCredentials(config)
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
	if config.ProxyUrl != "" {
		proxyUrl, err := httpcommon.NewProxyURIFromString(config.ProxyUrl)
		if err != nil {
			return AWSConfig, err
		}
		proxy = http.ProxyURL(proxyUrl.URI())
	}
	var tlsConfig *tls.Config
	if config.TLS != nil {
		TLSConfig, _ := tlscommon.LoadTLSConfig(config.TLS)
		tlsConfig = TLSConfig.ToConfig()
	}
	AWSConfig.HTTPClient = &http.Client{
		Transport: &http.Transport{
			Proxy:           proxy,
			TLSClientConfig: tlsConfig,
		},
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
		return awsConfig, fmt.Errorf("external.LoadDefaultAWSConfig failed with shared credential profile given: %w", err)
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
	var eurl string
	if endpoint != "" {
		parsedEndpoint, _ := url.Parse(endpoint)
		if parsedEndpoint.Scheme != "" {
			awsConfig.EndpointResolver = awssdk.ResolveWithEndpointURL(endpoint)
		} else {
			if regionName == "" {
				eurl = "https://" + serviceName + "." + endpoint
			} else {
				eurl = "https://" + serviceName + "." + regionName + "." + endpoint
			}
			awsConfig.EndpointResolver = awssdk.ResolveWithEndpointURL(eurl)
		}
	}
	return awsConfig
}

//Create AWS service name based on Region and FIPS
func CreateServiceName(serviceName string, fipsEnabled bool, region string) string {
	if fipsEnabled {
		_, found := OptionalGovCloudFIPS[serviceName]
		if !strings.HasPrefix(region, "us-gov-") || found {
			return serviceName + "-fips"
		}
	}
	return serviceName
}
