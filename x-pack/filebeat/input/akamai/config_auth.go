// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// This file was contributed to by generative AI

package akamai

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gofrs/uuid/v5"
)

// authConfig holds authentication configuration for the Akamai API.
type authConfig struct {
	EdgeGrid *edgeGridConfig `config:"edgegrid"`
}

// edgeGridConfig holds EdgeGrid authentication credentials.
type edgeGridConfig struct {
	ClientToken  string `config:"client_token"`
	ClientSecret string `config:"client_secret"`
	AccessToken  string `config:"access_token"`
}

// Validate validates the authentication configuration.
func (a *authConfig) Validate() error {
	if a == nil || a.EdgeGrid == nil {
		return errors.New("at least one auth method must be configured")
	}

	if a.EdgeGrid.ClientToken == "" {
		return errors.New("auth.edgegrid.client_token is required")
	}
	if a.EdgeGrid.ClientSecret == "" {
		return errors.New("auth.edgegrid.client_secret is required")
	}
	if a.EdgeGrid.AccessToken == "" {
		return errors.New("auth.edgegrid.access_token is required")
	}
	return nil
}

// EdgeGridSigner signs HTTP requests using Akamai EdgeGrid authentication.
type EdgeGridSigner struct {
	clientToken  string
	clientSecret string
	accessToken  string
}

// NewEdgeGridSigner creates a new EdgeGrid signer with the provided credentials.
func NewEdgeGridSigner(clientToken, clientSecret, accessToken string) *EdgeGridSigner {
	return &EdgeGridSigner{
		clientToken:  clientToken,
		clientSecret: clientSecret,
		accessToken:  accessToken,
	}
}

// Sign adds the EdgeGrid authorization header to the request.
// The signature is generated according to the Akamai EdgeGrid specification:
// https://techdocs.akamai.com/developer/docs/authenticate-with-edgegrid
func (s *EdgeGridSigner) Sign(req *http.Request) error {
	timestamp := time.Now().UTC().Format("20060102T15:04:05-0700")
	u, err := uuid.NewV4()
	if err != nil {
		return fmt.Errorf("failed to generate nonce: %w", err)
	}
	nonce := u.String()

	// Build the authorization header base
	authBase := fmt.Sprintf(
		"EG1-HMAC-SHA256 client_token=%s;access_token=%s;timestamp=%s;nonce=%s;",
		s.clientToken, s.accessToken, timestamp, nonce,
	)

	// Generate the signing key
	signingKey := s.createSigningKey(timestamp)

	// Build the data to sign
	dataToSign := s.buildDataToSign(req, authBase)

	// Generate the signature
	signature := s.computeSignature(dataToSign, signingKey)

	// Set the authorization header
	authHeader := authBase + "signature=" + signature
	req.Header.Set("Authorization", authHeader)

	return nil
}

// createSigningKey creates an HMAC-SHA256 signing key from the timestamp and client secret.
func (s *EdgeGridSigner) createSigningKey(timestamp string) string {
	mac := hmac.New(sha256.New, []byte(s.clientSecret))
	mac.Write([]byte(timestamp))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// buildDataToSign builds the string that will be signed.
// Format: Method\tScheme\tHost\tPath?Query\tHeaders\tContentHash\tAuthBase
func (s *EdgeGridSigner) buildDataToSign(req *http.Request, authBase string) string {
	scheme := strings.ToLower(req.URL.Scheme)
	if scheme == "" {
		scheme = "https"
	}
	host := strings.ToLower(req.URL.Host)
	path := req.URL.Path
	if path == "" {
		path = "/"
	}

	var sb strings.Builder
	sb.WriteString(req.Method)
	sb.WriteString("\t")
	sb.WriteString(scheme)
	sb.WriteString("\t")
	sb.WriteString(host)
	sb.WriteString("\t")
	sb.WriteString(path)
	if req.URL.RawQuery != "" {
		sb.WriteString("?")
		sb.WriteString(req.URL.RawQuery)
	}
	sb.WriteString("\t\t\t")
	sb.WriteString(authBase)

	return sb.String()
}

// computeSignature computes the HMAC-SHA256 signature.
func (s *EdgeGridSigner) computeSignature(data, key string) string {
	// Keep parity with the proven CEL implementation:
	// signature = base64(hmac(data, bytes(sig_key)))
	// where sig_key is already the base64-encoded output of the first HMAC.
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(data))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// EdgeGridTransport wraps an http.RoundTripper to add EdgeGrid authentication.
type EdgeGridTransport struct {
	Transport http.RoundTripper
	Signer    *EdgeGridSigner
}

// RoundTrip implements the http.RoundTripper interface.
func (t *EdgeGridTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request to avoid mutating the original
	reqClone := req.Clone(req.Context())
	if reqClone.URL == nil {
		reqClone.URL = &url.URL{}
	}

	// Sign the request
	if err := t.Signer.Sign(reqClone); err != nil {
		return nil, fmt.Errorf("failed to sign request: %w", err)
	}

	return t.Transport.RoundTrip(reqClone)
}
