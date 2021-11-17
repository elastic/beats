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

package http

import (
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/ecs"
)

// ProtocolFields contains HTTP fields. This contains all the HTTP fields from
// ECS. The ecs.Http type is not used because we customize some of the data
// data types to reduce memory allocations (common.NetString instead of string).
type ProtocolFields struct {
	// Http request method.
	// The field value must be normalized to lowercase for querying. See
	// "Lowercase Capitalization" in the "Implementing ECS"  section.
	RequestMethod common.NetString `ecs:"request.method"`

	// HTTP request ID.
	RequestID common.NetString `ecs:"request.id"`

	// The full http request body.
	RequestBodyContent common.NetString `ecs:"request.body.content"`

	// Referrer for this HTTP request.
	RequestReferrer common.NetString `ecs:"request.referrer"`

	// HTTP request mime-type.
	RequestMIMEType string `ecs:"request.mime_type"`

	// Http response status code.
	ResponseStatusCode int64 `ecs:"response.status_code"`

	// The full http response body.
	ResponseBodyContent common.NetString `ecs:"response.body.content"`

	// Http version.
	Version string `ecs:"version"`

	// Total size in bytes of the request (body and headers).
	RequestBytes int64 `ecs:"request.bytes"`

	// Size in bytes of the request body.
	RequestBodyBytes int64 `ecs:"request.body.bytes"`

	// Total size in bytes of the response (body and headers).
	ResponseBytes int64 `ecs:"response.bytes"`

	// Size in bytes of the response body.
	ResponseBodyBytes int64 `ecs:"response.body.bytes"`

	// HTTP request headers.
	RequestHeaders common.MapStr `packetbeat:"request.headers"`

	// HTTP response headers.
	ResponseHeaders common.MapStr `packetbeat:"response.headers"`

	// HTTP response mime-type.
	ResponseMIMEType string `ecs:"response.mime_type"`

	// HTTP response status phrase.
	ResponseStatusPhrase common.NetString `packetbeat:"response.status_phrase"`
}

// netURL returns a new ecs.Url object with data from the HTTP request.
func newURL(host string, port int64, path, query string) *ecs.Url {
	u := &ecs.Url{
		Scheme: "http",
		Domain: host,
		Path:   path,
		Query:  query,
	}
	if port != 80 {
		u.Port = port
	}
	if path != "" {
		periodIndex := strings.LastIndex(path, ".")
		if periodIndex != -1 && periodIndex < len(path) {
			u.Extension = path[(periodIndex + 1):]
		}
	}
	u.Full = synthesizeFullURL(u, port)
	return u
}

func synthesizeFullURL(u *ecs.Url, port int64) string {
	if u.Domain == "" || port <= 0 {
		return ""
	}

	host := u.Domain
	if port != 80 {
		host = net.JoinHostPort(u.Domain, strconv.Itoa(int(u.Port)))
	} else if strings.IndexByte(u.Domain, ':') != -1 {
		host = "[" + u.Domain + "]"
	}

	urlBuilder := url.URL{
		Scheme:   u.Scheme,
		Host:     host,
		Path:     u.Path,
		RawQuery: u.Query,
	}
	return urlBuilder.String()
}
