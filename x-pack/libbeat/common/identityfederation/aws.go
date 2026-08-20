// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Package identityfederation provides AWS, GCP, and Azure Identity Federation
// credential helpers shared across elastic/beats, elastic/elastic-agent, and
// elastic/cloudbeat.
//
// AWS symbols are prefixed with AWS, GCP symbols with GCP, and Azure symbols
// with Azure, so callers use identityfederation.AWSxxx, identityfederation.GCPxxx,
// identityfederation.Azurexxx.
package identityfederation

import (
	"context"
	"fmt"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// Environment variables set by the agentless-controller for Identity Federation.
const (
	AWSGlobalRoleARNEnvVar   = "CLOUD_CONNECTORS_GLOBAL_ROLE"
	AWSIDTokenFileEnvVar     = "CLOUD_CONNECTORS_ID_TOKEN_FILE"
	AWSCloudResourceIDEnvVar = "CLOUD_RESOURCE_ID"
	// AWSIRSATokenFileEnvVar is the standard AWS SDK env var for IRSA (IAM Roles
	// for Service Accounts) on EKS. When set, the pod authenticates via its
	// Kubernetes service-account token rather than an OIDC JWT from the agentless controller.
	AWSIRSATokenFileEnvVar = "AWS_WEB_IDENTITY_TOKEN_FILE" //nolint:gosec // G101 false positive: this is an env var name, not a credential
)

const defaultIntermediateDuration = 20 * time.Minute

// AWSFormatExternalID formats the ExternalID for AssumeRole calls in the Identity
// Federation flow. The format "resourceID-externalIDPart" is required by the
// Elastic trust policy to prevent confused-deputy attacks.
func AWSFormatExternalID(resourceID, externalIDPart string) string {
	return fmt.Sprintf("%s-%s", resourceID, externalIDPart)
}

// ──────────────────────────────────────────────
// Role chaining primitives
// ──────────────────────────────────────────────

// AWSConfigRoleChaining chains multiple role assumptions. At each step the STS
// client is constructed from the current config so that each step's credentials
// are derived from the previous step's assumed role.
func AWSConfigRoleChaining(initialConfig awssdk.Config, chain []AWSRoleChainingStep) *awssdk.Config {
	cnf := initialConfig
	for _, step := range chain {
		client := sts.NewFromConfig(cnf)
		cnf.Credentials = step.BuildCredentialsCache(client)
	}
	return &cnf
}

// AWSRoleChainingStep is a single step in an AWS role assumption chain.
type AWSRoleChainingStep interface {
	BuildCredentialsCache(client *sts.Client) *awssdk.CredentialsCache
}

// AWSAssumeRoleStep assumes a role using standard sts:AssumeRole.
type AWSAssumeRoleStep struct {
	RoleARN string
	Options func(*stscreds.AssumeRoleOptions)
	// CacheOptions, when non-nil, is applied to the CredentialsCache for this
	// step. Use this to set ExpiryWindow or other cache-level options.
	CacheOptions func(*awssdk.CredentialsCacheOptions)
}

// BuildCredentialsCache implements AWSRoleChainingStep.
func (s *AWSAssumeRoleStep) BuildCredentialsCache(client *sts.Client) *awssdk.CredentialsCache {
	provider := stscreds.NewAssumeRoleProvider(client, s.RoleARN, s.Options)
	if s.CacheOptions != nil {
		return awssdk.NewCredentialsCache(provider, s.CacheOptions)
	}
	return awssdk.NewCredentialsCache(provider)
}

// AWSWebIdentityRoleStep assumes a role using sts:AssumeRoleWithWebIdentity (OIDC/JWT).
type AWSWebIdentityRoleStep struct {
	RoleARN              string
	WebIdentityTokenFile string
	Options              func(*stscreds.WebIdentityRoleOptions)
}

// BuildCredentialsCache implements AWSRoleChainingStep.
func (s *AWSWebIdentityRoleStep) BuildCredentialsCache(client *sts.Client) *awssdk.CredentialsCache {
	return AWSNewWebIdentityCredentialsCache(client, s.RoleARN, s.WebIdentityTokenFile, s.Options)
}

// AWSNewWebIdentityCredentialsCache creates a credentials cache for JWT/OIDC-based
// authentication. The token file is re-read on each credential refresh.
func AWSNewWebIdentityCredentialsCache(
	client *sts.Client,
	roleARN string,
	tokenFilePath string,
	options func(*stscreds.WebIdentityRoleOptions),
) *awssdk.CredentialsCache {
	return awssdk.NewCredentialsCache(stscreds.NewWebIdentityRoleProvider(
		client,
		roleARN,
		stscreds.IdentityTokenFile(tokenFilePath),
		options,
	))
}

// Compile-time interface checks.
var (
	_ AWSRoleChainingStep = (*AWSAssumeRoleStep)(nil)
	_ AWSRoleChainingStep = (*AWSWebIdentityRoleStep)(nil)
)

// ──────────────────────────────────────────────
// High-level chain constructors
// ──────────────────────────────────────────────

// AWSIRSAChainConfig configures the 3-step IRSA role assumption chain.
//
// Chain: IRSA (implicit via LoadDefaultConfig) → GlobalRoleARN → RemoteRoleARN
type AWSIRSAChainConfig struct {
	// GlobalRoleARN is the Elastic-owned intermediary role ARN.
	GlobalRoleARN string
	// RemoteRoleARN is the customer's target role ARN.
	RemoteRoleARN string
	// ResourceID is the cloud resource identifier (CLOUD_RESOURCE_ID env var).
	ResourceID string
	// ExternalID is combined with ResourceID to form the full ExternalID
	// on the remote role assumption: AWSFormatExternalID(ResourceID, ExternalID).
	ExternalID string
	// Region sets the AWS region. Defaults to "us-east-1".
	Region string
	// AssumeRoleDuration is the duration for the remote role session.
	AssumeRoleDuration time.Duration
}

// AWSNewIRSAChain creates an AWS config using IRSA with role chaining.
//
// Chain:
//  1. LoadDefaultConfig – picks up IRSA credentials via AWS_WEB_IDENTITY_TOKEN_FILE
//  2. Assume GlobalRoleARN (20-minute intermediate session)
//  3. Assume RemoteRoleARN with ExternalID = AWSFormatExternalID(ResourceID, ExternalID)
func AWSNewIRSAChain(ctx context.Context, cfg AWSIRSAChainConfig) (*awssdk.Config, error) {
	region := cfg.Region
	if region == "" {
		region = "us-east-1"
	}

	// AWS enforces a hard 1-hour maximum on DurationSeconds when AssumeRole is
	// called using credentials from another assumed role (role chaining).
	if cfg.AssumeRoleDuration > time.Hour {
		return nil, fmt.Errorf("assume role duration cannot exceed 1h for identity federation role chaining")
	}

	baseCfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading default AWS config for IRSA chain: %w", err)
	}
	if baseCfg.Region == "" {
		baseCfg.Region = region
	}

	chain := []AWSRoleChainingStep{
		&AWSAssumeRoleStep{
			RoleARN: cfg.GlobalRoleARN,
			Options: func(aro *stscreds.AssumeRoleOptions) {
				aro.Duration = defaultIntermediateDuration
			},
		},
		&AWSAssumeRoleStep{
			RoleARN: cfg.RemoteRoleARN,
			Options: func(aro *stscreds.AssumeRoleOptions) {
				if cfg.AssumeRoleDuration > 0 {
					aro.Duration = cfg.AssumeRoleDuration
				}
				if cfg.ResourceID != "" && cfg.ExternalID != "" {
					aro.ExternalID = awssdk.String(AWSFormatExternalID(cfg.ResourceID, cfg.ExternalID))
				}
			},
		},
	}

	return AWSConfigRoleChaining(baseCfg, chain), nil
}

