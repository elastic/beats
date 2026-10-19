// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/aws-sdk-go-v2/service/sts/types"
	"github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/transport/tlscommon"

	"github.com/elastic/beats/v7/x-pack/libbeat/common/identityfederation"
)

func TestInitializeAWSConfigIdentityFederation(t *testing.T) {
	t.Setenv(identityfederation.AWSGlobalRoleARNEnvVar, "arn:aws:iam::999999999999:role/elastic-global-role")
	t.Setenv(identityfederation.AWSIDTokenFileEnvVar, "/path/token")
	t.Setenv(identityfederation.AWSCloudResourceIDEnvVar, "abc123")

	inputConfig := ConfigAWS{
		RoleArn:            "arn:aws:iam::123456789012:role/customer-role",
		ExternalID:         "external-id-456",
		UseCloudConnectors: true,
	}

	awsConfig, err := InitializeAWSConfig(inputConfig, logptest.NewTestingLogger(t, ""))
	assert.NoError(t, err)

	// we cannot append to APIOptions at this point (and mock the chain responses)
	// because a copy of config has already been passed to each sts client.
	// So lets just check that .Credentials is CredentialsCache (so cloud connectors init was run).
	c, isCredCache := awsConfig.Credentials.(*aws.CredentialsCache)
	require.True(t, isCredCache)
	require.NotNil(t, c)
}

// TestApplyIdentityFederationChain exercises the full two-step STS chain with mock responses,
// verifying that the correct parameters are sent to STS at each step.
func TestApplyIdentityFederationChain(t *testing.T) {
	config := ConfigAWS{
		RoleArn:                "arn:aws:iam::123456789012:role/customer-role",
		ExternalID:             "external-id-456",
		AssumeRoleDuration:     45 * time.Minute,
		AssumeRoleExpiryWindow: 10 * time.Minute,
	}

	globalRoleARN := "arn:aws:iam::999999999999:role/elastic-global-role"
	cloudResourceID := "abcd1234"
	tokenFileContent := "abc123"

	tmpDir := t.TempDir()
	pth := path.Join(tmpDir, "id_token")
	_ = os.WriteFile(pth, []byte(tokenFileContent), 0o644)

	t.Setenv(identityfederation.AWSGlobalRoleARNEnvVar, globalRoleARN)
	t.Setenv(identityfederation.AWSIDTokenFileEnvVar, pth)
	t.Setenv(identityfederation.AWSCloudResourceIDEnvVar, cloudResourceID)

	// Create a base AWS config with a mock STS interceptor injected via APIOptions.
	// The interceptor must be set on the base config before calling applyIdentityFederationChain,
	// because each step creates its STS client from the config at that point in the chain.
	baseConfig := &aws.Config{
		Region:       "us-east-1",
		BaseEndpoint: aws.String("https://aws.mock"),
	}

	receivedCalls := 0
	baseConfig.APIOptions = append(baseConfig.APIOptions, func(stack *middleware.Stack) error {
		return stack.Finalize.Add(
			middleware.FinalizeMiddlewareFunc(
				"mock",
				func(ctx context.Context, in middleware.FinalizeInput, next middleware.FinalizeHandler) (middleware.FinalizeOutput, middleware.Metadata, error) {
					req, is := in.Request.(*smithyhttp.Request)
					require.Truef(t, is, "expected *smithyhttp.Request, got: %T", in.Request)
					receivedCalls++
					bd, err := io.ReadAll(req.GetStream())
					assert.NoError(t, req.RewindStream())
					assert.NoError(t, err)
					body := string(bd)

					switch receivedCalls {

					// Step 1: AssumeRoleWithWebIdentity → Elastic Global Role
					case 1:
						q, err := url.ParseQuery(body)
						assert.NoError(t, err)
						assert.Equal(t, "AssumeRoleWithWebIdentity", q.Get("Action"))
						assert.Equal(t, "1200", q.Get("DurationSeconds")) // defaultIntermediateDuration
						assert.Equal(t, globalRoleARN, q.Get("RoleArn"))
						assert.Equal(t, tokenFileContent, q.Get("WebIdentityToken"))
						return middleware.FinalizeOutput{
							Result: &sts.AssumeRoleWithWebIdentityOutput{
								Credentials: &types.Credentials{
									AccessKeyId:     aws.String("AKIAFAKEEXAMPLE00001"),
									SecretAccessKey: aws.String("FAKEwJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY1"),
									SessionToken:    aws.String("FwoGZXIvYXdzEFAaDFAKESESSIONTOKENEXAMPLE1"),
									Expiration:      aws.Time(time.Now().Add(defaultIntermediateDuration)),
								},
							},
						}, middleware.Metadata{}, nil

					// Step 2: AssumeRole → customer remote role
					case 2:
						q, err := url.ParseQuery(body)
						assert.NoError(t, err)
						assert.Equal(t, "AssumeRole", q.Get("Action"))
						assert.Equal(t, "2700", q.Get("DurationSeconds")) // 45 * time.Minute
						assert.Equal(t, cloudResourceID+"-"+config.ExternalID, q.Get("ExternalId"))
						assert.Equal(t, config.RoleArn, q.Get("RoleArn"))
						return middleware.FinalizeOutput{
							Result: &sts.AssumeRoleOutput{
								Credentials: &types.Credentials{
									AccessKeyId:     aws.String("AKIAFAKEEXAMPLE00002"),
									SecretAccessKey: aws.String("FAKEwJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY2"),
									SessionToken:    aws.String("FwoGZXIvYXdzEFAaDFAKESESSIONTOKENEXAMPLE2"),
									Expiration:      aws.Time(time.Now().Add(defaultIntermediateDuration)),
								},
							},
						}, middleware.Metadata{}, nil

					default:
						t.Fatal("unexpected aws sdk call")
						return middleware.FinalizeOutput{}, middleware.Metadata{}, fmt.Errorf("unexpected operation")
					}
				},
			),
			middleware.After,
		)
	})

	err := applyIdentityFederationChain(config, baseConfig, logptest.NewTestingLogger(t, ""))
	require.NoError(t, err)

	require.NotNil(t, baseConfig.Credentials, "credentials provider should be set")
	crd, err := baseConfig.Credentials.Retrieve(t.Context())
	require.NoError(t, err)
	require.NotNil(t, crd)
	require.Equal(t, 2, receivedCalls)
}

