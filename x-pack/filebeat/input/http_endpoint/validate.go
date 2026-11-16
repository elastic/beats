// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package http_endpoint

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
)

var (
	errIncorrectUserOrPass    = errors.New("incorrect username or password")
	errIncorrectHeaderSecret  = errors.New("incorrect header or header secret")
	errMissingHMACHeader      = errors.New("missing HMAC header")
	errIncorrectHMACSignature = errors.New("invalid HMAC signature")
	errHMACBodyTooLarge       = errors.New("request body too large for HMAC validation")
)

type apiValidator struct {
	basicAuth          bool
	username, password string
	method             string
	contentType        string
	secretHeader       string
	secretValue        string
	hmacHeader         string
	hmacKey            string
	hmacType           string
	hmacPrefix         string
	maxBodySize        int64

	// inFlight is the handler's in-flight byte counter. If maxInFlight is
	// non-zero, bytes buffered for HMAC validation count towards it.
	inFlight    *atomic.Int64
	maxInFlight int64

	optionsHeaders http.Header
	optionsStatus  int
}

func (v *apiValidator) validateRequest(r *http.Request) (status int, err error) {
	if v.basicAuth {
		username, password, _ := r.BasicAuth()
		usernameMatch := subtle.ConstantTimeCompare([]byte(v.username), []byte(username))
		passwordMatch := subtle.ConstantTimeCompare([]byte(v.password), []byte(password))
		if usernameMatch&passwordMatch != 1 {
			return http.StatusUnauthorized, errIncorrectUserOrPass
		}
	}

	if v.secretHeader != "" && v.secretValue != "" {
		if subtle.ConstantTimeCompare([]byte(v.secretValue), []byte(r.Header.Get(v.secretHeader))) != 1 {
			return http.StatusUnauthorized, errIncorrectHeaderSecret
		}
	}

	if !v.isMethodOK(r.Method) {
		if r.Method == http.MethodOptions {
			return http.StatusBadRequest, errors.New("OPTIONS requests are only allowed with options_headers set")
		}
		return http.StatusMethodNotAllowed, fmt.Errorf("only %v requests are allowed", v.method)
	}

	if v.contentType != "" && r.Header.Get("Content-Type") != v.contentType {
		return http.StatusUnsupportedMediaType, fmt.Errorf("wrong Content-Type header, expecting %v", v.contentType)
	}

	if v.hmacHeader != "" && v.hmacKey != "" && v.hmacType != "" {
		// Check whether the HMAC header exists at all.
		if len(r.Header.Values(v.hmacHeader)) == 0 {
			return http.StatusUnauthorized, errMissingHMACHeader
		}
		// Read HMAC signature from HTTP header.
		hmacHeaderValue := r.Header.Get(v.hmacHeader)
		signature, err := decodeHeaderValue(strings.TrimPrefix(hmacHeaderValue, v.hmacPrefix))
		if err != nil {
			return http.StatusUnauthorized, fmt.Errorf("invalid HMAC signature encoding: %w", err)
		}

		// We need access to the request body to validate the signature, but we
		// must leave the body intact for future processing.
		// The body is buffered before the request is authenticated, so it
		// is always bounded. When max_body_bytes is not set, an extra byte is
		// read to distinguish an oversized body from one at the limit.
		n := v.maxBodySize
		if n < 0 {
			n = hmacBodyLimit + 1
		}
		body := io.LimitReader(r.Body, n)
		if v.maxInFlight > 0 {
			// The deferred Close releases the counted bytes when validation
			// returns; they are counted again when the handler reads the
			// buffered body.
			cr := newCountReader(io.NopCloser(body), v.inFlight, v.maxInFlight)
			defer cr.Close()
			body = cr
		}
		buf, err := io.ReadAll(body)
		if errors.Is(err, errMaxInFlightExceeded) {
			return http.StatusServiceUnavailable, err
		}
		if err != nil {
			return http.StatusInternalServerError, fmt.Errorf("failed to read request body: %w", err)
		}
		if v.maxBodySize < 0 && len(buf) > hmacBodyLimit {
			return http.StatusRequestEntityTooLarge, errHMACBodyTooLarge
		}
		// Set r.Body back to untouched original value.
		r.Body = io.NopCloser(bytes.NewBuffer(buf))

		// Compute HMAC of raw body.
		var mac hash.Hash
		switch v.hmacType {
		case "sha256":
			mac = hmac.New(sha256.New, []byte(v.hmacKey))
		case "sha1":
			mac = hmac.New(sha1.New, []byte(v.hmacKey))
		default:
			// Upstream config validation prevents this from happening.
			panic(fmt.Errorf("unhandled hmac.type %q", v.hmacType))
		}
		mac.Write(buf)
		actualMAC := mac.Sum(nil)

		if !hmac.Equal(signature, actualMAC) {
			return http.StatusUnauthorized, errIncorrectHMACSignature
		}
	}

	return http.StatusAccepted, nil
}

// hmacBodyLimit is the largest body that is buffered for HMAC validation
// when max_body_bytes is not set. It is at least as large as the largest
// payload GitHub sends to webhooks, which it caps at 25 MB.
const hmacBodyLimit = 25 << 20

func (v *apiValidator) isMethodOK(m string) bool {
	if m == http.MethodOptions {
		return v.optionsHeaders != nil
	}
	return v.method == "" || m == v.method
}

// decoders is the priority-ordered set of decoders to use for HMAC header values.
var decoders = [...]func(string) ([]byte, error){
	hex.DecodeString,
	base64.RawStdEncoding.DecodeString,
	base64.StdEncoding.DecodeString,
}

// decodeHeaderValue attempts to decode s as hex, unpadded base64
// ([base64.RawStdEncoding]), and padded base64 ([base64.StdEncoding]).
// The first successful decoding result is returned. If all decodings fail, it
// collects errors from each attempt and returns them as a single error.
func decodeHeaderValue(s string) ([]byte, error) {
	if s == "" {
		return nil, errors.New("unexpected empty header value")
	}
	var errs []error
	for _, d := range &decoders {
		b, err := d(s)
		if err == nil {
			return b, nil
		}
		errs = append(errs, err)
	}
	return nil, errors.Join(errs...)
}
