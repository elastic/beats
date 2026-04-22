// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Package aws provides AWS Identity Federation credential helpers shared across
// elastic/beats, elastic/elastic-agent, and elastic/cloudbeat.
//
// It implements the role assumption chains used in the Cloud Connectors
// Identity Federation flow:
//   - IRSA chain:  IRSA → GlobalRoleARN → RemoteRoleARN
//   - OIDC chain:  JWT  → GlobalRoleARN → RemoteRoleARN
//
// Environment variable names intentionally retain the CLOUD_CONNECTORS_ prefix
// because the agentless-controller populates them under those names.
package aws

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
	GlobalRoleARNEnvVar   = "CLOUD_CONNECTORS_GLOBAL_ROLE"
	IDTokenFileEnvVar     = "CLOUD_CONNECTORS_ID_TOKEN_FILE"
	CloudResourceIDEnvVar = "CLOUD_RESOURCE_ID"
)

const defaultIntermediateDuration = 20 * time.Minute

// FormatExternalID formats the ExternalID for AssumeRole calls in the Identity
// Federation flow. The format "resourceID-externalIDPart" is required by the
// Elastic trust policy to prevent confused-deputy attacks.
func FormatExternalID(resourceID, externalIDPart string) string {
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

// AssumeRoleStep assumes a role using standard sts:AssumeRole.
type AssumeRoleStep struct {
	RoleARN string
	Options func(*stscreds.AssumeRoleOptions)
	// CacheOptions, when non-nil, is applied to the CredentialsCache for this
	// step. Use this to set ExpiryWindow or other cache-level options.
	CacheOptions func(*awssdk.CredentialsCacheOptions)
}

// BuildCredentialsCache implements AWSRoleChainingStep.
func (s *AssumeRoleStep) BuildCredentialsCache(client *sts.Client) *awssdk.CredentialsCache {
	provider := stscreds.NewAssumeRoleProvider(client, s.RoleARN, s.Options)
	if s.CacheOptions != nil {
		return awssdk.NewCredentialsCache(provider, s.CacheOptions)
	}
	return awssdk.NewCredentialsCache(provider)
}

// WebIdentityRoleStep assumes a role using sts:AssumeRoleWithWebIdentity (OIDC/JWT).
type WebIdentityRoleStep struct {
	RoleARN              string
	WebIdentityTokenFile string
	Options              func(*stscreds.WebIdentityRoleOptions)
}

// BuildCredentialsCache implements AWSRoleChainingStep.
func (s *WebIdentityRoleStep) BuildCredentialsCache(client *sts.Client) *awssdk.CredentialsCache {
	return NewWebIdentityCredentialsCache(client, s.RoleARN, s.WebIdentityTokenFile, s.Options)
}

// NewWebIdentityCredentialsCache creates a credentials cache for JWT/OIDC-based
// authentication. The token file is re-read on each credential refresh.
func NewWebIdentityCredentialsCache(
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
	_ AWSRoleChainingStep = (*AssumeRoleStep)(nil)
	_ AWSRoleChainingStep = (*WebIdentityRoleStep)(nil)
)

// ──────────────────────────────────────────────
// High-level chain constructors
// ──────────────────────────────────────────────

// IRSAChainConfig configures the 3-step IRSA role assumption chain.
//
// Chain: IRSA (implicit via LoadDefaultConfig) → GlobalRoleARN → RemoteRoleARN
type IRSAChainConfig struct {
	// GlobalRoleARN is the Elastic-owned intermediary role ARN.
	GlobalRoleARN string
	// RemoteRoleARN is the customer's target role ARN.
	RemoteRoleARN string
	// ResourceID is the cloud resource identifier (CLOUD_RESOURCE_ID env var).
	ResourceID string
	// ExternalID is combined with ResourceID to form the full ExternalID
	// on the remote role assumption: FormatExternalID(ResourceID, ExternalID).
	ExternalID string
	// Region sets the AWS region. Defaults to "us-east-1".
	Region string
	// AssumeRoleDuration is the duration for the remote role session.
	AssumeRoleDuration time.Duration
}

// NewIRSAChain creates an AWS config using IRSA with role chaining.
//
// Chain:
//  1. LoadDefaultConfig – picks up IRSA credentials via AWS_WEB_IDENTITY_TOKEN_FILE
//  2. Assume GlobalRoleARN (20-minute intermediate session)
//  3. Assume RemoteRoleARN with ExternalID = FormatExternalID(ResourceID, ExternalID)
func NewIRSAChain(ctx context.Context, cfg IRSAChainConfig) (*awssdk.Config, error) {
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
		&AssumeRoleStep{
			RoleARN: cfg.GlobalRoleARN,
			Options: func(aro *stscreds.AssumeRoleOptions) {
				aro.Duration = defaultIntermediateDuration
			},
		},
		&AssumeRoleStep{
			RoleARN: cfg.RemoteRoleARN,
			Options: func(aro *stscreds.AssumeRoleOptions) {
				if cfg.AssumeRoleDuration > 0 {
					aro.Duration = cfg.AssumeRoleDuration
				}
				if cfg.ResourceID != "" && cfg.ExternalID != "" {
					aro.ExternalID = awssdk.String(FormatExternalID(cfg.ResourceID, cfg.ExternalID))
				}
			},
		},
	}

	return AWSConfigRoleChaining(baseCfg, chain), nil
}

// OIDCChainConfig configures the 2-step OIDC/WebIdentity role assumption chain.
//
// Chain: JWT → GlobalRoleARN → RemoteRoleARN
type OIDCChainConfig struct {
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

// NewOIDCChain creates an AWS config using OIDC/WebIdentity token-based
// authentication with role chaining.
//
// Chain:
//  1. AssumeRoleWithWebIdentity using JWTFilePath → GlobalRoleARN (20-minute intermediate session)
//  2. Assume RemoteRoleARN with ExternalID = FormatExternalID(ResourceID, ExternalID)
func NewOIDCChain(ctx context.Context, cfg OIDCChainConfig) (*awssdk.Config, error) {
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
		&WebIdentityRoleStep{
			RoleARN:              cfg.GlobalRoleARN,
			WebIdentityTokenFile: cfg.JWTFilePath,
			Options: func(o *stscreds.WebIdentityRoleOptions) {
				o.Duration = defaultIntermediateDuration
			},
		},
		&AssumeRoleStep{
			RoleARN: cfg.RemoteRoleARN,
			Options: func(aro *stscreds.AssumeRoleOptions) {
				if cfg.AssumeRoleDuration > 0 {
					aro.Duration = cfg.AssumeRoleDuration
				}
				if cfg.ResourceID != "" && cfg.ExternalID != "" {
					aro.ExternalID = awssdk.String(FormatExternalID(cfg.ResourceID, cfg.ExternalID))
				}
			},
		},
	}

	return AWSConfigRoleChaining(baseCfg, chain), nil
}