// TestApplyIdentityFederationChainIRSA exercises the IRSA two-step STS chain:
// IRSA pod creds → AssumeRole(GlobalRole) → AssumeRole(customer role).
func TestApplyIdentityFederationChainIRSA(t *testing.T) {
	config := ConfigAWS{
		RoleArn:            "arn:aws:iam::123456789012:role/customer-role",
		ExternalID:         "external-id-456",
		AssumeRoleDuration: 45 * time.Minute,
	}

	globalRoleARN := "arn:aws:iam::999999999999:role/elastic-global-role"
	cloudResourceID := "abcd1234"

	t.Setenv(identityfederation.AWSIRSATokenFileEnvVar, "/var/run/secrets/irsa-token") // trigger IRSA path
	t.Setenv(identityfederation.AWSGlobalRoleARNEnvVar, globalRoleARN)
	// AWSIDTokenFileEnvVar intentionally NOT set — IRSA path must not require it
	t.Setenv(identityfederation.AWSCloudResourceIDEnvVar, cloudResourceID)

	baseConfig := &aws.Config{
		Region:       "us-east-1",
		BaseEndpoint: aws.String("https://aws.mock"),
	}

	receivedCalls := 0
	baseConfig.APIOptions = append(baseConfig.APIOptions, func(stack *middleware.Stack) error {
		return stack.Finalize.Add(
			middleware.FinalizeMiddlewareFunc(
				"mock",
				func(ctx context.Context, in middleware.FinalizeInput, next middleware.FinalizeHandler) (middleware.FinalizeOutput, middleware.Metadata, error) {
					req, is := in.Request.(*smithyhttp.Request)
					require.Truef(t, is, "expected *smithyhttp.Request, got: %T", in.Request)
					receivedCalls++
					bd, err := io.ReadAll(req.GetStream())
					assert.NoError(t, req.RewindStream())
					assert.NoError(t, err)
					body := string(bd)

					switch receivedCalls {

					// Step 1: AssumeRole → Elastic Global Role (IRSA path, no WebIdentity)
					case 1:
						q, err := url.ParseQuery(body)
						assert.NoError(t, err)
						assert.Equal(t, "AssumeRole", q.Get("Action"))
						assert.Equal(t, "1200", q.Get("DurationSeconds")) // defaultIntermediateDuration
						assert.Equal(t, globalRoleARN, q.Get("RoleArn"))
						assert.Empty(t, q.Get("WebIdentityToken"), "IRSA step must not use WebIdentity")
						return middleware.FinalizeOutput{
							Result: &sts.AssumeRoleOutput{
								Credentials: &types.Credentials{
									AccessKeyId:     aws.String("AKIAFAKEEXAMPLE00001"),
									SecretAccessKey: aws.String("FAKEwJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY1"),
									SessionToken:    aws.String("FwoGZXIvYXdzEFAaDFAKESESSIONTOKENEXAMPLE1"),
									Expiration:      aws.Time(time.Now().Add(defaultIntermediateDuration)),
								},
							},
						}, middleware.Metadata{}, nil

					// Step 2: AssumeRole → customer remote role
					case 2:
						q, err := url.ParseQuery(body)
						assert.NoError(t, err)
						assert.Equal(t, "AssumeRole", q.Get("Action"))
						assert.Equal(t, "2700", q.Get("DurationSeconds")) // 45 * time.Minute
						assert.Equal(t, cloudResourceID+"-"+config.ExternalID, q.Get("ExternalId"))
						assert.Equal(t, config.RoleArn, q.Get("RoleArn"))
						return middleware.FinalizeOutput{
							Result: &sts.AssumeRoleOutput{
								Credentials: &types.Credentials{
									AccessKeyId:     aws.String("AKIAFAKEEXAMPLE00002"),
									SecretAccessKey: aws.String("FAKEwJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY2"),
									SessionToken:    aws.String("FwoGZXIvYXdzEFAaDFAKESESSIONTOKENEXAMPLE2"),
									Expiration:      aws.Time(time.Now().Add(defaultIntermediateDuration)),
								},
							},
						}, middleware.Metadata{}, nil

					default:
						t.Fatal("unexpected aws sdk call")
						return middleware.FinalizeOutput{}, middleware.Metadata{}, fmt.Errorf("unexpected operation")
					}
				},
			),
			middleware.After,
		)
	})

	err := applyIdentityFederationChain(config, baseConfig, logptest.NewTestingLogger(t, ""))
	require.NoError(t, err)

	require.NotNil(t, baseConfig.Credentials, "credentials provider should be set")
	crd, err := baseConfig.Credentials.Retrieve(t.Context())
	require.NoError(t, err)
	require.NotNil(t, crd)
	require.Equal(t, 2, receivedCalls)
}

