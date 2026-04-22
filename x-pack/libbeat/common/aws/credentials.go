// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/transport/httpcommon"
	"github.com/elastic/elastic-agent-libs/transport/tlscommon"

	identityfederationaws "github.com/elastic/beats/v7/x-pack/libbeat/common/identityfederation/aws"
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

	// UseCloudConnectors enables the Identity Federation flow. When true,
	// InitializeAWSConfig sets up the OIDC role-chaining credentials using
	// environment variables provided by the agentless controller.
	UseCloudConnectors bool `config:"use_cloud_connectors"`
}

// InitializeAWSConfig function creates the awssdk.Config object from the provided config
func InitializeAWSConfig(beatsConfig ConfigAWS, logger *logp.Logger) (awssdk.Config, error) {
	awsConfig, _ := getAWSCredentials(beatsConfig, logger)
	if awsConfig.Region == "" {
		if beatsConfig.DefaultRegion != "" {
			awsConfig.Region = beatsConfig.DefaultRegion
		} else {
			awsConfig.Region = "us-east-1"
		}
	}

	// Assume IAM role if iam_role config parameter is given
	if beatsConfig.RoleArn != "" && !beatsConfig.UseCloudConnectors {
		addAssumeRoleProviderToAwsConfig(beatsConfig, &awsConfig, logger)
	}

	// If identity federation is enabled, set up the OIDC role-chaining credentials.
	if beatsConfig.UseCloudConnectors {
		if err := applyIdentityFederationChain(beatsConfig, &awsConfig, logger); err != nil {
			return awsConfig, err
		}
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
		TLSConfig, _ := tlscommon.LoadTLSConfig(beatsConfig.TLS, logger)
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

// getAWSCredentials function gets aws credentials from the config.
// If access keys given, use them as credentials.
// If access keys are not given, then load from AWS config file. If credential_profile_name is not
// given, default profile will be used.
// If role_arn is given, assume the IAM role either with access keys or default profile.
func getAWSCredentials(beatsConfig ConfigAWS, logger *logp.Logger) (awssdk.Config, error) {
	// Check if accessKeyID or secretAccessKey or sessionToken is given from configuration
	if beatsConfig.AccessKeyID != "" || beatsConfig.SecretAccessKey != "" || beatsConfig.SessionToken != "" {
		return getConfigForKeys(beatsConfig), nil
	}

	return getConfigSharedCredentialProfile(beatsConfig, logger)
}

// getConfigForKeys creates a default AWS config and adds a CredentialsProvider using the provided Beats config.
// Provided config must contain an accessKeyID, secretAccessKey and sessionToken to generate a valid CredentialsProfile
func getConfigForKeys(beatsConfig ConfigAWS) awssdk.Config {
	config := awssdk.NewConfig()
	config.Credentials = credentials.NewStaticCredentialsProvider(
		beatsConfig.AccessKeyID,
		beatsConfig.SecretAccessKey,
		beatsConfig.SessionToken)
	return *config
}

// getConfigSharedCredentialProfile If accessKeyID, secretAccessKey or sessionToken is not given,
// then load from default config // Please see https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-profiles.html
//
//	with more details. If credential_profile_name is empty, then default profile is used.
func getConfigSharedCredentialProfile(beatsConfig ConfigAWS, logger *logp.Logger) (awssdk.Config, error) {
	logger = logger.Named("WithSharedConfigProfile")

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

	if beatsConfig.ProfileName != "" || beatsConfig.SharedCredentialFile != "" {
		logger.Debug("Using shared credential profile for AWS credential")
	} else {
		logger.Debug("Using default config for AWS")
	}
	return cfg, nil
}

// addAssumeRoleProviderToAwsConfig adds the credentials provider to the current AWS config by using the role ARN stored in Beats config
func addAssumeRoleProviderToAwsConfig(config ConfigAWS, awsConfig *awssdk.Config, logger *logp.Logger) {
	logger = logger.Named("addAssumeRoleProviderToAwsConfig")
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

// defaultIntermediateDuration is the session duration for the intermediate Elastic Global Role.
// It is intentionally short because it is only used as a stepping stone to the customer's remote role.
const defaultIntermediateDuration = 20 * time.Minute

// applyIdentityFederationChain configures awsConfig with Identity Federation role-chaining
// credentials. It reads the three required env vars set by the agentless controller and
// builds a two-step STS chain:
//
//  1. AssumeRoleWithWebIdentity → Elastic Global Role (using the OIDC JWT from IDTokenFileEnvVar)
//  2. AssumeRole → customer's remote role (RoleArn + ExternalID from ConfigAWS)
func applyIdentityFederationChain(config ConfigAWS, awsConfig *awssdk.Config, logger *logp.Logger) error {
	logger = logger.Named("applyIdentityFederationChain")
	logger.Debug("Switching credentials provider to Identity Federation")

	globalRoleARN := os.Getenv(identityfederationaws.GlobalRoleARNEnvVar)
	idTokenPath := os.Getenv(identityfederationaws.IDTokenFileEnvVar)
	cloudResourceID := os.Getenv(identityfederationaws.CloudResourceIDEnvVar)

	var errs []error
	if globalRoleARN == "" {
		errs = append(errs, errors.New("elastic global role arn is not configured"))
	}
	if idTokenPath == "" {
		errs = append(errs, errors.New("id token path is not configured"))
	}
	if cloudResourceID == "" {
		errs = append(errs, errors.New("cloud resource id is not configured"))
	}
	if len(errs) > 0 {
		return fmt.Errorf("identity federation config is invalid: %w", errors.Join(errs...))
	}

	chain := []identityfederationaws.AWSRoleChainingStep{
		// Step 1: Assume the Elastic Global Role with web identity using the ID token
		// provided by the agentless OIDC issuer.
		&identityfederationaws.WebIdentityRoleStep{
			RoleARN:              globalRoleARN,
			WebIdentityTokenFile: idTokenPath,
			Options: func(opt *stscreds.WebIdentityRoleOptions) {
				opt.Duration = defaultIntermediateDuration
			},
		},
		// Step 2: Assume the remote role (the user's configured role), using the
		// previously assumed Elastic Global Role credentials.
		&identityfederationaws.AssumeRoleStep{
			RoleARN: config.RoleArn,
			Options: func(aro *stscreds.AssumeRoleOptions) {
				aro.Duration = config.AssumeRoleDuration
				if config.ExternalID != "" {
					aro.ExternalID = awssdk.String(identityfederationaws.FormatExternalID(cloudResourceID, config.ExternalID))
				}
			},
			CacheOptions: func(options *awssdk.CredentialsCacheOptions) {
				if config.AssumeRoleExpiryWindow > 0 {
					options.ExpiryWindow = config.AssumeRoleExpiryWindow
				}
			},
		},
	}

	result := identityfederationaws.AWSConfigRoleChaining(*awsConfig, chain)
	awsConfig.Credentials = result.Credentials
	return nil
}
