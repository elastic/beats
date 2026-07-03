// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azureblobstorage

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/management/status"
	beattest "github.com/elastic/beats/v7/libbeat/publisher/testing"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/azureblobstorage/mock"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

// customTransporter implements the Transporter interface with a custom Do & RoundTrip method
type customTransporter struct {
	rt      http.RoundTripper
	servURL string
}

func (t *customTransporter) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.rt.RoundTrip(req)
}

// Do is responsible for the routing of the request to the appropriate handler based on the request URL
func (t *customTransporter) Do(req *http.Request) (*http.Response, error) {
	logp.L().Named("azure-blob-storage-test").Debug("request URL: ", req.URL)

	// Intercept calls to Azure AD endpoints and route them to our test server
	testURL, _ := url.Parse(t.servURL)
	if req.URL.Host == "login.microsoftonline.com" || req.URL.Host == testURL.Host {
		// Handle Azure AD endpoint patterns:
		// /{tenant-id}/v2.0/.well-known/openid-configuration
		// /{tenant-id}/oauth2/v2.0/token
		re := regexp.MustCompile(`^/([0-9a-fA-F-]+|common)/?(oauth2/v2\.0/token|v2\.0/\.well-known/openid-configuration)`)
		matches := re.FindStringSubmatch(req.URL.Path)

		if len(matches) == 3 {
			tenant_id := matches[1]
			action := matches[2]

			switch action {
			case "v2.0/.well-known/openid-configuration":
				return createJSONResponse(map[string]interface{}{
					"token_endpoint":         "https://login.microsoftonline.com/" + tenant_id + "/oauth2/v2.0/token",
					"authorization_endpoint": "https://login.microsoftonline.com/" + tenant_id + "/oauth2/v2.0/authorize",
					"issuer":                 "https://login.microsoftonline.com/" + tenant_id + "/v2.0",
				}, 200)

			case "oauth2/v2.0/token":
				return createJSONResponse(map[string]interface{}{
					"token_type":   "Bearer",
					"expires_in":   3600,
					"access_token": "mock_access_token_123",
				}, 200)
			}
		}
	}

	// Fall back to original behavior for other requests (Azure Storage)
	return t.rt.RoundTrip(req)
}

func createJSONResponse(data interface{}, statusCode int) (*http.Response, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	resp := &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(bytes.NewBuffer(jsonData)),
		Header:     make(http.Header),
	}

	resp.Header.Set("Content-Type", "application/json")
	return resp, nil
}

// flakyTransporter wraps customTransporter and answers the first failListBlobs
// "list blobs" requests with a synthetic Azure "503 ServerBusy" before letting
// requests through. It reproduces the transient throttling from sdh-beats#7324
// so we can assert that blob listing (pagination) is now retried.
type flakyTransporter struct {
	inner         *customTransporter
	failListBlobs int

	mu           sync.Mutex
	listAttempts int
}

func (t *flakyTransporter) Do(req *http.Request) (*http.Response, error) {
	// The list-blobs call is the pagination request (GET on the container with
	// comp=list); blob downloads use a different path and are left untouched.
	if req.URL.Query().Get("comp") == "list" {
		t.mu.Lock()
		shouldFail := t.listAttempts < t.failListBlobs
		if shouldFail {
			t.listAttempts++
		}
		t.mu.Unlock()
		if shouldFail {
			return serverBusyResponse(req), nil
		}
	}
	return t.inner.Do(req)
}

// attempts reports how many list-blobs requests have been observed so far,
// under the same lock used to record them.
func (t *flakyTransporter) attempts() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.listAttempts
}

func serverBusyResponse(req *http.Request) *http.Response {
	const body = `<?xml version="1.0" encoding="utf-8"?><Error><Code>ServerBusy</Code><Message>The server is busy.</Message></Error>`
	h := make(http.Header)
	h.Set("Content-Type", "application/xml")
	return &http.Response{
		StatusCode: http.StatusServiceUnavailable,
		Status:     "503 The server is busy.",
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     h,
		Request:    req,
	}
}

