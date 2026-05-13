// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"bytes"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

func mockNow(v time.Time) func() time.Time { return func() time.Time { return v } }

type mockRoundTripper struct {
	mock.Mock
	req *http.Request
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	m.req = req                                        // store the request for later assertions.
	return args.Get(0).(*http.Response), args.Error(1) //nolint:errcheck // not needed here.
}

func TestSignerTransportRoundTrip(t *testing.T) {
	now := mockNow(time.Date(2025, time.October, 11, 16, 0, 0, 0, time.UTC))

	// fake credentials received from this: https://docs.aws.amazon.com/STS/latest/APIReference/API_GetAccessKeyInfo.html
	fakeStaticCreds := credentials.NewStaticCredentialsProvider("AKIAIOSFODNN7EXAMPLE", "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY", "session")

	tests := []struct {
		name                   string
		defaultServiceName     string
		defaultRegion          string
		url                    string
		requestBody            io.Reader
		requestHeaders         map[string]string
		credentials            aws.CredentialsProvider
		now                    func() time.Time
		initMockRoundTripper   func(*mockRoundTripper)
		expectError            bool
		expectedRequestHeaders map[string]string
		expectedRequestBody    []byte
	}{
		{
			name:               "no body",
			defaultServiceName: "",
			defaultRegion:      "",
			url:                "https://guardduty.us-east-1.amazonaws.com/detector/abc123/findings",
			requestBody:        http.NoBody,
			requestHeaders:     map[string]string{},
			credentials:        fakeStaticCreds,
			now:                now,
			initMockRoundTripper: func(mrt *mockRoundTripper) {
				mrt.On("RoundTrip", mock.Anything).Return(&http.Response{}, nil).Once()
			},
			expectError: false,
			expectedRequestHeaders: map[string]string{
				"Authorization":        "AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20251011/us-east-1/guardduty/aws4_request, SignedHeaders=host;x-amz-date;x-amz-security-token, Signature=a73ff41e90b3e54c8855dc53cb352c244f4cf39122838e4ded22eef0fde01095",
				"X-Amz-Date":           "20251011T160000Z",
				"X-Amz-Security-Token": "session",
			},
			expectedRequestBody: []byte{},
		},
		{
			name:               "no body overwrite service name and region",
			defaultServiceName: "guardduty",
			defaultRegion:      "us-east-1",
			url:                "https://guardduty2.us-east-2.amazonaws.com/detector/abc123/findings",
			requestBody:        http.NoBody,
			requestHeaders:     map[string]string{},
			credentials:        fakeStaticCreds,
			now:                now,
			initMockRoundTripper: func(mrt *mockRoundTripper) {
				mrt.On("RoundTrip", mock.Anything).Return(&http.Response{}, nil).Once()
			},
			expectError: false,
			expectedRequestHeaders: map[string]string{
				"Authorization":        "AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20251011/us-east-1/guardduty/aws4_request, SignedHeaders=host;x-amz-date;x-amz-security-token, Signature=2bc3ea894efa9703ec95cac0bdcd6a1067a64636058b66e88640af2dc06ff2dd",
				"X-Amz-Date":           "20251011T160000Z",
				"X-Amz-Security-Token": "session",
			},
			expectedRequestBody: []byte{},
		},
		{
			name:               "with body",
			defaultServiceName: "",
			defaultRegion:      "",
			url:                "https://guardduty.us-east-1.amazonaws.com/detector/abc123/findings",
			requestBody:        bytes.NewBuffer([]byte(`{"findingIds": [ "abc" ], "sortCriteria": {"attributeName":"updatedAt","orderBy":"ASC"}}`)),
			requestHeaders:     map[string]string{},
			credentials:        fakeStaticCreds,
			now:                now,
			initMockRoundTripper: func(mrt *mockRoundTripper) {
				mrt.On("RoundTrip", mock.Anything).Return(&http.Response{}, nil).Once()
			},
			expectError: false,
			expectedRequestHeaders: map[string]string{
				"Authorization":        "AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20251011/us-east-1/guardduty/aws4_request, SignedHeaders=content-length;host;x-amz-date;x-amz-security-token, Signature=1cba6843418733071843e982a5e399eebfa3caeef3bae336ab4477abf42a9fb7",
				"X-Amz-Date":           "20251011T160000Z",
				"X-Amz-Security-Token": "session",
			},
			expectedRequestBody: []byte(`{"findingIds": [ "abc" ], "sortCriteria": {"attributeName":"updatedAt","orderBy":"ASC"}}`),
		},
		{
			name:               "with body and headers",
			defaultServiceName: "",
			defaultRegion:      "",
			url:                "https://guardduty.us-east-1.amazonaws.com/detector/abc123/findings",
			requestBody:        bytes.NewBuffer([]byte(`{"findingIds": [ "abc" ], "sortCriteria": {"attributeName":"updatedAt","orderBy":"ASC"}}`)),
			requestHeaders:     map[string]string{"X-Extra-Header": "abc123"},
			credentials:        fakeStaticCreds,
			now:                now,
			initMockRoundTripper: func(mrt *mockRoundTripper) {
				mrt.On("RoundTrip", mock.Anything).Return(&http.Response{}, nil).Once()
			},
			expectError: false,
			expectedRequestHeaders: map[string]string{
				"Authorization":        "AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20251011/us-east-1/guardduty/aws4_request, SignedHeaders=content-length;host;x-amz-date;x-amz-security-token;x-extra-header, Signature=a9ae9766395c5749fca156baf9c65ef78d4d3053866299db48838c3546aaeb25",
				"X-Amz-Date":           "20251011T160000Z",
				"X-Amz-Security-Token": "session",
				"X-Extra-Header":       "abc123",
			},
			expectedRequestBody: []byte(`{"findingIds": [ "abc" ], "sortCriteria": {"attributeName":"updatedAt","orderBy":"ASC"}}`),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			logger := logptest.NewTestingLogger(t, "")

			m := mockRoundTripper{}
			if tc.initMockRoundTripper != nil {
				tc.initMockRoundTripper(&m)
			}

			st := initializeSignerTransport(logger, tc.defaultServiceName, tc.defaultRegion, tc.credentials, &m)
			st.now = tc.now

			assert.Equal(t, tc.defaultServiceName, st.serviceName)
			assert.Equal(t, tc.defaultRegion, st.region)

			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, tc.url, tc.requestBody)
			require.NoError(t, err)
			for k, v := range tc.requestHeaders {
				req.Header.Set(k, v)
			}

			_, err = st.RoundTrip(req) //nolint:bodyclose // we don't actually have response body here
			errAssert := assert.NoError
			if tc.expectError {
				errAssert = assert.Error
			}
			errAssert(t, err)

			gotHeaders := map[string]string{}
			for k := range m.req.Header {
				gotHeaders[k] = m.req.Header.Get(k)
			}
			assert.Equal(t, tc.expectedRequestHeaders, gotHeaders)

			// ensure that request's body is readable (and not consumed) after the hash operation.
			b, err := io.ReadAll(req.Body)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedRequestBody, b)
		})
	}
}

