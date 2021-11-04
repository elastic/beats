// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package ecs

// Fields related to HTTP activity. Use the `url` field set to store the url of
// the request.
type Http struct {
	// A unique identifier for each HTTP request to correlate logs between
	// clients and servers in transactions.
	// The id may be contained in a non-standard HTTP header, such as
	// `X-Request-ID` or `X-Correlation-ID`.
	RequestID string `ecs:"request.id"`

	// HTTP request method in original case.
	RequestMethod string `ecs:"request.method"`

	// Mime type of the body of the request.
	// This value must only be populated based on the content of the request
	// body, not on the `Content-Type` header. Comparing the mime type of a
	// request with the request's Content-Type header can be helpful in
	// detecting threats or misconfigured clients.
	RequestMimeType string `ecs:"request.mime_type"`

	// The full HTTP request body.
	RequestBodyContent string `ecs:"request.body.content"`

	// Referrer for this HTTP request.
	RequestReferrer string `ecs:"request.referrer"`

	// HTTP response status code.
	ResponseStatusCode int64 `ecs:"response.status_code"`

	// Mime type of the body of the response.
	// This value must only be populated based on the content of the response
	// body, not on the `Content-Type` header. Comparing the mime type of a
	// response with the response's Content-Type header can be helpful in
	// detecting misconfigured servers.
	ResponseMimeType string `ecs:"response.mime_type"`

	// The full HTTP response body.
	ResponseBodyContent string `ecs:"response.body.content"`

	// HTTP version.
	Version string `ecs:"version"`

	// Total size in bytes of the request (body and headers).
	RequestBytes int64 `ecs:"request.bytes"`

	// Size in bytes of the request body.
	RequestBodyBytes int64 `ecs:"request.body.bytes"`

	// Total size in bytes of the response (body and headers).
	ResponseBytes int64 `ecs:"response.bytes"`

	// Size in bytes of the response body.
	ResponseBodyBytes int64 `ecs:"response.body.bytes"`
}