// Test_ListBlobsRetriesOnTransientError verifies that a transient 503 during
// blob listing no longer kills the input: with retries configured, the pager
// recovers and every expected blob is still ingested.
func Test_ListBlobsRetriesOnTransientError(t *testing.T) {
	logp.TestingSetup()

	serv := httptest.NewServer(mock.AzureStorageServer())
	t.Cleanup(serv.Close)

	// Fail the first three list attempts, then succeed. With max_retries: 10 the
	// pager has enough attempts left to get through the throttling.
	transport := &flakyTransporter{
		inner:         &customTransporter{rt: http.DefaultTransport, servURL: serv.URL},
		failListBlobs: 3,
	}

	baseConfig := map[string]interface{}{
		"account_name": "beatsblobnew",
		"auth.oauth2": map[string]interface{}{
			"client_id":     "12345678-90ab-cdef-1234-567890abcdef",
			"client_secret": "abcdefg1234567890!@#$%^&*()-_=+",
			"tenant_id":     "87654321-abcd-ef90-1234-fedcba098765",
		},
		"max_workers": 2,
		"poll":        false,
		"containers": []map[string]interface{}{
			{"name": beatsContainer},
		},
		// Keep the backoff tiny so the test stays fast.
		"retry": map[string]interface{}{
			"max_retries":         10,
			"initial_retry_delay": "1ms",
			"max_retry_delay":     "5ms",
		},
	}
	expected := map[string]bool{
		mock.Beatscontainer_blob_ata_json:      true,
		mock.Beatscontainer_blob_data3_json:    true,
		mock.Beatscontainer_blob_docs_ata_json: true,
	}

	cfg := conf.MustNewConfigFrom(baseConfig)
	c := config{}
	require.NoError(t, cfg.Unpack(&c))

	// inject the flaky transport; the retry policy is wired from c.Retry.
	c.Auth.OAuth2.clientOptions = azcore.ClientOptions{
		InsecureAllowCredentialWithHTTP: true,
		Transport:                       transport,
	}

	input := newStatelessInput(c, serv.URL+"/", logp.NewNopLogger())
	assert.NoError(t, input.Test(v2.TestContext{}), "input.Test should succeed")

	chanClient := beattest.NewChanClient(len(expected))
	t.Cleanup(func() { _ = chanClient.Close() })

	ctx, cancel := newV2Context(t)
	t.Cleanup(cancel)
	ctx.ID += "-retry"

	var g errgroup.Group
	g.Go(func() error {
		return input.Run(ctx, chanClient)
	})

	timeout := time.NewTimer(10 * time.Second)
	t.Cleanup(func() { timeout.Stop() })

	var receivedCount int
wait:
	for {
		select {
		case <-timeout.C:
			t.Errorf("timed out waiting for %d events after transient list failures", len(expected))
			cancel()
			break wait
		case got := <-chanClient.Channel:
			val, err := got.Fields.GetValue("message")
			assert.NoError(t, err, "published event should carry a message field")
			assert.True(t, expected[val.(string)], "received unexpected message: %v", val)
			receivedCount++
			if receivedCount == len(expected) {
				cancel()
				break wait
			}
		}
	}

	// Wait for the input to finish so the transport goroutines are done before
	// we inspect the attempt counter.
	assert.NoError(t, g.Wait(), "input run should complete without error")
	assert.GreaterOrEqual(t, transport.attempts(), 3,
		"the list-blobs request should have been retried past the transient 503s")
}

// recordingStatusReporter captures the statuses reported by the input so tests
// can assert on the transitions.
type recordingStatusReporter struct {
	mu       sync.Mutex
	statuses []status.Status
}

func (r *recordingStatusReporter) UpdateStatus(s status.Status, _ string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.statuses = append(r.statuses, s)
}

func (r *recordingStatusReporter) has(target status.Status) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, s := range r.statuses {
		if s == target {
			return true
		}
	}
	return false
}

// Test_ListBlobsNonFatalWhilePolling verifies that a sustained listing failure
// (throttling that outlives the SDK's per-request retries) is non-fatal when
// polling: the input is marked Degraded and keeps running so it can recover on
// a later poll. Without polling there is no next cycle, so the failure surfaces
// and the input stops.
func Test_ListBlobsNonFatalWhilePolling(t *testing.T) {
	logp.TestingSetup()

	// A server that always answers with 503 ServerBusy, simulating a sustained
	// Azure Storage outage.
	serv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = io.WriteString(w, `<?xml version="1.0" encoding="utf-8"?><Error><Code>ServerBusy</Code><Message>The server is busy.</Message></Error>`)
	}))
	t.Cleanup(serv.Close)

	baseConfig := map[string]interface{}{
		"account_name":                        "beatsblobnew",
		"auth.shared_credentials.account_key": "7pfLm1betGiRyyABEM/RFrLYlafLZHbLtGhB52LkWVeBxE7la9mIvk6YYAbQKYE/f0GdhiaOZeV8+AStsAdr/Q==",
		"max_workers":                         1,
		"containers": []map[string]interface{}{
			{"name": beatsContainer},
		},
		// Keep the retry backoff tiny so the failing list call returns quickly.
		"retry": map[string]interface{}{
			"max_retries":         1,
			"initial_retry_delay": "1ms",
			"max_retry_delay":     "5ms",
		},
	}

	newSched := func(t *testing.T, poll bool, statusRec status.StatusReporter) *scheduler {
		t.Helper()
		cfg := conf.MustNewConfigFrom(baseConfig)
		c := defaultConfig()
		require.NoError(t, cfg.Unpack(&c), "config should unpack")

		log := logp.NewNopLogger()
		serviceClient, credential, err := fetchServiceClientAndCreds(c, c.Retry, serv.URL+"/", log)
		require.NoError(t, err, "service client should be created")
		containerClient, err := fetchContainerClient(serviceClient, beatsContainer, log)
		require.NoError(t, err, "container client should be created")

		src := &Source{
			AccountName:   c.AccountName,
			ContainerName: beatsContainer,
			MaxWorkers:    1,
			Poll:          poll,
			PollInterval:  time.Millisecond,
			Retry:         c.Retry,
		}
		return newScheduler(&publisher{}, containerClient, credential, src, &c, newState(), serv.URL+"/", statusRec, nil, log)
	}

	t.Run("polling keeps the input alive and Degraded", func(t *testing.T) {
		rec := &recordingStatusReporter{}
		sched := newSched(t, true, rec)

		err := sched.scheduleOnce(context.Background())
		assert.NoError(t, err, "a listing failure while polling must be non-fatal so the input keeps running")
		assert.True(t, rec.has(status.Degraded), "the input should be marked Degraded after a listing failure")
		assert.False(t, rec.has(status.Failed), "a polling listing failure must not mark the input Failed")
	})

	t.Run("without polling the failure is fatal", func(t *testing.T) {
		rec := &recordingStatusReporter{}
		sched := newSched(t, false, rec)

		err := sched.scheduleOnce(context.Background())
		assert.Error(t, err, "without polling there is no recovery, so the listing failure must surface")
		assert.True(t, rec.has(status.Failed), "a one-shot listing failure should mark the input Failed")
	})
}

