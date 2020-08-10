// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package http_endpoint

import (
	"errors"
	"fmt"
	"net/http"
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
}

var errIncorrectUserOrPass = errors.New("Incorrect username or password")
var errIncorrectHeaderSecret = errors.New("Incorrect header or header secret")

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
		return http.StatusMethodNotAllowed, fmt.Errorf("Only %v requests supported", v.method)
	}

	if v.contentType != "" && r.Header.Get("Content-Type") != v.contentType {
		return http.StatusUnsupportedMediaType, fmt.Errorf("Wrong Content-Type header, expecting %v", v.contentType)
	}

	return 0, nil
}
