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

	"go.elastic.co/apm/internal/apmhttputil"
	"go.elastic.co/apm/internal/wildcard"
	"go.elastic.co/apm/model"
)

// Context provides methods for setting transaction and error context.
//
// NOTE this is entirely unrelated to the standard library's context.Context.
type Context struct {
	model               model.Context
	request             model.Request
	requestBody         model.RequestBody
	requestSocket       model.RequestSocket
	response            model.Response
	user                model.User
	service             model.Service
	serviceFramework    model.Framework
	captureHeaders      bool
	captureBodyMask     CaptureBodyMode
	sanitizedFieldNames wildcard.Matchers
}

func (c *Context) build() *model.Context {
	switch {
	case c.model.Request != nil:
	case c.model.Response != nil:
	case c.model.User != nil:
	case c.model.Service != nil:
	case len(c.model.Tags) != 0:
	case len(c.model.Custom) != 0:
	default:
		return nil
	}
	if len(c.sanitizedFieldNames) != 0 {
		if c.model.Request != nil {
			sanitizeRequest(c.model.Request, c.sanitizedFieldNames)
		}
		if c.model.Response != nil {
			sanitizeResponse(c.model.Response, c.sanitizedFieldNames)
		}

	}
	return &c.model
}

func (c *Context) reset() {
	*c = Context{
		model: model.Context{
			Custom: c.model.Custom[:0],
			Tags:   c.model.Tags[:0],
		},
		captureBodyMask: c.captureBodyMask,
		request: model.Request{
			Headers: c.request.Headers[:0],
		},
		response: model.Response{
			Headers: c.response.Headers[:0],
		},
	}
}

// SetTag calls SetLabel(key, value).
//
// SetTag is deprecated, and will be removed in a future major version.
func (c *Context) SetTag(key, value string) {
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
func (c *Context) SetLabel(key string, value interface{}) {
	// Note that we do not attempt to de-duplicate the keys.
	// This is OK, since json.Unmarshal will always take the
	// final instance.
	c.model.Tags = append(c.model.Tags, model.IfaceMapItem{
		Key:   cleanLabelKey(key),
		Value: makeLabelValue(value),
	})
}

// SetCustom sets custom context.
//
// Invalid characters ('.', '*', and '"') in the key will be
// replaced with an underscore. The value may be any JSON-encodable
// value.
func (c *Context) SetCustom(key string, value interface{}) {
	// Note that we do not attempt to de-duplicate the keys.
	// This is OK, since json.Unmarshal will always take the
	// final instance.
	c.model.Custom = append(c.model.Custom, model.IfaceMapItem{
		Key:   cleanLabelKey(key),
		Value: value,
	})
}

// SetFramework sets the framework name and version in the context.
//
// This is used for identifying the framework in which the context
// was created, such as Gin or Echo.
//
// If the name is empty, this is a no-op. If version is empty, then
// it will be set to "unspecified".
func (c *Context) SetFramework(name, version string) {
	if name == "" {
		return
	}
	if version == "" {
		// Framework version is required.
		version = "unspecified"
	}
	c.serviceFramework = model.Framework{
		Name:    truncateString(name),
		Version: truncateString(version),
	}
	c.service.Framework = &c.serviceFramework
	c.model.Service = &c.service
}

// SetHTTPRequest sets details of the HTTP request in the context.
//
// This function relates to server-side requests. Various proxy
// forwarding headers are taken into account to reconstruct the URL,
// and determining the client address.
//
// If the request URL contains user info, it will be removed and
// excluded from the URL's "full" field.
//
// If the request contains HTTP Basic Authentication, the username
// from that will be recorded in the context. Otherwise, if the
// request contains user info in the URL (i.e. a client-side URL),
// that will be used.
func (c *Context) SetHTTPRequest(req *http.Request) {
	// Special cases to avoid calling into fmt.Sprintf in most cases.
	var httpVersion string
	switch {
	case req.ProtoMajor == 1 && req.ProtoMinor == 1:
		httpVersion = "1.1"
	case req.ProtoMajor == 2 && req.ProtoMinor == 0:
		httpVersion = "2.0"
	default:
		httpVersion = fmt.Sprintf("%d.%d", req.ProtoMajor, req.ProtoMinor)
	}

	c.request = model.Request{
		Body:        c.request.Body,
		URL:         apmhttputil.RequestURL(req),
		Method:      truncateString(req.Method),
		HTTPVersion: httpVersion,
		Cookies:     req.Cookies(),
	}
	c.model.Request = &c.request

	if c.captureHeaders {
		for k, values := range req.Header {
			if k == "Cookie" {
				// We capture cookies in the request structure.
				continue
			}
			c.request.Headers = append(c.request.Headers, model.Header{
				Key: k, Values: values,
			})
		}
	}

	c.requestSocket = model.RequestSocket{
		Encrypted:     req.TLS != nil,
		RemoteAddress: apmhttputil.RemoteAddr(req),
	}
	if c.requestSocket != (model.RequestSocket{}) {
		c.request.Socket = &c.requestSocket
	}

	username, _, ok := req.BasicAuth()
	if !ok && req.URL.User != nil {
		username = req.URL.User.Username()
	}
	c.user.Username = truncateString(username)
	if c.user.Username != "" {
		c.model.User = &c.user
	}
}

// SetHTTPRequestBody sets the request body in context given a (possibly nil)
// BodyCapturer returned by Tracer.CaptureHTTPRequestBody.
func (c *Context) SetHTTPRequestBody(bc *BodyCapturer) {
	if bc == nil || bc.captureBody&c.captureBodyMask == 0 {
		return
	}
	if bc.setContext(&c.requestBody) {
		c.request.Body = &c.requestBody
	}
}

// SetHTTPResponseHeaders sets the HTTP response headers in the context.
func (c *Context) SetHTTPResponseHeaders(h http.Header) {
	if !c.captureHeaders {
		return
	}
	for k, values := range h {
		c.response.Headers = append(c.response.Headers, model.Header{
			Key: k, Values: values,
		})
	}
	if len(c.response.Headers) != 0 {
		c.model.Response = &c.response
	}
}

// SetHTTPStatusCode records the HTTP response status code.
//
// If, when the transaction ends, its Outcome field has not
// been explicitly set, it will be set based on the status code:
// "success" if statusCode < 500, and "failure" otherwise.
func (c *Context) SetHTTPStatusCode(statusCode int) {
	c.response.StatusCode = statusCode
	c.model.Response = &c.response
}

// SetUserID sets the ID of the authenticated user.
func (c *Context) SetUserID(id string) {
	c.user.ID = truncateString(id)
	if c.user.ID != "" {
		c.model.User = &c.user
	}
}

// SetUserEmail sets the email for the authenticated user.
func (c *Context) SetUserEmail(email string) {
	c.user.Email = truncateString(email)
	if c.user.Email != "" {
		c.model.User = &c.user
	}
}

// SetUsername sets the username of the authenticated user.
func (c *Context) SetUsername(username string) {
	c.user.Username = truncateString(username)
	if c.user.Username != "" {
		c.model.User = &c.user
	}
}

// outcome returns the outcome to assign to the associated transaction,
// based on context (e.g. HTTP status code).
func (c *Context) outcome() string {
	if c.response.StatusCode != 0 {
		if c.response.StatusCode < 500 {
			return "success"
		}
		return "failure"
	}
	return ""
}
