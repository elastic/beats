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

package apm // import "go.elastic.co/apm"

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"go.elastic.co/apm/internal/apmhttputil"
	"go.elastic.co/apm/model"
)

// SpanContext provides methods for setting span context.
type SpanContext struct {
	model                model.SpanContext
	destination          model.DestinationSpanContext
	destinationService   model.DestinationServiceSpanContext
	databaseRowsAffected int64
	database             model.DatabaseSpanContext
	http                 model.HTTPSpanContext
}

// DatabaseSpanContext holds database span context.
type DatabaseSpanContext struct {
	// Instance holds the database instance name.
	Instance string

	// Statement holds the statement executed in the span,
	// e.g. "SELECT * FROM foo".
	Statement string

	// Type holds the database type, e.g. "sql".
	Type string

	// User holds the username used for database access.
	User string
}

// DestinationServiceSpanContext holds destination service span span.
type DestinationServiceSpanContext struct {
	// Name holds a name for the destination service, which may be used
	// for grouping and labeling in service maps.
	Name string

	// Resource holds an identifier for a destination service resource,
	// such as a message queue.
	Resource string
}

func (c *SpanContext) build() *model.SpanContext {
	switch {
	case len(c.model.Tags) != 0:
	case c.model.Database != nil:
	case c.model.HTTP != nil:
	case c.model.Destination != nil:
	default:
		return nil
	}
	return &c.model
}

func (c *SpanContext) reset() {
	*c = SpanContext{
		model: model.SpanContext{
			Tags: c.model.Tags[:0],
		},
	}
}

// SetTag calls SetLabel(key, value).
//
// SetTag is deprecated, and will be removed in a future major version.
func (c *SpanContext) SetTag(key, value string) {
	c.SetLabel(key, value)
}

// SetLabel sets a label in the context.
//
// Invalid characters ('.', '*', and '"') in the key will be replaced with
// underscores.
//
// If the value is numerical or boolean, then it will be sent to the server
// as a JSON number or boolean; otherwise it will converted to a string, using
// `fmt.Sprint` if necessary. String values longer than 1024 characters will
// be truncated.
func (c *SpanContext) SetLabel(key string, value interface{}) {
	// Note that we do not attempt to de-duplicate the keys.
	// This is OK, since json.Unmarshal will always take the
	// final instance.
	c.model.Tags = append(c.model.Tags, model.IfaceMapItem{
		Key:   cleanLabelKey(key),
		Value: makeLabelValue(value),
	})
}

// SetDatabase sets the span context for database-related operations.
func (c *SpanContext) SetDatabase(db DatabaseSpanContext) {
	c.database = model.DatabaseSpanContext{
		Instance:  truncateString(db.Instance),
		Statement: truncateLongString(db.Statement),
		Type:      truncateString(db.Type),
		User:      truncateString(db.User),
	}
	c.model.Database = &c.database
}

// SetDatabaseRowsAffected records the number of rows affected by
// a database operation.
func (c *SpanContext) SetDatabaseRowsAffected(n int64) {
	c.databaseRowsAffected = n
	c.database.RowsAffected = &c.databaseRowsAffected
}

// SetHTTPRequest sets the details of the HTTP request in the context.
//
// This function relates to client requests. If the request URL contains
// user info, it will be removed and excluded from the stored URL.
//
// SetHTTPRequest makes implicit calls to SetDestinationAddress and
// SetDestinationService, using details from req.URL.
func (c *SpanContext) SetHTTPRequest(req *http.Request) {
	if req.URL == nil {
		return
	}
	c.http.URL = req.URL
	c.model.HTTP = &c.http

	addr, port := apmhttputil.DestinationAddr(req)
	c.SetDestinationAddress(addr, port)

	destinationServiceURL := url.URL{Scheme: req.URL.Scheme, Host: req.URL.Host}
	destinationServiceResource := destinationServiceURL.Host
	if port != 0 && port == apmhttputil.SchemeDefaultPort(req.URL.Scheme) {
		var hasDefaultPort bool
		if n := len(destinationServiceURL.Host); n > 0 && destinationServiceURL.Host[n-1] != ']' {
			if i := strings.LastIndexByte(destinationServiceURL.Host, ':'); i != -1 {
				// Remove the default port from destination.service.name.
				destinationServiceURL.Host = destinationServiceURL.Host[:i]
				hasDefaultPort = true
			}
		}
		if !hasDefaultPort {
			// Add the default port to destination.service.resource.
			destinationServiceResource = fmt.Sprintf("%s:%d", destinationServiceResource, port)
		}
	}
	c.SetDestinationService(DestinationServiceSpanContext{
		Name:     destinationServiceURL.String(),
		Resource: destinationServiceResource,
	})
}

// SetHTTPStatusCode records the HTTP response status code.
//
// If, when the transaction ends, its Outcome field has not
// been explicitly set, it will be set based on the status code:
// "success" if statusCode < 400, and "failure" otherwise.
func (c *SpanContext) SetHTTPStatusCode(statusCode int) {
	c.http.StatusCode = statusCode
	c.model.HTTP = &c.http
}

// SetDestinationAddress sets the destination address and port in the context.
//
// SetDestinationAddress has no effect when called when an empty addr.
func (c *SpanContext) SetDestinationAddress(addr string, port int) {
	if addr != "" {
		c.destination.Address = truncateString(addr)
		c.destination.Port = port
		c.model.Destination = &c.destination
	}
}

// SetDestinationService sets the destination service info in the context.
func (c *SpanContext) SetDestinationService(service DestinationServiceSpanContext) {
	c.destinationService.Name = truncateString(service.Name)
	c.destinationService.Resource = truncateString(service.Resource)
	c.destination.Service = &c.destinationService
	c.model.Destination = &c.destination
}

// outcome returns the outcome to assign to the associated span, based on
// context (e.g. HTTP status code).
func (c *SpanContext) outcome() string {
	if c.http.StatusCode != 0 {
		if c.http.StatusCode < 400 {
			return "success"
		}
		return "failure"
	}
	return ""
}
