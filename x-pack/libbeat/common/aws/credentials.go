// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
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
func InitializeAWSConfig(beatsConfig ConfigAWS) (awssdk.Config, error) {
	awsConfig, _ := GetAWSCredentials(beatsConfig)
	if awsConfig.Region == "" {
		if beatsConfig.DefaultRegion != "" {
			awsConfig.Region = beatsConfig.DefaultRegion
		} else {
			awsConfig.Region = "us-east-1"
		}
	}

	// Assume IAM role if iam_role config parameter is given
	if beatsConfig.RoleArn != "" {
		addStaticCredentialsProviderToAwsConfig(beatsConfig, &awsConfig)
	}

	var proxy func(*http.Request) (*url.URL, error)
	if beatsConfig.ProxyUrl != "" {
		proxyUrl, err := httpcommon.NewProxyURIFromString(beatsConfig.ProxyUrl)
		if err != nil {
			return awsConfig, err
		}
		proxy = http.ProxyURL(proxyUrl.URI())
	}
	var tlsConfig *tls.Config
	if beatsConfig.TLS != nil {
		TLSConfig, _ := tlscommon.LoadTLSConfig(beatsConfig.TLS)
		tlsConfig = TLSConfig.ToConfig()
	}
	awsConfig.HTTPClient = &http.Client{
		Transport: &http.Transport{
			Proxy:           proxy,
			TLSClientConfig: tlsConfig,
		},
	}
	return awsConfig, nil
}

// GetAWSCredentials function gets aws credentials from the config.
// If access keys given, use them as credentials.
// If access keys are not given, then load from AWS config file. If credential_profile_name is not
// given, default profile will be used.
// If role_arn is given, assume the IAM role either with access keys or default profile.
func GetAWSCredentials(beatsConfig ConfigAWS) (awssdk.Config, error) {
	// Check if accessKeyID or secretAccessKey or sessionToken is given from configuration
	if beatsConfig.AccessKeyID != "" || beatsConfig.SecretAccessKey != "" || beatsConfig.SessionToken != "" {
		return getConfigForKeys(beatsConfig), nil
	}

	return getConfigSharedCredentialProfile(beatsConfig)
}

// getConfigForKeys creates a default AWS config and adds a CredentialsProvider using the provided Beats config.
// Provided config must contain an accessKeyID, secretAccessKey and sessionToken to generate a valid CredentialsProfile
func getConfigForKeys(beatsConfig ConfigAWS) awssdk.Config {
	logger := logp.NewLogger("getConfigForKeys")

	config := awssdk.NewConfig()
	awsCredentials := awssdk.Credentials{
		AccessKeyID:     beatsConfig.AccessKeyID,
		SecretAccessKey: beatsConfig.SecretAccessKey,
	}

	if beatsConfig.SessionToken != "" {
		awsCredentials.SessionToken = beatsConfig.SessionToken
	}

	config.Credentials = credentials.StaticCredentialsProvider{
		Value: awsCredentials,
	}

	// Assume IAM role if iam_role config parameter is given
	if beatsConfig.RoleArn != "" {
		logger.Debug("Using role arn and access keys for AWS credential")
		addStaticCredentialsProviderToAwsConfig(beatsConfig, config)
		return *config
	}

	return *config
}

// getConfigSharedCredentialProfile If accessKeyID, secretAccessKey or sessionToken is not given, iam_role is not given,
// then load from default config // Please see https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-profiles.html
//	with more details. If credential_profile_name is empty, then default profile is used.
func getConfigSharedCredentialProfile(beatsConfig ConfigAWS) (awssdk.Config, error) {
	logger := logp.NewLogger("WithSharedConfigProfile")

	var options []func(*awsConfig.LoadOptions) error
	if beatsConfig.ProfileName != "" {
		options = append(options, awsConfig.WithSharedConfigProfile(beatsConfig.ProfileName))
	}

	// If shared_credential_file is empty, then external.LoadDefaultAWSConfig
	// function will load AWS config from current user's home directory.
	// Linux/OSX: "$HOME/.aws/credentials"
	// Windows:   "%USERPROFILE%\.aws\credentials"
	if beatsConfig.SharedCredentialFile != "" {
		options = append(options, awsConfig.WithSharedConfigFiles([]string{beatsConfig.SharedCredentialFile}))
	}

	cfg, err := awsConfig.LoadDefaultConfig(context.TODO(), options...)
	if err != nil {
		return cfg, fmt.Errorf("awsConfig.LoadDefaultConfig failed with shared credential profile given: [%w]", err)
	}

	// Assume IAM role if iam_role config parameter is given
	if beatsConfig.RoleArn != "" {
		logger.Debug("Using role arn and shared credential profile for AWS credential")
		addStaticCredentialsProviderToAwsConfig(beatsConfig, &cfg)
		return cfg, nil
	}

	logger.Debug("Using shared credential profile for AWS credential")
	return cfg, nil
}

// switchToAssumeRoleProvider TODO (this function seems deprecated and replaced by addStaticCredentialsProviderToAwsConfig) switches the credentials provider in the awsConfig to the `AssumeRoleProvider`.
func switchToAssumeRoleProvider(config ConfigAWS, awsConfig awssdk.Config) awssdk.Config {
//	logger := logp.NewLogger("switchToAssumeRoleProvider")
//	logger.Debug("Switching credentials provider to AssumeRoleProvider")
//	stsSvc := sts.New(awsConfig)
//	stsCredProvider := stscreds.NewAssumeRoleProvider(stsSvc, config.RoleArn)
//	awsConfig.Credentials = stsCredProvider
	return awsConfig
}

// addStaticCredentialsProviderToAwsConfig adds a static credentials provider to the current AWS config by using the keys stored in Beats config
func addStaticCredentialsProviderToAwsConfig(beatsConfig ConfigAWS, awsConfig *awssdk.Config) {
	logger := logp.NewLogger("addStaticCredentialsProviderToAwsConfig")
	logger.Debug("Switching credentials provider to AssumeRoleProvider")
	staticCredentialsProvider := credentials.NewStaticCredentialsProvider(
		beatsConfig.AccessKeyID,
		beatsConfig.SecretAccessKey,
		beatsConfig.SessionToken)

	awsConfig.Credentials = staticCredentialsProvider

	return
}

// EnrichAWSConfigWithEndpoint function enabled endpoint resolver for AWS service clients when endpoint is given in config.
func EnrichAWSConfigWithEndpoint(endpoint string, serviceName string, regionName string, beatsConfig awssdk.Config) awssdk.Config {
	var eurl string
	if endpoint != "" {
		parsedEndpoint, _ := url.Parse(endpoint)

		// Beats uses the provided endpoint if the scheme is present or...
		if parsedEndpoint.Scheme != "" {
			eurl = endpoint
		} else {
			// ...build one by using the scheme, service and region names.
			if regionName == "" {
				eurl = "https://" + serviceName + "." + endpoint
			} else {
				eurl = "https://" + serviceName + "." + regionName + "." + endpoint
			}
		}

		beatsConfig.EndpointResolverWithOptions = awssdk.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (awssdk.Endpoint, error) {
				return awssdk.Endpoint{URL: eurl}, nil
			})
	}
	return beatsConfig
}

// CreateServiceName based on Service name, Region and FIPS. Returns service name if Fips is not enabled.
func CreateServiceName(serviceName string, fipsEnabled bool, region string) string {
	if fipsEnabled {
		_, found := OptionalGovCloudFIPS[serviceName]
		if !strings.HasPrefix(region, "us-gov-") || found {
			return serviceName + "-fips"
		}
	}
	return serviceName
}
