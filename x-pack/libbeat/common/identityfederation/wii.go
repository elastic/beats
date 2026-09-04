// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package identityfederation

import (
	"bytes"
	"context"
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
	WIIIssuerURLEnvVar   = "WORKLOAD_IDENTITY_ISSUER_URL"
	WIISSLCertFileEnvVar = "WORKLOAD_IDENTITY_SSL_CERT_FILE"
	WIISSLKeyFileEnvVar  = "WORKLOAD_IDENTITY_SSL_KEY_FILE"
	WIISSLCAFileEnvVar   = "WORKLOAD_IDENTITY_SSL_CA_FILE"
)

const (
	wiiConnectTimeout      = 5 * time.Second
	wiiRequestTimeout      = 10 * time.Second
	wiiMaxResponseSize     = 1 << 20 // 1 MiB
	wiiRetryInitialDelay   = 200 * time.Millisecond
	wiiRetryMaxDelay       = 5 * time.Second
	wiiRetryMaxAttempts    = 3
	wiiRefreshBeforeExpiry = time.Minute
	wiiMaxTokenLifetime    = 365 * 24 * time.Hour // epoch-millis guard
)

// WIITokenSource fetches JWTs from the workload-identity-issuer over mTLS.
//
//	POST <issuer-url>/token  {"aud": "<audience>"}
//	200 OK                   {"token": "<jwt>", "expires_at": <epoch-seconds>}
type WIITokenSource struct {
	tokenEndpoint string
	certFile      string
	keyFile       string
	audience      string

	mu           sync.Mutex
	cachedToken  []byte
	cachedExpiry time.Time
}

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

type wiiHTTPError struct {
	status int
}

func (e *wiiHTTPError) Error() string {
	return fmt.Sprintf("workload-identity issuer returned HTTP %d", e.status)
}

func isRetryableWIIError(err error) bool {
	var httpErr *wiiHTTPError
	if errors.As(err, &httpErr) {
		return httpErr.status == http.StatusTooManyRequests || (httpErr.status >= 500 && httpErr.status <= 599)
	}
	var urlErr *url.Error
	return errors.As(err, &urlErr)
}

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
		time.Sleep(rand.N(delayBound) + 1)
		delayBound = min(delayBound*2, wiiRetryMaxDelay)
	}
}

func (w *WIITokenSource) fetchToken() ([]byte, time.Time, error) {
	tlsCfg := &tls.Config{
		GetClientCertificate: func(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
			cert, err := tls.LoadX509KeyPair(w.certFile, w.keyFile)
			if err != nil {
				return nil, fmt.Errorf("loading WII client cert from %s: %w", w.certFile, err)
			}
			return &cert, nil
		},
		MinVersion: tls.VersionTLS12,
	}

	if caFile := os.Getenv(WIISSLCAFileEnvVar); caFile != "" {
		caBytes, caErr := os.ReadFile(caFile) //nolint:gosec // G703: the path is operator-provided configuration, not untrusted input
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
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, w.tokenEndpoint, bytes.NewReader(body))
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
