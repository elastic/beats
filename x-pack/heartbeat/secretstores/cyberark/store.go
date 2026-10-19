// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Package cyberark implements the "cyberark" secret store for Heartbeat
// monitors: CyberArk Central Credential Provider (CCP / AIMWebService)
// accounts, read over HTTP/HTTPS with mTLS client certificates, token, or
// Basic authentication.
//
// References have the form cyberark://<safe>/<object>#<field> or
// cyberark://<safe>/<folder>/<object>#<field>. Segments and field are
// percent-decoded.
package cyberark

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hashicorp/go-cleanhttp"

	"github.com/elastic/beats/v7/heartbeat/monitors/secretstores"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

// storeType is the secret_stores type and the reference prefix.
const storeType = "cyberark"

const (
	requestTimeout  = 10 * time.Second
	maxResponseSize = 1 << 20
)

// fieldAliases maps common lowercase field names to their CyberArk CCP
// response property names when the lowercase name is absent from the response.
var fieldAliases = map[string]string{
	"password": "Content",
	"username": "UserName",
	"address":  "Address",
	"database": "Database",
}

func init() {
	if err := secretstores.Register(storeType, newStore); err != nil {
		panic(err)
	}
}

type store struct {
	cfg    config
	client *http.Client
	logger *logp.Logger
	reader *secretstores.Reader[accountRef]
}

func newStore(cfg *conf.C, logger *logp.Logger) (secretstores.Store, error) {
	c := defaultConfig()
	if err := cfg.Unpack(&c); err != nil {
		return nil, err
	}
	if err := c.validate(); err != nil {
		return nil, err
	}

	client, err := newClient(c, logger)
	if err != nil {
		return nil, err
	}

	st := &store{cfg: c, client: client, logger: logger.Named(storeType)}
	st.reader = secretstores.NewReader[accountRef](st.logger)
	st.reader.Name = storeType
	st.reader.Address = c.Address
	st.reader.Noun = "account"
	st.reader.Describe = accountRef.String
	st.reader.Fetch = st.fetch
	return st, nil
}

func newClient(c config, logger *logp.Logger) (*http.Client, error) {
	tlsConfig, err := clientTLSConfig(c, logger)
	if err != nil {
		return nil, err
	}
	transport := cleanhttp.DefaultPooledTransport()
	transport.TLSClientConfig = tlsConfig

	return &http.Client{
		Transport: transport,
		Timeout:   requestTimeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}, nil
}

func clientTLSConfig(c config, logger *logp.Logger) (*tls.Config, error) {
	return secretstores.ClientTLSConfig(c.SSL, c.Address, logger)
}

func (s *store) resolve(ctx context.Context, content string) (string, error) {
	ref, field, err := parseReference(content)
	if err != nil {
		return "", fmt.Errorf("cyberark: invalid reference: %w", err)
	}
	data, err := s.reader.Read(ctx, ref)
	if err != nil {
		return "", err
	}
	v, ok := lookupField(data, field)
	if !ok {
		return "", fmt.Errorf("cyberark: field %q not found in account %s", field, ref)
	}
	return secretstores.StringValue(storeType, v)
}

func lookupField(data map[string]any, field string) (any, bool) {
	if v, ok := data[field]; ok {
		return v, true
	}
	if canonical, ok := fieldAliases[field]; ok {
		v, found := data[canonical]
		return v, found
	}
	return nil, false
}

func (s *store) fetch(ctx context.Context, ref accountRef) (map[string]any, error) {
	reqURL, err := s.accountURL(ref)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("cyberark: could not build request for %s: %w", ref, err)
	}
	req.Header.Set("Accept", "application/json")
	if s.cfg.Auth.Token != "" {
		req.Header.Set("Authorization", "Bearer "+s.cfg.Auth.Token)
	} else if s.cfg.Auth.Basic != nil {
		req.SetBasicAuth(s.cfg.Auth.Basic.Username, s.cfg.Auth.Basic.Password)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, requestError(err, "reading "+ref.String())
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return nil, requestError(err, "reading "+ref.String())
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("cyberark: no account at %s", ref)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, responseError(resp.StatusCode, body, "reading "+ref.String())
	}

	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	var data map[string]any
	if err := dec.Decode(&data); err != nil {
		return nil, fmt.Errorf("cyberark: reading %s failed: invalid JSON response: %w", ref, err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("cyberark: no account at %s", ref)
	}
	return data, nil
}

func (s *store) accountURL(ref accountRef) (string, error) {
	u, err := url.Parse(strings.TrimRight(s.cfg.Address, "/") + s.cfg.ServicePath)
	if err != nil {
		return "", fmt.Errorf("cyberark: invalid address: %w", err)
	}
	q := u.Query()
	q.Set("AppID", s.cfg.AppID)
	q.Set("Safe", ref.safe)
	if ref.folder != "" {
		q.Set("Folder", ref.folder)
	}
	q.Set("Object", ref.object)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// ccpError is the JSON error payload returned by CyberArk CCP AIMWebService.
type ccpError struct {
	ErrorCode string `json:"ErrorCode"`
	ErrorMsg  string `json:"ErrorMsg"`
}

func responseError(statusCode int, body []byte, msg string) error {
	cause := fmt.Sprintf("status %d", statusCode)
	var apiErr ccpError
	if err := json.Unmarshal(body, &apiErr); err == nil {
		switch {
		case apiErr.ErrorCode != "" && apiErr.ErrorMsg != "":
			cause += fmt.Sprintf(": %s: %s", apiErr.ErrorCode, strings.TrimSpace(apiErr.ErrorMsg))
		case apiErr.ErrorMsg != "":
			cause += ": " + strings.TrimSpace(apiErr.ErrorMsg)
		case apiErr.ErrorCode != "":
			cause += ": " + apiErr.ErrorCode
		}
	}
	text := fmt.Sprintf("cyberark: %s failed with %s", msg, cause)
	if statusCode == http.StatusTooManyRequests || statusCode >= 500 {
		return &secretstores.ConnectionError{Err: errors.New(text), Cause: cause, Retry: true}
	}
	return errors.New(text)
}

func requestError(err error, msg string) error {
	if errors.Is(err, context.Canceled) {
		return fmt.Errorf("cyberark: %s: %w", msg, err)
	}
	if connErr, ok := secretstores.URLConnectionError(storeType, msg, err); ok {
		return connErr
	}
	return fmt.Errorf("cyberark: %s failed: %w", msg, err)
}
