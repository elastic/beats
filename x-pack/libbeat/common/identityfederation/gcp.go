// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package identityfederation

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
)

const (
	gcpSTSTokenURL        = "https://sts.googleapis.com/v1/token"                                  //nolint:gosec // not a credential, it's a public API endpoint
	gcpIAMCredentialsURL  = "https://iamcredentials.googleapis.com/v1/projects/-/serviceAccounts/" //nolint:gosec // not a credential, it's a public API endpoint
	awsTokenType          = "urn:ietf:params:aws:token-type:aws4_request"                          //nolint:gosec // not a credential, it's an IETF token type identifier
	jwtTokenType          = "urn:ietf:params:oauth:token-type:jwt"                                 //nolint:gosec // not a credential, it's an IETF token type identifier
	gcpCloudPlatformScope = "https://www.googleapis.com/auth/cloud-platform"
	defaultAWSRegion      = "us-east-1"
)

// GCPWIIParams configures the direct OIDC Workload Identity Federation flow using WII.
//
// Authentication chain:
//  1. POST WII /token via mTLS → short-lived JWT (audience = GCP WIF provider URL)
//  2. GCP STS SubjectTokenType=jwt token exchange → short-lived GCP access token
//  3. (optional) Impersonate ServiceAccountEmail
type GCPWIIParams struct {
	// Audience is the GCP WIF provider URL:
	// "//iam.googleapis.com/projects/<proj>/locations/global/workloadIdentityPools/<pool>/providers/<prov>"
	Audience string
	// WIIIssuerURL is the WII ECP proxy URL (WORKLOAD_IDENTITY_ISSUER_URL env var).
	WIIIssuerURL string
	// WIICertFile is the path to the WII client certificate (WORKLOAD_IDENTITY_SSL_CERT_FILE env var).
	WIICertFile string
	// WIIKeyFile is the path to the WII client private key (WORKLOAD_IDENTITY_SSL_KEY_FILE env var).
	WIIKeyFile string
	// ServiceAccountEmail is the GCP service account to impersonate.
	// If empty, service account impersonation is skipped.
	ServiceAccountEmail string
	// HTTPClient is an optional HTTP client for the GCP STS token exchange.
	HTTPClient *http.Client
}

// wiiOIDCSubjectTokenSupplier implements externalaccount.SubjectTokenSupplier by
// fetching a JWT from the WII /token endpoint via mTLS.
type wiiOIDCSubjectTokenSupplier struct {
	source *WIITokenSource
}

func (s *wiiOIDCSubjectTokenSupplier) SubjectToken(_ context.Context, _ externalaccount.SupplierOptions) (string, error) {
	token, err := s.source.GetIdentityToken()
	if err != nil {
		return "", fmt.Errorf("fetching WII OIDC subject token for GCP: %w", err)
	}
	return string(token), nil
}

// GCPNewWIITokenSource creates an OAuth2 token source for GCP using direct OIDC
// Workload Identity Federation via the WII /token endpoint (mTLS). The JWT is
// submitted directly to GCP STS — no intermediate AWS role is assumed.
//
// The Audience field must be the GCP WIF provider URL that the customer's identity
// pool expects (used both as the WII token request audience and the GCP STS audience).
func GCPNewWIITokenSource(ctx context.Context, params GCPWIIParams) (oauth2.TokenSource, error) {
	if params.Audience == "" {
		return nil, fmt.Errorf("GCPWIIParams.Audience is required")
	}
	if params.WIIIssuerURL == "" {
		return nil, fmt.Errorf("GCPWIIParams.WIIIssuerURL is required")
	}
	if params.WIICertFile == "" || params.WIIKeyFile == "" {
		return nil, fmt.Errorf("GCPWIIParams.WIICertFile and WIIKeyFile are required")
	}

	tokenSource, err := NewWIITokenSource(params.WIIIssuerURL, params.WIICertFile, params.WIIKeyFile, params.Audience)
	if err != nil {
		return nil, fmt.Errorf("configuring WII token source: %w", err)
	}

	extCfg := externalaccount.Config{
		Audience:             params.Audience,
		SubjectTokenType:     jwtTokenType,
		TokenURL:             gcpSTSTokenURL,
		Scopes:               []string{gcpCloudPlatformScope},
		SubjectTokenSupplier: &wiiOIDCSubjectTokenSupplier{source: tokenSource},
	}
	if params.ServiceAccountEmail != "" {
		extCfg.ServiceAccountImpersonationURL = gcpIAMCredentialsURL + params.ServiceAccountEmail + ":generateAccessToken"
	}

	if params.HTTPClient != nil {
		ctx = context.WithValue(ctx, oauth2.HTTPClient, params.HTTPClient)
	}

	return externalaccount.NewTokenSource(ctx, extCfg)
}

// GCPParams configures the AWS-mediated GCP Workload Identity Federation flow.
//
// Authentication chain:
//  1. Read JWT from JWTFilePath
//  2. AssumeRoleWithWebIdentity → GlobalRoleARN (using JWT)
//  3. Supply AWS credentials to GCP STS for WIF token exchange
//  4. Impersonate ServiceAccountEmail in the customer's GCP project (when set)
type GCPParams struct {
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

func (p GCPParams) validate() error {
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

// GCPNewTokenSource creates an OAuth2 token source for GCP using AWS-mediated Workload
// Identity Federation. The returned token source automatically refreshes credentials.
//
// If params.HTTPClient is set it is injected into both the AWS STS call and the
// GCP token exchange context, enabling FIPS-compliant TLS throughout the chain.
func GCPNewTokenSource(ctx context.Context, params GCPParams) (oauth2.TokenSource, error) {
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
	credsCache := AWSNewWebIdentityCredentialsCache(
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
