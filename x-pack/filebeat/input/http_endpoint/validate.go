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
	"io/ioutil"
	"net/http"
	"strings"
)

type validator interface {
	// ValidateHeader checks the HTTP headers for compliance. The body must not
	// be touched.
	ValidateHeader(*http.Request) (int, error)
}

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

var (
	errIncorrectUserOrPass    = errors.New("incorrect username or password")
	errIncorrectHeaderSecret  = errors.New("incorrect header or header secret")
	errMissingHMACHeader      = errors.New("missing HMAC header")
	errIncorrectHMACSignature = errors.New("invalid HMAC signature")
)

func (v *apiValidator) ValidateHeader(r *http.Request) (int, error) {
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
		// We need access to the request body to validate the signature.
		buf, _ := ioutil.ReadAll(r.Body)
		rdr1 := ioutil.NopCloser(bytes.NewBuffer(buf))
		originalBody := ioutil.NopCloser(bytes.NewBuffer(buf))

		var bodyBytes, _ = ioutil.ReadAll(rdr1)
		var mac hash.Hash
		if v.hmacType == "sha256" {
			mac = hmac.New(sha256.New, []byte(v.hmacKey))
		} else {
			mac = hmac.New(sha1.New, []byte(v.hmacKey))
		}

		mac.Write(bodyBytes)
		actualMAC := mac.Sum(nil)

		hmacHeaderValue := r.Header.Get(v.hmacHeader)
		if v.hmacPrefix != "" {
			hmacHeaderValue = strings.Replace(hmacHeaderValue, v.hmacPrefix, "", 1)
		}

		signature, _ := hex.DecodeString(hmacHeaderValue)

		// Set r.Body back to untouched original value.
		r.Body = originalBody

		if !hmac.Equal(signature, actualMAC) {
			return http.StatusUnauthorized, errIncorrectHmacSignature
		}
	}

	return 0, nil
}
