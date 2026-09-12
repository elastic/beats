// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package billing

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/azure"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

const testEndpoint = "https://management.azure.com/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Consumption/usageDetails"

// stubTransport is a policy.Transporter that replays canned responses and records
// the headers of every request it receives.
type stubTransport struct {
	mu        sync.Mutex
	responses []func(*http.Request) *http.Response
	requests  []http.Header
}

func (t *stubTransport) Do(req *http.Request) (*http.Response, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.requests = append(t.requests, req.Header.Clone())

	build := t.responses[len(t.responses)-1]
	if len(t.requests) <= len(t.responses) {
		build = t.responses[len(t.requests)-1]
	}

	resp := build(req)
	resp.Request = req
	if resp.Body == nil {
		resp.Body = http.NoBody
	}
	if resp.Header == nil {
		resp.Header = http.Header{}
	}

	return resp, nil
}

func (t *stubTransport) requestCount() int {
	t.mu.Lock()
	defer t.mu.Unlock()

	return len(t.requests)
}

func (t *stubTransport) recordedRequests() []http.Header {
	t.mu.Lock()
	defer t.mu.Unlock()

	return append([]http.Header(nil), t.requests...)
}

// response returns a canned response builder with the given status and headers.
func response(status int, headers map[string]string) func(*http.Request) *http.Response {
	return func(*http.Request) *http.Response {
		header := http.Header{}
		for name, value := range headers {
			// Assign directly so the exact (non canonical) header casing used by
			// Azure survives into the test fixture.
			header[name] = []string{value}
		}

		return &http.Response{StatusCode: status, Header: header, Body: http.NoBody}
	}
}

// headerValueInsensitive looks a header up ignoring key casing, which is required
// because we intentionally store some header keys in non canonical form.
func headerValueInsensitive(header http.Header, name string) (string, bool) {
	for key, values := range header {
		if strings.EqualFold(key, name) && len(values) > 0 {
			return values[0], true
		}
	}

	return "", false
}

