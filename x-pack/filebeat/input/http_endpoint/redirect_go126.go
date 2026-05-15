// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// TODO: Remove this and in-line the constant when go1.25 and earlier is no longer supported.

//go:build go1.26

package http_endpoint

import "net/http"

// cleanRedirectCode is the status code used when redirecting requests
// with unclean paths. Go 1.26 changed http.ServeMux from 301 to 307;
// we match whichever version we are compiled with.
const cleanRedirectCode = http.StatusTemporaryRedirect