func TestApplyIdentityFederationChainValidation(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	validRoleARN := "arn:aws:iam::123456789012:role/customer-role"

	t.Run("missing cloud resource id", func(t *testing.T) {
		t.Setenv(identityfederation.AWSGlobalRoleARNEnvVar, "arn:aws:iam::999999999999:role/elastic-global-role")
		t.Setenv(identityfederation.AWSIDTokenFileEnvVar, "/path/token")
		// CloudResourceIDEnvVar intentionally not set

		err := applyIdentityFederationChain(ConfigAWS{RoleArn: validRoleARN}, &aws.Config{}, logger)
		require.ErrorContains(t, err, "cloud resource id")
	})

	t.Run("missing all env vars", func(t *testing.T) {
		err := applyIdentityFederationChain(ConfigAWS{}, &aws.Config{}, logger)
		require.ErrorContains(t, err, "elastic global role")
		require.ErrorContains(t, err, "id token")
		require.ErrorContains(t, err, "cloud resource id")
		require.ErrorContains(t, err, "role_arn")
	})

	t.Run("assume role duration exceeds 1h", func(t *testing.T) {
		t.Setenv(identityfederation.AWSGlobalRoleARNEnvVar, "arn:aws:iam::999999999999:role/elastic-global-role")
		t.Setenv(identityfederation.AWSIDTokenFileEnvVar, "/path/token")
		t.Setenv(identityfederation.AWSCloudResourceIDEnvVar, "abc123")

		err := applyIdentityFederationChain(ConfigAWS{RoleArn: validRoleARN, AssumeRoleDuration: 2 * time.Hour}, &aws.Config{}, logger)
		require.ErrorContains(t, err, "assume role duration cannot exceed 1h")
	})

	t.Run("irsa: id token path not required when AWS_WEB_IDENTITY_TOKEN_FILE is set", func(t *testing.T) {
		t.Setenv(identityfederation.AWSIRSATokenFileEnvVar, "/var/run/secrets/token")
		t.Setenv(identityfederation.AWSGlobalRoleARNEnvVar, "arn:aws:iam::999999999999:role/elastic-global-role")
		t.Setenv(identityfederation.AWSCloudResourceIDEnvVar, "abc123")
		// IDTokenFileEnvVar intentionally not set — should not be required in IRSA mode

		// We only check that the error is not about "id token"; the chain itself will fail later when STS is called, but config validation should pass.
		err := applyIdentityFederationChain(ConfigAWS{RoleArn: validRoleARN}, &aws.Config{Region: "us-east-1"}, logger)
		if err != nil {
			require.NotContains(t, err.Error(), "id token")
		}
	})

	t.Run("wii: cert and key files are required and there is no silent legacy fallback", func(t *testing.T) {
		t.Setenv(identityfederation.WIIIssuerURLEnvVar, "https://wii.example.com")
		t.Setenv(identityfederation.AWSGlobalRoleARNEnvVar, "arn:aws:iam::999999999999:role/elastic-global-role")
		t.Setenv(identityfederation.AWSIDTokenFileEnvVar, "/path/token")
		t.Setenv(identityfederation.AWSCloudResourceIDEnvVar, "abc123")

		err := applyIdentityFederationChain(ConfigAWS{RoleArn: validRoleARN}, &aws.Config{}, logger)
		require.ErrorContains(t, err, "WORKLOAD_IDENTITY_SSL_CERT_FILE")
	})

	t.Run("wii: role_arn is required", func(t *testing.T) {
		t.Setenv(identityfederation.WIIIssuerURLEnvVar, "https://wii.example.com")
		t.Setenv(identityfederation.WIISSLCertFileEnvVar, "/path/wii-client.crt")
		t.Setenv(identityfederation.WIISSLKeyFileEnvVar, "/path/wii-client.key")

		err := applyIdentityFederationChain(ConfigAWS{}, &aws.Config{}, logger)
		require.ErrorContains(t, err, "role_arn is required for WII identity federation")
	})
}