// doThroughPolicy runs a single response through the retry-after translation policy
// and returns the (possibly rewritten) response.
func doThroughPolicy(t *testing.T, status int, headers map[string]string) *http.Response {
	t.Helper()

	transport := &stubTransport{responses: []func(*http.Request) *http.Response{response(status, headers)}}

	pipeline := runtime.NewPipeline("azurebillingtest", "v0.0.0", runtime.PipelineOptions{}, &policy.ClientOptions{
		Transport: transport,
		// Disable retries: this test only exercises the translation policy.
		Retry:            policy.RetryOptions{MaxRetries: -1},
		PerRetryPolicies: []policy.Policy{newRetryAfterTranslationPolicy(logptest.NewTestingLogger(t, ""))},
	})

	req, err := runtime.NewRequest(context.Background(), http.MethodGet, testEndpoint)
	require.NoError(t, err)

	resp, err := pipeline.Do(req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	return resp
}

func TestRetryAfterTranslationPolicy(t *testing.T) {
	tests := []struct {
		name             string
		status           int
		headers          map[string]string
		wantRetryAfter   string
		wantNoRetryAfter bool
	}{
		{
			name:           "qpu retry after is translated",
			status:         http.StatusTooManyRequests,
			headers:        map[string]string{"x-ms-ratelimit-microsoft.costmanagement-qpu-retry-after": "3"},
			wantRetryAfter: "3",
		},
		{
			name:           "entity retry after is translated",
			status:         http.StatusTooManyRequests,
			headers:        map[string]string{"x-ms-ratelimit-microsoft.costmanagement-entity-retry-after": "45"},
			wantRetryAfter: "45",
		},
		{
			name:           "tenant retry after is translated",
			status:         http.StatusTooManyRequests,
			headers:        map[string]string{"x-ms-ratelimit-microsoft.costmanagement-tenant-retry-after": "60"},
			wantRetryAfter: "60",
		},
		{
			name:           "client retry after is translated",
			status:         http.StatusTooManyRequests,
			headers:        map[string]string{"x-ms-ratelimit-microsoft.costmanagement-client-retry-after": "17"},
			wantRetryAfter: "17",
		},
		{
			name:           "consumption retry after is translated",
			status:         http.StatusTooManyRequests,
			headers:        map[string]string{"x-ms-ratelimit-microsoft.consumption-retry-after": "22"},
			wantRetryAfter: "22",
		},
		{
			// Header names as actually returned by the Consumption usageDetails API
			// (captured from a live run; the SDK canonicalises the casing this way).
			name:           "consumption clientappid retry after (observed live) is translated",
			status:         http.StatusTooManyRequests,
			headers:        map[string]string{"X-Ms-Ratelimit-Microsoft.consumption-Clientappid-Retry-After": "40"},
			wantRetryAfter: "40",
		},
		{
			name:           "consumption tenant retry after (observed live) is translated",
			status:         http.StatusTooManyRequests,
			headers:        map[string]string{"X-Ms-Ratelimit-Microsoft.consumption-Tenant-Retry-After": "55"},
			wantRetryAfter: "55",
		},
		{
			name:   "remaining quota headers observed live are not retry hints",
			status: http.StatusTooManyRequests,
			headers: map[string]string{
				"X-Ms-Ratelimit-Remaining-Microsoft.consumption-Clientappid-Requests": "14",
				"X-Ms-Ratelimit-Remaining-Microsoft.consumption-Tenant-Requests":      "9",
				"X-Ms-Ratelimit-Remaining-Subscription-Reads":                         "11999",
			},
			wantNoRetryAfter: true,
		},
		{
			name:   "the maximum of several headers wins",
			status: http.StatusTooManyRequests,
			headers: map[string]string{
				"x-ms-ratelimit-microsoft.costmanagement-qpu-retry-after":    "10",
				"x-ms-ratelimit-microsoft.costmanagement-entity-retry-after": "90",
				"x-ms-ratelimit-microsoft.costmanagement-tenant-retry-after": "30",
			},
			wantRetryAfter: "90",
		},
		{
			name:   "an existing Retry-After is not overwritten",
			status: http.StatusTooManyRequests,
			headers: map[string]string{
				"Retry-After": "5",
				"x-ms-ratelimit-microsoft.costmanagement-qpu-retry-after": "120",
			},
			wantRetryAfter: "5",
		},
		{
			name:   "an existing x-ms-retry-after-ms is not overwritten",
			status: http.StatusTooManyRequests,
			headers: map[string]string{
				"x-ms-retry-after-ms": "1500",
				"x-ms-ratelimit-microsoft.costmanagement-qpu-retry-after": "120",
			},
			wantNoRetryAfter: true,
		},
		{
			name:   "an existing Retry-After-Ms is not overwritten",
			status: http.StatusTooManyRequests,
			headers: map[string]string{
				"Retry-After-Ms": "1500",
				"x-ms-ratelimit-microsoft.costmanagement-qpu-retry-after": "120",
			},
			wantNoRetryAfter: true,
		},
		{
			name:             "invalid values are ignored",
			status:           http.StatusTooManyRequests,
			headers:          map[string]string{"x-ms-ratelimit-microsoft.costmanagement-qpu-retry-after": "soon"},
			wantNoRetryAfter: true,
		},
		{
			name:             "empty values are ignored",
			status:           http.StatusTooManyRequests,
			headers:          map[string]string{"x-ms-ratelimit-microsoft.costmanagement-qpu-retry-after": "   "},
			wantNoRetryAfter: true,
		},
		{
			name:             "non positive values are ignored",
			status:           http.StatusTooManyRequests,
			headers:          map[string]string{"x-ms-ratelimit-microsoft.costmanagement-qpu-retry-after": "-4"},
			wantNoRetryAfter: true,
		},
		{
			name:   "invalid values do not mask valid ones",
			status: http.StatusTooManyRequests,
			headers: map[string]string{
				"x-ms-ratelimit-microsoft.costmanagement-qpu-retry-after":    "nope",
				"x-ms-ratelimit-microsoft.costmanagement-entity-retry-after": "7",
			},
			wantRetryAfter: "7",
		},
		{
			name:           "fractional seconds are rounded up",
			status:         http.StatusTooManyRequests,
			headers:        map[string]string{"x-ms-ratelimit-microsoft.costmanagement-qpu-retry-after": "2.4"},
			wantRetryAfter: "3",
		},
		{
			name:           "hints above the retry cap are clamped so azcore retries instead of giving up",
			status:         http.StatusTooManyRequests,
			headers:        map[string]string{"x-ms-ratelimit-microsoft.costmanagement-qpu-retry-after": "3600"},
			wantRetryAfter: strconv.Itoa(int(billingMaxRetryDelay.Seconds())),
		},
		{
			name:           "absurdly large values are bounded, not overflowed",
			status:         http.StatusTooManyRequests,
			headers:        map[string]string{"x-ms-ratelimit-microsoft.costmanagement-qpu-retry-after": "1e300"},
			wantRetryAfter: strconv.Itoa(int(billingMaxRetryDelay.Seconds())),
		},
		{
			name:   "quota reporting headers are not retry hints",
			status: http.StatusTooManyRequests,
			headers: map[string]string{
				"x-ms-ratelimit-microsoft.costmanagement-qpu-consumed":  "1",
				"x-ms-ratelimit-microsoft.costmanagement-qpu-remaining": "11",
			},
			wantNoRetryAfter: true,
		},
		{
			name:             "successful responses are left alone",
			status:           http.StatusOK,
			headers:          map[string]string{"x-ms-ratelimit-microsoft.costmanagement-qpu-retry-after": "9"},
			wantNoRetryAfter: true,
		},
		{
			name:           "throttling reported as 503 is translated too",
			status:         http.StatusServiceUnavailable,
			headers:        map[string]string{"x-ms-ratelimit-microsoft.costmanagement-tenant-retry-after": "12"},
			wantRetryAfter: "12",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resp := doThroughPolicy(t, test.status, test.headers)

			got, found := headerValueInsensitive(resp.Header, "Retry-After")
			if test.wantNoRetryAfter {
				assert.False(t, found, "Retry-After must not be set, got %q", got)
				return
			}

			require.True(t, found, "Retry-After must be set")
			assert.Equal(t, test.wantRetryAfter, got)
		})
	}
}

