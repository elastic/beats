// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	awslogging "github.com/aws/smithy-go/logging"

	"github.com/elastic/elastic-agent-libs/logp"
)

type SignerInputConfig struct {
	Enabled     *bool  `config:"enabled"`
	ServiceName string `config:"service_name"` // optional
	ConfigAWS   `config:",inline"`
}

// IsEnabled returns true if the `enable` field is set to true in the yaml.
func (c *SignerInputConfig) IsEnabled() bool {
	return c != nil && (c.Enabled == nil || *c.Enabled)
}

// SignerTransport implements [http.RoundTripper] interface
// and sings requests with aws v4 signer before send them to the next roundtripper.
type SignerTransport struct {
	next        http.RoundTripper
	credentials aws.CredentialsProvider
	signer      *v4.Signer
	logger      *logp.Logger
	serviceName string
	region      string
	now         func() time.Time // we don't use [time.Now] directly, so we can mock time in tests.
}

func InitializeSingerTransport(cfg SignerInputConfig, logger *logp.Logger, nextTransport http.RoundTripper) (*SignerTransport, error) {
	awsConfig, err := InitializeAWSConfig(cfg.ConfigAWS, logger)
	if err != nil {
		return nil, err
	}

	return initializeSingerTransport(logger, cfg.ServiceName, cfg.DefaultRegion, awsConfig.Credentials, nextTransport), nil
}

func initializeSingerTransport(logger *logp.Logger, defaultServiceName string, defaultRegion string, credentials aws.CredentialsProvider, nextTransport http.RoundTripper) *SignerTransport {
	return &SignerTransport{
		next:        nextTransport,
		credentials: credentials,
		signer: v4.NewSigner(func(signer *v4.SignerOptions) {
			signer.Logger = awslogging.LoggerFunc(func(classification awslogging.Classification, format string, v ...any) {
				switch classification {
				case awslogging.Debug:
					logger.Debugf(format, v...)
				case awslogging.Warn:
					logger.Warnf(format, v...)
				}
			})
		}),
		logger:      logger,
		serviceName: defaultRegion,
		region:      defaultServiceName,
		now:         time.Now,
	}
}

func (st *SignerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// resolve service name and region (if they are not configured)
	serviceName, region, err := st.getServiceAndRegion(req)
	if err != nil {
		return nil, fmt.Errorf("error while getting service name and region: [%w]", err)
	}

	// retrieve credentials
	creds, err := st.credentials.Retrieve(req.Context())
	if err != nil {
		return nil, fmt.Errorf("error while retrieving credentials: [%w]", err)
	}

	// body hash
	payloadHash, err := st.bodySHA256Hash(req)
	if err != nil {
		return nil, fmt.Errorf("error while calculating body hash: [%w]", err)
	}

	// sing the request
	err = st.signer.SignHTTP(req.Context(), creds, req, payloadHash, serviceName, region, st.now())
	if err != nil {
		return nil, fmt.Errorf("error while signing the request: %w", err)
	}

	// next transport
	return st.next.RoundTrip(req)
}

// bodySHA256Hash returns the sha256 hash of the request's body by reading a copy of the body.
// The request's Body remains readable and unmodified after this function returns.
func (st *SignerTransport) bodySHA256Hash(req *http.Request) (string, error) {
	if req.Body == nil || req.Body == http.NoBody {
		return hex.EncodeToString(sha256.New().Sum(nil)), nil
	}

	// this is a copy of the original body
	body, err := st.getBody(req)
	if err != nil {
		return "", err
	}

	hash := sha256.New()

	if _, err := io.Copy(hash, body); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), body.Close()
}

// getBody returns a copy of the request's body as a [io.ReadCloser].
// The request's Body remains readable and unmodified after this function returns.
func (st *SignerTransport) getBody(req *http.Request) (io.ReadCloser, error) {
	if req.GetBody != nil {
		// [http.Request] GetBody dictates that a new copy of the body must be returned.
		return req.GetBody()
	}

	if req.Body == http.NoBody || req.Body == nil {
		return req.Body, nil
	}

	// If the GetBody does not exist we need to manually copy the body.
	// In Beats use-case its not possible for this to happen,
	// since, both in cel and httpjson, the request is initialized
	// with *bytes.Buffer, *bytes.Reader or *strings.Reader as body, which gets GetBody initialized.
	// httpjson: (x-pack/filebeat/input/httpjson/request.go newHTTPRequest)
	// cel: mito repo (lib/http.go)
	// We cover the edge case here by reading and coping the body manually.
	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, fmt.Errorf("error while reading request body: [%w]", err)
	}
	if err := req.Body.Close(); err != nil {
		st.logger.Warnf("error while closing copied body %s", err.Error())
	}

	// reset body to the request
	req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	return io.NopCloser(bytes.NewBuffer(bodyBytes)), nil
}

// getServiceAndRegion returns the service name and the region for the upcoming request.
// If service name and region are configured with default values, those take precedence.
// Otherwise it will try to parse the values from [http.Request] Host value.
func (st *SignerTransport) getServiceAndRegion(req *http.Request) (serviceName, region string, err error) {
	serviceName = st.serviceName
	region = st.region

	if serviceName == "" || region == "" {
		s, r, err := parseServiceAndRegionFromHost(req.Host)
		if err != nil {
			return "", "", err
		}
		if serviceName == "" {
			serviceName = s
		}
		if region == "" {
			region = r
		}
	}

	return serviceName, region, nil
}

func parseServiceAndRegionFromHost(host string) (service, region string, err error) {
	parts := strings.SplitN(host, ".", 4)

	if len(parts) < 4 {
		return "", "", errMalformedHost
	}

	return parts[0], parts[1], nil
}

var errMalformedHost = errors.New("malformed host string")
