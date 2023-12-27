// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package http_endpoint

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/http"
	"strings"
)

var (
	errIncorrectUserOrPass    = errors.New("incorrect username or password")
	errIncorrectHeaderSecret  = errors.New("incorrect header or header secret")
	errMissingHMACHeader      = errors.New("missing HMAC header")
	errIncorrectHMACSignature = errors.New("invalid HMAC signature")
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
}

func (v *apiValidator) validateRequest(r *http.Request) (status int, err error) {
	if v.basicAuth {
		username, password, _ := r.BasicAuth()
		if v.username != username || v.password != password {
			return http.StatusUnauthorized, errIncorrectUserOrPass
		}
	}

	if v.secretHeader != "" && v.secretValue != "" {
		if v.secretValue != r.Header.Get(v.secretHeader) {
			return http.StatusUnauthorized, errIncorrectHeaderSecret
		}
	}

	if v.method != "" && v.method != r.Method {
		return http.StatusMethodNotAllowed, fmt.Errorf("only %v requests are allowed", v.method)
	}

	if v.contentType != "" && r.Header.Get("Content-Type") != v.contentType {
		return http.StatusUnsupportedMediaType, fmt.Errorf("wrong Content-Type header, expecting %v", v.contentType)
	}

	if v.hmacHeader != "" && v.hmacKey != "" && v.hmacType != "" {
		// Read HMAC signature from HTTP header.
		hmacHeaderValue := r.Header.Get(v.hmacHeader)
		if v.hmacHeader == "" {
			return http.StatusUnauthorized, errMissingHMACHeader
		}
		if v.hmacPrefix != "" {
			hmacHeaderValue = strings.TrimPrefix(hmacHeaderValue, v.hmacPrefix)
		}
		signature, err := hex.DecodeString(hmacHeaderValue)
		if err != nil {
			return http.StatusUnauthorized, fmt.Errorf("invalid HMAC signature hex: %w", err)
		}

		// We need access to the request body to validate the signature, but we
		// must leave the body intact for future processing.
		buf, err := io.ReadAll(r.Body)
		if err != nil {
			return http.StatusInternalServerError, fmt.Errorf("failed to read request body: %w", err)
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