func TestRetryAfterTranslationPolicyHandlesMissingHeaders(t *testing.T) {
	// A response with no headers at all must not panic nor gain a Retry-After.
	resp := doThroughPolicy(t, http.StatusTooManyRequests, nil)

	_, found := headerValueInsensitive(resp.Header, "Retry-After")
	assert.False(t, found)
}

func TestNewRetryOptionsKeepsAzcoreDefaultStatusCodes(t *testing.T) {
	options := newRetryOptions()

	assert.Nil(t, options.StatusCodes,
		"StatusCodes must stay nil so azcore keeps its defaults, which include 429")
	assert.Equal(t, billingMaxRetries, options.MaxRetries)
	assert.Equal(t, billingRetryDelay, options.RetryDelay)
	assert.Equal(t, billingMaxRetryDelay, options.MaxRetryDelay)

	// A per-minute quota must be able to clear within a single retry delay.
	assert.GreaterOrEqual(t, options.MaxRetryDelay, time.Minute,
		"MaxRetryDelay must exceed the Cost Management per-minute window, "+
			"otherwise azcore abandons the request instead of waiting")
}

func TestNewClientOptionsWiresBothPolicies(t *testing.T) {
	options := newClientOptions(azure.Config{
		ActiveDirectoryEndpoint: "https://login.microsoftonline.com/",
	}, logptest.NewTestingLogger(t, ""))

	require.Len(t, options.PerCallPolicies, 1, "the ClientType policy must run once per request")
	require.IsType(t, clientTypePolicy{}, options.PerCallPolicies[0])

	require.Len(t, options.PerRetryPolicies, 1, "the Retry-After translation must run on every try")
	require.IsType(t, &retryAfterTranslationPolicy{}, options.PerRetryPolicies[0])

	assert.Equal(t, newRetryOptions(), options.Retry)
	assert.NotEmpty(t, options.Cloud.Services, "the cloud configuration must be preserved")
}

