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
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sts"

	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/transport/httpcommon"
	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
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
	ExternalID           string            `config:"external_id"`
	ProxyUrl             string            `config:"proxy_url"`
	FIPSEnabled          bool              `config:"fips_enabled"`
	TLS                  *tlscommon.Config `config:"ssl" yaml:"ssl,omitempty" json:"ssl,omitempty"`
	DefaultRegion        string            `config:"default_region"`

	// The duration of the role session. Defaults to 15m when not set.
	AssumeRoleDuration time.Duration `config:"assume_role.duration"`

	// AssumeRoleExpiryWindow will allow the credentials to trigger refreshing prior to the credentials
	// actually expiring. If expiry_window is less than or equal to zero, the setting is ignored.
	AssumeRoleExpiryWindow time.Duration `config:"assume_role.expiry_window"`
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
		addAssumeRoleProviderToAwsConfig(beatsConfig, &awsConfig)
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
	config := awssdk.NewConfig()
	awsCredentials := awssdk.Credentials{
		AccessKeyID:     beatsConfig.AccessKeyID,
		SecretAccessKey: beatsConfig.SecretAccessKey,
	}

	if beatsConfig.SessionToken != "" {
		awsCredentials.SessionToken = beatsConfig.SessionToken
	}

	addStaticCredentialsProviderToAwsConfig(beatsConfig, config)

	return *config
}

// getConfigSharedCredentialProfile If accessKeyID, secretAccessKey or sessionToken is not given,
// then load from default config // Please see https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-profiles.html
//
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

	logger.Debug("Using shared credential profile for AWS credential")
	return cfg, nil
}

// addAssumeRoleProviderToAwsConfig adds the credentials provider to the current AWS config by using the role ARN stored in Beats config
func addAssumeRoleProviderToAwsConfig(config ConfigAWS, awsConfig *awssdk.Config) {
	logger := logp.NewLogger("addAssumeRoleProviderToAwsConfig")
	logger.Debug("Switching credentials provider to AssumeRoleProvider")
	stsSvc := sts.NewFromConfig(*awsConfig)
	stsCredProvider := stscreds.NewAssumeRoleProvider(stsSvc, config.RoleArn, func(aro *stscreds.AssumeRoleOptions) {
		if config.ExternalID != "" {
			aro.ExternalID = awssdk.String(config.ExternalID)
		}
		if config.AssumeRoleDuration > 0 {
			aro.Duration = config.AssumeRoleDuration
		}
	})
	awsConfig.Credentials = awssdk.NewCredentialsCache(stsCredProvider, func(options *awssdk.CredentialsCacheOptions) {
		if config.AssumeRoleExpiryWindow > 0 {
			options.ExpiryWindow = config.AssumeRoleExpiryWindow
		}
	})
}

// addStaticCredentialsProviderToAwsConfig adds a static credentials provider to the current AWS config by using the keys stored in Beats config
func addStaticCredentialsProviderToAwsConfig(beatsConfig ConfigAWS, awsConfig *awssdk.Config) {
	logger := logp.NewLogger("addStaticCredentialsProviderToAwsConfig")
	logger.Debug("Switching credentials provider to StaticCredentialsProvider")
	staticCredentialsProvider := credentials.NewStaticCredentialsProvider(
		beatsConfig.AccessKeyID,
		beatsConfig.SecretAccessKey,
		beatsConfig.SessionToken)

	awsConfig.Credentials = staticCredentialsProvider
}
