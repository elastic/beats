// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"os"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	"github.com/elastic/elastic-agent-libs/logp"
)

// These env vars are provided by agentless controller when the cloud connectors flow is enabled.
const (
	CloudConnectorsGlobalRoleEnvVar        = "CLOUD_CONNECTORS_GLOBAL_ROLE"
	CloudConnectorsJWTPathEnvVar           = "CLOUD_CONNECTORS_ID_TOKEN_FILE"
	CloudConnectorsCloudResourceIDEnvVar   = "CLOUD_RESOURCE_ID"
	CloudConnectorsAWSElasticResourceIDKey = "elastic_resource_id"
)

// CloudConnectorsConfig is the config for the cloud connectors flow
type CloudConnectorsConfig struct {
	ElasticGlobalRoleARN string
	IDTokenPath          string
	CloudResourceID      string
}

func parseCloudConnectorsConfigFromEnv() CloudConnectorsConfig {
	return CloudConnectorsConfig{
		ElasticGlobalRoleARN: os.Getenv(CloudConnectorsGlobalRoleEnvVar),
		IDTokenPath:          os.Getenv(CloudConnectorsJWTPathEnvVar),
		CloudResourceID:      os.Getenv(CloudConnectorsCloudResourceIDEnvVar),
	}
}

const defaultIntermediateDuration = 20 * time.Minute

func addCloudConnectorsCredentials(config ConfigAWS, cloudConnectorsConfig CloudConnectorsConfig, awsConfig *awssdk.Config, logger *logp.Logger) {
	logger = logger.Named("addCloudConnectorsCredentials")
	logger.Debug("Switching credentials provider to Cloud Connectors")

	addCredentialsChain(
		awsConfig,

		// step 1 assume Elastic Global Role with web identity using the id token provided by the agentless OIDC issuer.
		func(c awssdk.Config) awssdk.CredentialsProvider {
			provider := stscreds.NewWebIdentityRoleProvider(
				sts.NewFromConfig(c), // client uses credentials from previous config.
				cloudConnectorsConfig.ElasticGlobalRoleARN,
				stscreds.IdentityTokenFile(cloudConnectorsConfig.IDTokenPath),
				func(opt *stscreds.WebIdentityRoleOptions) {
					opt.Duration = defaultIntermediateDuration
				},
			)
			return awssdk.NewCredentialsCache(provider)
		},

		// step 2 assume the remote role (users's configured one) having the previous one in chain.
		func(c awssdk.Config) awssdk.CredentialsProvider {
			assumeRoleProvider := stscreds.NewAssumeRoleProvider(
				sts.NewFromConfig(c), // client uses credentials from previous config.
				config.RoleArn,
				func(aro *stscreds.AssumeRoleOptions) {
					aro.Duration = config.AssumeRoleDuration
					if config.ExternalID != "" {
						aro.ExternalID = awssdk.String(config.ExternalID)

						// The source identity is set by the system (env var) and not user input (package policy).
						// It should be requested on the other side (remote role) as a condition to assume.
						aro.SourceIdentity = awssdk.String(cloudConnectorsConfig.CloudResourceID)
					}
				},
			)
			return awssdk.NewCredentialsCache(assumeRoleProvider, func(options *awssdk.CredentialsCacheOptions) {
				if config.AssumeRoleExpiryWindow > 0 {
					options.ExpiryWindow = config.AssumeRoleExpiryWindow
				}
			})
		},
	)
}

func addCredentialsChain(awsConfig *awssdk.Config, chain ...func(awssdk.Config) awssdk.CredentialsProvider) {
	for _, fn := range chain {
		awsConfig.Credentials = fn(*awsConfig)
	}
}