// TestRetryAfterTranslationEndToEnd drives a full azcore pipeline, configured exactly
// like the billing clients, against a transport that throttles the first try with only
// a Cost Management rate limit header. It asserts the request is retried and succeeds,
// and that ClientType is stamped on every try.
func TestRetryAfterTranslationEndToEnd(t *testing.T) {
	transport := &stubTransport{
		responses: []func(*http.Request) *http.Response{
			// First try: throttled, with a hint azcore alone cannot read.
			response(http.StatusTooManyRequests, map[string]string{
				"x-ms-ratelimit-microsoft.costmanagement-qpu-retry-after": "1",
			}),
			// Second try: success.
			response(http.StatusOK, nil),
		},
	}

	clientOptions := newClientOptions(azure.Config{
		ActiveDirectoryEndpoint: "https://login.microsoftonline.com/",
	}, logptest.NewTestingLogger(t, ""))
	clientOptions.Transport = transport
	// Keep the test fast: the hint is 1s, so no exponential fallback is needed, but
	// cap the delay well below the 5m production ceiling as a safety net.
	clientOptions.Retry.MaxRetryDelay = 5 * time.Second

	pipeline := runtime.NewPipeline("azurebillingtest", "v0.0.0", runtime.PipelineOptions{}, &clientOptions.ClientOptions)

	req, err := runtime.NewRequest(context.Background(), http.MethodGet, testEndpoint)
	require.NoError(t, err)

	start := time.Now()
	resp, err := pipeline.Do(req)
	elapsed := time.Since(start)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "the retried request must succeed")
	assert.Equal(t, 2, transport.requestCount(), "the throttled request must be retried exactly once")

	// The wait must come from the translated header (~1s), not from azcore's
	// exponential fallback, whose first delay would be at least 8s (0.8 * 10s).
	assert.Less(t, elapsed, 5*time.Second,
		"the retry must have used the 1s service hint, not the exponential fallback")
	assert.GreaterOrEqual(t, elapsed, 900*time.Millisecond,
		"the retry must have honoured the 1s service hint")

	requests := transport.recordedRequests()
	require.Len(t, requests, 2)
	for i, header := range requests {
		value, found := headerValueInsensitive(header, clientTypeHeaderName)
		assert.True(t, found, "request %d must carry a %s header", i, clientTypeHeaderName)
		assert.Equal(t, clientTypeHeaderValue, value, "request %d must identify this caller", i)
	}
}

// TestExponentialFallbackWithoutHint documents that a 429 carrying no usable hint
// still falls back to azcore's exponential backoff, seeded with our larger base delay.
func TestExponentialFallbackWithoutHint(t *testing.T) {
	transport := &stubTransport{
		responses: []func(*http.Request) *http.Response{
			response(http.StatusTooManyRequests, nil),
			response(http.StatusOK, nil),
		},
	}

	clientOptions := newClientOptions(azure.Config{
		ActiveDirectoryEndpoint: "https://login.microsoftonline.com/",
	}, logptest.NewTestingLogger(t, ""))
	clientOptions.Transport = transport
	// Shrink the production delays so the test does not sleep for 10s.
	clientOptions.Retry.RetryDelay = 10 * time.Millisecond
	clientOptions.Retry.MaxRetryDelay = 100 * time.Millisecond

	pipeline := runtime.NewPipeline("azurebillingtest", "v0.0.0", runtime.PipelineOptions{}, &clientOptions.ClientOptions)

	req, err := runtime.NewRequest(context.Background(), http.MethodGet, testEndpoint)
	require.NoError(t, err)

	resp, err := pipeline.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 2, transport.requestCount(), "429 must be retried even without a hint")
}