func TestInitializeAWSConfig(t *testing.T) {
	inputConfig := ConfigAWS{
		AccessKeyID:     "123",
		SecretAccessKey: "abc",
		TLS: &tlscommon.Config{
			VerificationMode: 1,
		},
		ProxyUrl: "http://proxy:3128",
	}
	awsConfig, err := InitializeAWSConfig(inputConfig, logptest.NewTestingLogger(t, ""))
	assert.NoError(t, err)

	retrievedAWSConfig, err := awsConfig.Credentials.Retrieve(context.Background())
	assert.NoError(t, err)

	assert.Equal(t, inputConfig.AccessKeyID, retrievedAWSConfig.AccessKeyID)
	assert.Equal(t, inputConfig.SecretAccessKey, retrievedAWSConfig.SecretAccessKey)
	assert.True(t, awsConfig.HTTPClient.(*http.Client).Transport.(*http.Transport).TLSClientConfig.InsecureSkipVerify) //nolint:errcheck // no need in test
	assert.NotNil(t, awsConfig.HTTPClient.(*http.Client).Transport.(*http.Transport).Proxy)                            //nolint:errcheck // no need in test
}

func TestGetAWSCredentials(t *testing.T) {
	inputConfig := ConfigAWS{
		AccessKeyID:     "123",
		SecretAccessKey: "abc",
		SessionToken:    "fake-session-token",
	}
	awsConfig, err := getAWSCredentials(inputConfig, logptest.NewTestingLogger(t, ""))
	assert.NoError(t, err)

	retrievedAWSConfig, err := awsConfig.Credentials.Retrieve(context.Background())
	assert.NoError(t, err)

	assert.Equal(t, inputConfig.AccessKeyID, retrievedAWSConfig.AccessKeyID)
	assert.Equal(t, inputConfig.SecretAccessKey, retrievedAWSConfig.SecretAccessKey)
	assert.Equal(t, inputConfig.SessionToken, retrievedAWSConfig.SessionToken)
}

func TestDefaultRegion(t *testing.T) {
	cases := []struct {
		title          string
		region         string
		expectedRegion string
	}{
		{
			"No default region set",
			"",
			"us-east-1",
		},
		{
			"us-west-1 region set as default",
			"us-west-1",
			"us-west-1",
		},
	}
	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			inputConfig := ConfigAWS{
				AccessKeyID:     "123",
				SecretAccessKey: "abc",
			}
			if c.region != "" {
				inputConfig.DefaultRegion = c.region
			}
			awsConfig, err := InitializeAWSConfig(inputConfig, logptest.NewTestingLogger(t, ""))
			assert.NoError(t, err)
			assert.Equal(t, c.expectedRegion, awsConfig.Region)
		})
	}
}

