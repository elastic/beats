// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package usage

import "errors"

var (
	// ErrNoState indicates no previous state exists for the given API key
	ErrNoState = errors.New("no previous state found")

	// ErrHTTPClientTimeout indicates request timeout
	ErrHTTPClientTimeout = errors.New("http client request timeout")
)
