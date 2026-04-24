// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Package gcp provides GCP Workload Identity Federation authentication for the
// Identity Federation flow. It bridges AWS OIDC credentials to GCP service account
// impersonation via the GCP external account token exchange.
package gcp

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google/externalaccount"

	libbeatidaws "github.com/elastic/beats/v7/x-pack/libbeat/common/identityfederation/aws"
)

const (
	gcpSTSTokenURL        = "https://sts.googleapis.com/v1/token"        //nolint:gosec // not a credential, it's a public API endpoint
	gcpIAMCredentialsURL  = "https://iamcredentials.googleapis.com/v1/projects/-/serviceAccounts/" //nolint:gosec // not a credential, it's a public API endpoint
	awsTokenType          = "urn:ietf:params:aws:token-type:aws4_request" //nolint:gosec // not a credential, it's an IETF token type identifier
	gcpCloudPlatformScope = "https://www.googleapis.com/auth/cloud-platform"
	defaultAWSRegion      = "us-east-1"
)

// Params configures the AWS-mediated GCP Workload Identity Federation flow.
//
// Authentication chain:
//  1. Read JWT from JWTFilePath
//  2. AssumeRoleWithWebIdentity → GlobalRoleARN (using JWT)
//  3. Supply AWS credentials to GCP STS for WIF token exchange
//  4. Impersonate ServiceAccountEmail in the customer's GCP project (when set)
type Params struct {
	// Audience is the Workload Identity Federation audience URL.
	Audience string
	// GlobalRoleARN is the Elastic-owned AWS role ARN assumed via WebIdentity.
	GlobalRoleARN string
	// JWTFilePath is the path to the OIDC identity token file.
	JWTFilePath string
	// SessionName is used as the RoleSessionName for the WebIdentity call.
	// Recommended format: "resourceID-identityFederationID".
	SessionName string
	// ServiceAccountEmail is the GCP service account to impersonate.
	// If empty, service account impersonation is skipped.
	ServiceAccountEmail string
	// AWSRegion sets the AWS region for STS calls. Defaults to "us-east-1".
	AWSRegion string
	// HTTPClient is an optional HTTP client for both AWS STS and GCP token exchange.
	// Set this when FIPS-compliant TLS is required.
	HTTPClient *http.Client
}

func (p Params) validate() error {
	var errs []error
	if p.Audience == "" {
		errs = append(errs, errors.New("audience is required"))
	}
	if p.GlobalRoleARN == "" {
		errs = append(errs, errors.New("global role ARN is required"))
	}
	if p.JWTFilePath == "" {
		errs = append(errs, errors.New("JWT file path is required"))
	}
	if len(errs) > 0 {
		return fmt.Errorf("invalid GCP identity federation params: %w", errors.Join(errs...))
	}
	return nil
}

// NewTokenSource creates an OAuth2 token source for GCP using AWS-mediated Workload
// Identity Federation. The returned token source automatically refreshes credentials.
//
// If params.HTTPClient is set it is injected into both the AWS STS call and the
// GCP token exchange context, enabling FIPS-compliant TLS throughout the chain.
func NewTokenSource(ctx context.Context, params Params) (oauth2.TokenSource, error) {
	if err := params.validate(); err != nil {
		return nil, err
	}

	region := params.AWSRegion
	if region == "" {
		region = defaultAWSRegion
	}

	stsOpts := sts.Options{Region: region}
	if params.HTTPClient != nil {
		stsOpts.HTTPClient = params.HTTPClient
	}
	stsClient := sts.New(stsOpts)

	sessionName := params.SessionName
	credsCache := libbeatidaws.NewWebIdentityCredentialsCache(
		stsClient,
		params.GlobalRoleARN,
		params.JWTFilePath,
		func(o *stscreds.WebIdentityRoleOptions) {
			if sessionName != "" {
				o.RoleSessionName = sessionName
			}
		},
	)

	credSupplier := &awsCredentialsSupplier{
		region:     region,
		credsCache: credsCache,
	}

	extCfg := externalaccount.Config{
		Audience:                       params.Audience,
		SubjectTokenType:               awsTokenType,
		TokenURL:                       gcpSTSTokenURL,
		Scopes:                         []string{gcpCloudPlatformScope},
		AwsSecurityCredentialsSupplier: credSupplier,
	}
	if params.ServiceAccountEmail != "" {
		extCfg.ServiceAccountImpersonationURL = gcpIAMCredentialsURL + params.ServiceAccountEmail + ":generateAccessToken"
	}

	if params.HTTPClient != nil {
		ctx = context.WithValue(ctx, oauth2.HTTPClient, params.HTTPClient)
	}

	return externalaccount.NewTokenSource(ctx, extCfg)
}

// awsCredentialsSupplier implements externalaccount.AwsSecurityCredentialsSupplier.
type awsCredentialsSupplier struct {
	region     string
	credsCache *awssdk.CredentialsCache
}

func (s *awsCredentialsSupplier) AwsRegion(_ context.Context, _ externalaccount.SupplierOptions) (string, error) {
	return s.region, nil
}

func (s *awsCredentialsSupplier) AwsSecurityCredentials(ctx context.Context, _ externalaccount.SupplierOptions) (*externalaccount.AwsSecurityCredentials, error) {
	creds, err := s.credsCache.Retrieve(ctx)
	if err != nil {
		return nil, fmt.Errorf("retrieving AWS credentials for GCP WIF: %w", err)
	}
	return &externalaccount.AwsSecurityCredentials{
		AccessKeyID:     creds.AccessKeyID,
		SecretAccessKey: creds.SecretAccessKey,
		SessionToken:    creds.SessionToken,
	}, nil
}