func TestApplyIdentityFederationChainWII(t *testing.T) {
	const wiiToken = "eyJhbGciOiJSUzI1NiJ9.wii-test.sig" //nolint:gosec // G101: not a credential, an inert JWT-shaped test fixture

	certPEM, keyPEM, err := genSelfSignedClientCert()
	require.NoError(t, err)
	tmpDir := t.TempDir()
	certPath := path.Join(tmpDir, "wii-client.crt")
	keyPath := path.Join(tmpDir, "wii-client.key")
	require.NoError(t, os.WriteFile(certPath, certPEM, 0o600))
	require.NoError(t, os.WriteFile(keyPath, keyPEM, 0o600))

	wiiCalls := 0
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wiiCalls++
		assert.Equal(t, "/token", r.URL.Path)
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.JSONEq(t, `{"aud":"sts.amazonaws.com"}`, string(body))
		_, _ = fmt.Fprintf(w, `{"token":%q,"expires_at":%d}`, wiiToken, time.Now().Add(time.Hour).Unix())
	}))
	defer srv.Close()

	serverCertPath := path.Join(tmpDir, "wii-server-ca.crt")
	require.NoError(t, os.WriteFile(serverCertPath, pem.EncodeToMemory(
		&pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw}), 0o600))

	t.Setenv(identityfederation.WIIIssuerURLEnvVar, srv.URL)
	t.Setenv(identityfederation.WIISSLCertFileEnvVar, certPath)
	t.Setenv(identityfederation.WIISSLKeyFileEnvVar, keyPath)
	t.Setenv(identityfederation.WIISSLCAFileEnvVar, serverCertPath)

	config := ConfigAWS{
		RoleArn:            "arn:aws:iam::123456789012:role/customer-role",
		ExternalID:         "external-id-456",
		AssumeRoleDuration: 45 * time.Minute,
	}

	baseConfig := &aws.Config{
		Region:       "us-east-1",
		BaseEndpoint: aws.String("https://aws.mock"),
	}

	stsCalls := 0
	baseConfig.APIOptions = append(baseConfig.APIOptions, func(stack *middleware.Stack) error {
		return stack.Finalize.Add(
			middleware.FinalizeMiddlewareFunc(
				"mock",
				func(ctx context.Context, in middleware.FinalizeInput, next middleware.FinalizeHandler) (middleware.FinalizeOutput, middleware.Metadata, error) {
					req, is := in.Request.(*smithyhttp.Request)
					require.Truef(t, is, "expected *smithyhttp.Request, got: %T", in.Request)
					stsCalls++
					bd, err := io.ReadAll(req.GetStream())
					assert.NoError(t, req.RewindStream())
					assert.NoError(t, err)

					q, err := url.ParseQuery(string(bd))
					assert.NoError(t, err)
					assert.Equal(t, "AssumeRoleWithWebIdentity", q.Get("Action"))
					assert.Equal(t, config.RoleArn, q.Get("RoleArn"), "the customer role must be assumed directly")
					assert.Equal(t, wiiToken, q.Get("WebIdentityToken"))
					assert.Equal(t, "2700", q.Get("DurationSeconds"))
					assert.Empty(t, q.Get("ExternalId"), "ExternalID must not be sent on the WII path")
					return middleware.FinalizeOutput{
						Result: &sts.AssumeRoleWithWebIdentityOutput{
							Credentials: &types.Credentials{
								AccessKeyId:     aws.String("AKIAFAKEEXAMPLE00003"),
								SecretAccessKey: aws.String("FAKEwJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY3"),
								SessionToken:    aws.String("FwoGZXIvYXdzEFAaDFAKESESSIONTOKENEXAMPLE3"),
								Expiration:      aws.Time(time.Now().Add(time.Hour)),
							},
						},
					}, middleware.Metadata{}, nil
				},
			),
			middleware.After,
		)
	})

	err = applyIdentityFederationChain(config, baseConfig, logptest.NewTestingLogger(t, ""))
	require.NoError(t, err)

	crd, err := baseConfig.Credentials.Retrieve(t.Context())
	require.NoError(t, err)
	require.Equal(t, "AKIAFAKEEXAMPLE00003", crd.AccessKeyID)
	require.Equal(t, 1, stsCalls, "WII path must issue a single direct STS call")
	require.Equal(t, 1, wiiCalls)
}

func genSelfSignedClientCert() (certPEM, keyPEM []byte, err error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "wii-test-client"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return nil, nil, err
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, nil, err
	}
	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	return certPEM, keyPEM, nil
}
