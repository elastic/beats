// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package esapi

import (
	"context"
	"io"
	"net/http"
)

const (
	headerContentType = "Content-Type"
)

var (
	headerContentTypeJSON = []string{"application/json"}
)

// Request defines the API request.
//
type Request interface {
	Do(ctx context.Context, transport Transport) (*Response, error)
}

// newRequest creates an HTTP request.
//
func newRequest(method, path string, body io.Reader) (*http.Request, error) {
	return http.NewRequest(method, path, body)
}
