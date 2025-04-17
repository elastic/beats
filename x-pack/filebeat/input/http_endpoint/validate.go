// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package http_endpoint

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
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
	maxBodySize        int64
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
		body := io.Reader(r.Body)
		if v.maxBodySize >= 0 {
			body = io.LimitReader(body, v.maxBodySize)
		}
		buf, err := io.ReadAll(body)
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
