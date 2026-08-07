// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package identityfederation

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	// WIIIssuerURLEnvVar is the env var set by the agentless controller for the
	// workload-identity-issuer URL. Its presence acts as a feature gate: when set,
	// beats uses WII mTLS token exchange instead of the legacy
	// CLOUD_CONNECTORS_ID_TOKEN_FILE OIDC path. This mirrors Elasticsearch, where
	// the workload-identity module activates on a non-blank
	// workload_identity.issuer.url setting.
	WIIIssuerURLEnvVar = "WORKLOAD_IDENTITY_ISSUER_URL"
	// WIISSLCertFileEnvVar is the env var pointing to the WII client certificate file.
	WIISSLCertFileEnvVar = "WORKLOAD_IDENTITY_SSL_CERT_FILE"
	// WIISSLKeyFileEnvVar is the env var pointing to the WII client private key file.
	WIISSLKeyFileEnvVar = "WORKLOAD_IDENTITY_SSL_KEY_FILE"
	// WIISSLCAFileEnvVar is the optional env var pointing to a CA bundle used to verify
	// the WII server's TLS certificate. When not set, the system root CAs are used —
	// the production issuer endpoint presents a publicly trusted certificate, so this
	// is only needed for dev/test setups with private server CAs.
	WIISSLCAFileEnvVar = "WORKLOAD_IDENTITY_SSL_CA_FILE"
)

// Transport and retry tuning, matching the defaults of Elasticsearch's
// workload-identity module (WorkloadIdentityHttpSettings / WorkloadIdentityIssuerSettings).
const (
	wiiConnectTimeout      = 5 * time.Second
	wiiRequestTimeout      = 10 * time.Second
	wiiMaxResponseSize     = 1 << 20 // 1 MiB
	wiiRetryInitialDelay   = 200 * time.Millisecond
	wiiRetryMaxDelay       = 5 * time.Second
	wiiRetryMaxAttempts    = 3
	wiiRefreshBeforeExpiry = time.Minute
	// wiiMaxTokenLifetime is a sanity ceiling on expires_at, guarding against
	// response-encoding bugs (e.g. epoch millis read as epoch seconds). Not a policy
	// bound on issuer lifetime.
	wiiMaxTokenLifetime = 365 * 24 * time.Hour
)

// WIITokenSource obtains short-lived JWTs from the workload-identity-issuer
// POST /token endpoint over mTLS, using the client certificate provisioned by the
// agentless controller. It implements stscreds.IdentityTokenRetriever and is the Go
// equivalent of Elasticsearch's HttpsWorkloadIdentityIssuerClient.
//
// Wire contract (same as Elasticsearch):
//
//	POST <issuer-url>/token
//	Content-Type: application/json
//	{"aud": "<audience>"}
//
//	200 OK
//	{"token": "<jwt>", "expires_at": <epoch-seconds>}
//
// The audience is opaque to the issuer and copied verbatim into the JWT aud claim:
//   - AWS:   "sts.amazonaws.com"
//   - GCP:   the WIF provider URL ("//iam.googleapis.com/projects/…/providers/…")
//   - Azure: the Azure AD audience (e.g. "api://AzureADTokenExchange")
//
// Tokens are cached until expires_at minus a refresh margin; concurrent callers share
// a single in-flight fetch. Failures are never cached. Transient failures (HTTP 429,
// 5xx, network errors) are retried with jittered exponential backoff; other HTTP
// errors are not. The client certificate is re-read from disk on every fetch so
// rotation by the controller is picked up without a restart.
type WIITokenSource struct {
	tokenEndpoint string
	certFile      string
	keyFile       string
	audience      string

	// mu serializes fetches: the holder performs the HTTP exchange while
	// concurrent callers block and then serve from the refreshed cache.
	mu           sync.Mutex
	cachedToken  []byte
	cachedExpiry time.Time
}

// NewWIITokenSource creates a WIITokenSource for the given audience. The issuer URL
// must be https, include a host, and carry no query string or fragment (the same
// validation Elasticsearch applies at node startup, surfaced here at configuration
// time rather than at the first token request).
func NewWIITokenSource(issuerURL, certFile, keyFile, audience string) (*WIITokenSource, error) {
	endpoint, err := resolveWIITokenEndpoint(issuerURL)
	if err != nil {
		return nil, err
	}
	return &WIITokenSource{
		tokenEndpoint: endpoint,
		certFile:      certFile,
		keyFile:       keyFile,
		audience:      audience,
	}, nil
}

// resolveWIITokenEndpoint validates the issuer base URL and appends the /token path.
func resolveWIITokenEndpoint(issuerURL string) (string, error) {
	if issuerURL == "" {
		return "", errors.New("workload-identity issuer URL must be configured")
	}
	parsed, err := url.Parse(issuerURL)
	if err != nil {
		return "", fmt.Errorf("workload-identity issuer URL %q is not a valid URL: %w", issuerURL, err)
	}
	if !strings.EqualFold(parsed.Scheme, "https") {
		return "", fmt.Errorf("workload-identity issuer URL %q must use the https scheme", issuerURL)
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("workload-identity issuer URL %q must include a host", issuerURL)
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("workload-identity issuer URL %q must not include a query string or fragment", issuerURL)
	}
	return strings.TrimRight(issuerURL, "/") + "/token", nil
}