// AWSOIDCChainConfig configures the 2-step OIDC/WebIdentity role assumption chain.
//
// Chain: JWT → GlobalRoleARN → RemoteRoleARN
type AWSOIDCChainConfig struct {
	// JWTFilePath is the path to the OIDC identity token file.
	JWTFilePath string
	// GlobalRoleARN is the Elastic-owned intermediary role ARN.
	GlobalRoleARN string
	// RemoteRoleARN is the customer's target role ARN.
	RemoteRoleARN string
	// ResourceID is the cloud resource identifier (CLOUD_RESOURCE_ID env var).
	ResourceID string
	// ExternalID is combined with ResourceID to form the full ExternalID.
	ExternalID string
	// Region sets the AWS region. Defaults to "us-east-1".
	Region string
	// AssumeRoleDuration is the duration for the remote role session.
	AssumeRoleDuration time.Duration
}

// AWSNewOIDCChain creates an AWS config using OIDC/WebIdentity token-based
// authentication with role chaining.
//
// Chain:
//  1. AssumeRoleWithWebIdentity using JWTFilePath → GlobalRoleARN (20-minute intermediate session)
//  2. Assume RemoteRoleARN with ExternalID = AWSFormatExternalID(ResourceID, ExternalID)
func AWSNewOIDCChain(ctx context.Context, cfg AWSOIDCChainConfig) (*awssdk.Config, error) {
	region := cfg.Region
	if region == "" {
		region = "us-east-1"
	}

	// AWS enforces a hard 1-hour maximum on DurationSeconds when AssumeRole is
	// called using credentials from another assumed role (role chaining).
	if cfg.AssumeRoleDuration > time.Hour {
		return nil, fmt.Errorf("assume role duration cannot exceed 1h for identity federation role chaining")
	}

	baseCfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading default AWS config for OIDC chain: %w", err)
	}
	if baseCfg.Region == "" {
		baseCfg.Region = region
	}

	chain := []AWSRoleChainingStep{
		&AWSWebIdentityRoleStep{
			RoleARN:              cfg.GlobalRoleARN,
			WebIdentityTokenFile: cfg.JWTFilePath,
			Options: func(o *stscreds.WebIdentityRoleOptions) {
				o.Duration = defaultIntermediateDuration
			},
		},
		&AWSAssumeRoleStep{
			RoleARN: cfg.RemoteRoleARN,
			Options: func(aro *stscreds.AssumeRoleOptions) {
				if cfg.AssumeRoleDuration > 0 {
					aro.Duration = cfg.AssumeRoleDuration
				}
				if cfg.ResourceID != "" && cfg.ExternalID != "" {
					aro.ExternalID = awssdk.String(AWSFormatExternalID(cfg.ResourceID, cfg.ExternalID))
				}
			},
		},
	}

	return AWSConfigRoleChaining(baseCfg, chain), nil
}