func Test_OAuth2(t *testing.T) {
	tests := []struct {
		name        string
		baseConfig  map[string]interface{}
		mockHandler func() http.Handler
		expected    map[string]bool
	}{
		{
			name: "OAuth2TConfig",
			baseConfig: map[string]interface{}{
				"account_name": "beatsblobnew",
				"auth.oauth2": map[string]interface{}{
					"client_id":     "12345678-90ab-cdef-1234-567890abcdef",
					"client_secret": "abcdefg1234567890!@#$%^&*()-_=+",
					"tenant_id":     "87654321-abcd-ef90-1234-fedcba098765",
				},
				"max_workers":   2,
				"poll":          true,
				"poll_interval": "30s",
				"containers": []map[string]interface{}{
					{
						"name": beatsContainer,
					},
				},
			},
			mockHandler: mock.AzureStorageServer,
			expected: map[string]bool{
				mock.Beatscontainer_blob_ata_json:      true,
				mock.Beatscontainer_blob_data3_json:    true,
				mock.Beatscontainer_blob_docs_ata_json: true,
			},
		},
	}

	logp.TestingSetup()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serv := httptest.NewServer(tt.mockHandler())
			t.Cleanup(serv.Close)

			httpClient := &http.Client{
				Transport: &customTransporter{
					rt:      http.DefaultTransport,
					servURL: serv.URL,
				},
			}

			cfg := conf.MustNewConfigFrom(tt.baseConfig)
			conf := config{}
			err := cfg.Unpack(&conf)
			assert.NoError(t, err)

			// inject custom transport & client options
			conf.Auth.OAuth2.clientOptions = azcore.ClientOptions{
				InsecureAllowCredentialWithHTTP: true,
				Transport:                       httpClient.Transport.(*customTransporter),
			}

			input := newStatelessInput(conf, serv.URL+"/", logp.NewNopLogger())

			assert.Equal(t, "azure-blob-storage-stateless", input.Name())
			assert.NoError(t, input.Test(v2.TestContext{}))

			chanClient := beattest.NewChanClient(len(tt.expected))
			t.Cleanup(func() { _ = chanClient.Close() })

			ctx, cancel := newV2Context(t)
			t.Cleanup(cancel)
			ctx.ID += tt.name

			var g errgroup.Group
			g.Go(func() error {
				return input.Run(ctx, chanClient)
			})

			var timeout *time.Timer
			if conf.PollInterval != nil {
				timeout = time.NewTimer(1*time.Second + *conf.PollInterval)
			} else {
				timeout = time.NewTimer(10 * time.Second)
			}
			t.Cleanup(func() { timeout.Stop() })

			var receivedCount int
		wait:
			for {
				select {
				case <-timeout.C:
					t.Errorf("timed out waiting for %d events", len(tt.expected))
					cancel()
					return
				case got := <-chanClient.Channel:
					var val interface{}
					var err error
					val, err = got.Fields.GetValue("message")
					assert.NoError(t, err)
					assert.True(t, tt.expected[val.(string)])
					receivedCount += 1
					if receivedCount == len(tt.expected) {
						cancel()
						break wait
					}
				}
			}
		})
	}
}