// GetIdentityToken implements stscreds.IdentityTokenRetriever. It returns the cached
// JWT when it is still fresh (expires_at minus the refresh margin), otherwise fetches
// a new one from the issuer.
func (w *WIITokenSource) GetIdentityToken() ([]byte, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.cachedToken != nil && time.Now().Before(w.cachedExpiry.Add(-wiiRefreshBeforeExpiry)) {
		return w.cachedToken, nil
	}

	token, expiresAt, err := w.fetchTokenWithRetry()
	if err != nil {
		return nil, err
	}
	w.cachedToken = token
	w.cachedExpiry = expiresAt
	return token, nil
}

// wiiHTTPError carries the issuer's HTTP status so the retry policy can
// distinguish transient statuses from configuration/credential errors.
type wiiHTTPError struct {
	status int
}

func (e *wiiHTTPError) Error() string {
	// The response body is deliberately kept out of the error (it is not under our
	// control and may end up in logs); only the status is surfaced.
	return fmt.Sprintf("workload-identity issuer returned HTTP %d", e.status)
}

// isRetryableWIIError reports whether a subsequent attempt could plausibly succeed:
// transport-level errors and the issuer-reported transient statuses (429 plus any
// 5xx). Other HTTP statuses and response parse/validation failures are not retried.
// Same policy as Elasticsearch's HttpsWorkloadIdentityIssuerClient.isRetryable.
func isRetryableWIIError(err error) bool {
	var httpErr *wiiHTTPError
	if errors.As(err, &httpErr) {
		return httpErr.status == http.StatusTooManyRequests || (httpErr.status >= 500 && httpErr.status <= 599)
	}
	var urlErr *url.Error
	return errors.As(err, &urlErr)
}

// fetchTokenWithRetry runs fetchToken with up to wiiRetryMaxAttempts attempts,
// sleeping a jittered exponential backoff (initial wiiRetryInitialDelay, capped at
// wiiRetryMaxDelay) between retryable failures.
func (w *WIITokenSource) fetchTokenWithRetry() ([]byte, time.Time, error) {
	var errs []error
	delayBound := wiiRetryInitialDelay
	for attempt := 1; ; attempt++ {
		token, expiresAt, err := w.fetchToken()
		if err == nil {
			return token, expiresAt, nil
		}
		errs = append(errs, err)
		if attempt >= wiiRetryMaxAttempts || !isRetryableWIIError(err) {
			return nil, time.Time{}, fmt.Errorf(
				"fetching workload-identity token from %s (%d attempts): %w",
				w.tokenEndpoint, attempt, errors.Join(errs...))
		}
		// Uniform jitter within the current bound, then double the bound.
		time.Sleep(rand.N(delayBound) + 1)
		delayBound = min(delayBound*2, wiiRetryMaxDelay)
	}
}

// fetchToken performs a single POST /token exchange.
func (w *WIITokenSource) fetchToken() ([]byte, time.Time, error) {
	// Re-read the cert from disk on every fetch to pick up controller-driven rotation.
	cert, err := tls.LoadX509KeyPair(w.certFile, w.keyFile)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("loading WII client cert from %s: %w", w.certFile, err)
	}

	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	// WORKLOAD_IDENTITY_SSL_CA_FILE overrides the server CA trust — only used when the
	// issuer endpoint presents a cert from a private CA (dev/test setups).
	if caFile := os.Getenv(WIISSLCAFileEnvVar); caFile != "" {
		caBytes, caErr := os.ReadFile(caFile)
		if caErr != nil {
			return nil, time.Time{}, fmt.Errorf("reading WII CA file %s: %w", caFile, caErr)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caBytes) {
			return nil, time.Time{}, fmt.Errorf("WII CA file %s contains no valid PEM certificates", caFile)
		}
		tlsCfg.RootCAs = pool
	}

	client := &http.Client{
		Timeout: wiiRequestTimeout,
		Transport: &http.Transport{
			TLSClientConfig: tlsCfg,
			DialContext: (&net.Dialer{
				Timeout: wiiConnectTimeout,
			}).DialContext,
		},
	}

	body, err := json.Marshal(map[string]string{"aud": w.audience})
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("encoding WII token request: %w", err)
	}
	req, err := http.NewRequest(http.MethodPost, w.tokenEndpoint, strings.NewReader(string(body)))
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("creating WII token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("calling WII token endpoint: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, wiiMaxResponseSize))
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("reading WII token response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, time.Time{}, &wiiHTTPError{status: resp.StatusCode}
	}

	return parseWIITokenResponse(responseBody)
}

// parseWIITokenResponse parses the issuer's JSON response. Both fields are required
// (Elasticsearch's parser rejects responses missing either); unknown fields are
// tolerated so the issuer can add fields without breaking clients.
func parseWIITokenResponse(body []byte) ([]byte, time.Time, error) {
	var envelope struct {
		Token     string `json:"token"`
		ExpiresAt *int64 `json:"expires_at"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, time.Time{}, fmt.Errorf("parsing WII token response: %w", err)
	}
	if envelope.Token == "" {
		return nil, time.Time{}, errors.New("WII token response is missing the token field")
	}
	if envelope.ExpiresAt == nil {
		return nil, time.Time{}, errors.New("WII token response is missing the expires_at field")
	}

	expiresAt := time.Unix(*envelope.ExpiresAt, 0)
	now := time.Now()
	if expiresAt.Before(now) {
		return nil, time.Time{}, fmt.Errorf("WII token expires_at %s is in the past", expiresAt.UTC().Format(time.RFC3339))
	}
	if expiresAt.Sub(now) > wiiMaxTokenLifetime {
		return nil, time.Time{}, fmt.Errorf("WII token expires_at %s exceeds the maximum acceptable lifetime of %s",
			expiresAt.UTC().Format(time.RFC3339), wiiMaxTokenLifetime)
	}

	return []byte(envelope.Token), expiresAt, nil
}