func TestBodySHA256Hash(t *testing.T) {
	tests := []struct {
		name         string
		body         io.Reader
		expectedHash string
		expectError  bool
	}{
		{
			name:         "no body",
			body:         http.NoBody,
			expectedHash: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			expectError:  false,
		},
		{
			name:         "with body that initializes GetBody",
			body:         bytes.NewReader([]byte(`"abc"`)),
			expectedHash: "6cc43f858fbb763301637b5af970e2a46b46f461f27e5a0f41e009c59b827b25",
			expectError:  false,
		},
		{
			name:         "with body without initialized GetBody",
			body:         io.NopCloser(bytes.NewReader([]byte(`"abc"`))),
			expectedHash: "6cc43f858fbb763301637b5af970e2a46b46f461f27e5a0f41e009c59b827b25",
			expectError:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			logger := logptest.NewTestingLogger(t, "")
			st := SignerTransport{
				next:        nil,
				credentials: nil,
				signer:      nil,
				logger:      logger,
				serviceName: "",
				region:      "",
				now:         time.Now,
			}

			req, err := http.NewRequestWithContext(t.Context(), "GET", "sample.amazonaws.com", tc.body)
			require.NoError(t, err)

			gotHash, gotErr := st.bodySHA256Hash(req)

			assert.Equal(t, tc.expectedHash, gotHash, "hash of body is different than expected")
			errAssert := assert.NoError
			if tc.expectError {
				errAssert = assert.Error
			}
			errAssert(t, gotErr)
		})
	}
}

func TestGetServiceAndRegion(t *testing.T) {
	tests := []struct {
		name                  string
		configuredServiceName string
		configuredRegion      string
		requestHost           string
		expectedServiceName   string
		expectedRegion        string
		expectError           bool
	}{
		{
			name:                  "extract from host",
			configuredServiceName: "",
			configuredRegion:      "",
			requestHost:           "guardduty.us-east-1.amazonaws.com",
			expectedServiceName:   "guardduty",
			expectedRegion:        "us-east-1",
			expectError:           false,
		},
		{
			name:                  "configured values take precedence",
			configuredServiceName: "guardduty",
			configuredRegion:      "us-east-1",
			requestHost:           "abc.us-east-2.amazonaws.com",
			expectedServiceName:   "guardduty",
			expectedRegion:        "us-east-1",
			expectError:           false,
		},
		{
			name:                  "service name configured region from url",
			configuredServiceName: "guardduty",
			configuredRegion:      "",
			requestHost:           "abc.us-east-2.amazonaws.com",
			expectedServiceName:   "guardduty",
			expectedRegion:        "us-east-2",
			expectError:           false,
		},
		{
			name:                  "service name from url region configured",
			configuredServiceName: "",
			configuredRegion:      "us-east-1",
			requestHost:           "guardduty.us-east-2.amazonaws.com",
			expectedServiceName:   "guardduty",
			expectedRegion:        "us-east-1",
			expectError:           false,
		},
		{
			name:                  "malformed host",
			configuredServiceName: "",
			configuredRegion:      "",
			requestHost:           "amazonaws.com",
			expectedServiceName:   "",
			expectedRegion:        "",
			expectError:           true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			logger := logptest.NewTestingLogger(t, "")
			st := SignerTransport{
				next:        nil,
				credentials: nil,
				signer:      nil,
				logger:      logger,
				serviceName: tc.configuredServiceName,
				region:      tc.configuredRegion,
				now:         time.Now,
			}

			req := &http.Request{Host: tc.requestHost}

			gotServiceName, gotRegion, gotErr := st.getServiceAndRegion(req)

			assert.Equal(t, tc.expectedServiceName, gotServiceName, "service name is different than expected")
			assert.Equal(t, tc.expectedRegion, gotRegion, "service name is different than expected")
			errAssert := assert.NoError
			if tc.expectError {
				errAssert = assert.Error
			}
			errAssert(t, gotErr)
		})
	}
}
