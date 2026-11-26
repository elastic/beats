// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httplog

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"

	"github.com/elastic/beats/v7/x-pack/filebeat/input/internal/private"
	"github.com/elastic/beats/v7/x-pack/filebeat/otel"
)

const (
	// redacted replaces the value of a credential in a trace.
	redacted = "REDACTED"

	// unparsedBody is logged in place of a body that names a credential
	// field, but that cannot be parsed to mask the field's value.
	unparsedBody = "REDACTED: unparsed body containing credential field names"

	// unparsedQuery is logged in place of a query that cannot be parsed. It
	// contains no characters that need escaping, so the URL that holds it
	// stays well formed.
	unparsedQuery = "REDACTED-unparsed-query"
)

// masker masks the credentials in the headers that are logged. The zero value
// masks the known credential headers without any names from the caller.
type masker struct {
	// headers are the lower-case names of headers that the caller has
	// identified as carrying credentials.
	headers map[string]struct{}
}

// newMasker returns a masker that also masks the values of the headers named
// in sensitive, ignoring case.
func newMasker(sensitive []string) masker {
	m := masker{headers: make(map[string]struct{}, len(sensitive))}
	for _, n := range sensitive {
		m.headers[strings.ToLower(n)] = struct{}{}
	}
	return m
}

// header returns a copy of h in which the values of sensitive headers are
// masked. The scheme of an Authorization or Proxy-Authorization value is kept
// if it is a well-known one. If no value is masked, h is returned.
func (m masker) header(h http.Header) http.Header {
	if len(h) == 0 {
		return h
	}
	got, changed, err := maskValues(h, m.headerValue)
	if err != nil {
		return http.Header{"Error": {"REDACTED: header redaction failed"}}
	}
	if !changed {
		return h
	}
	return got
}

func (m masker) headerValue(path []string, s string) (string, bool) {
	if len(path) == 0 || !m.sensitiveHeader(path[0]) {
		return "", false
	}
	switch strings.ToLower(path[0]) {
	case "authorization", "proxy-authorization":
		scheme, _, ok := strings.Cut(s, " ")
		if _, known := authSchemes[strings.ToLower(scheme)]; ok && known {
			return scheme + " " + redacted, true
		}
	}
	return redacted, true
}

// authSchemes are the lower-case authentication schemes that remain visible
// in masked Authorization and Proxy-Authorization headers. Any other prefix
// is masked with the rest of the value, since it could be part of a secret.
var authSchemes = setOf(
	"apikey",
	"aws4-hmac-sha256",
	"basic",
	"bearer",
	"digest",
	"negotiate",
	"ntlm",
	"token",
)

func (m masker) sensitiveHeader(name string) bool {
	lower := strings.ToLower(name)
	if _, ok := m.headers[lower]; ok {
		return true
	}
	if _, ok := denyHeaders[lower]; ok {
		return true
	}
	return otel.SensitiveName(name)
}

// denyHeaders are the lower-case names of headers whose values are always
// masked, in addition to those the caller names and those that the otel
// classifier reports as sensitive.
var denyHeaders = setOf(
	"authorization",
	"cookie",
	"proxy-authorization",
	"set-cookie",
	"x-api-key",
	"x-apikeys",
)

// maskURL returns the string form of u and its query with credentials masked.
// The password and user name of any user info are masked.
func maskURL(u *url.URL) (original, query string) {
	c := *u
	c.RawQuery = maskQuery(u.RawQuery)
	if c.User != nil {
		if _, ok := c.User.Password(); ok {
			c.User = url.UserPassword(redacted, redacted)
		} else {
			c.User = url.User(redacted)
		}
	}
	return c.String(), c.RawQuery
}

// maskQuery returns the raw query with the values of parameters named in
// queryNames masked. If no value is masked the query is returned as it was.
func maskQuery(raw string) string {
	if raw == "" {
		return raw
	}
	q, err := url.ParseQuery(raw)
	if err != nil {
		return unparsedQuery
	}
	got, changed, err := maskValues(q, func(path []string, _ string) (string, bool) {
		if len(path) == 0 {
			return "", false
		}
		_, ok := queryNames[strings.ToLower(path[0])]
		return redacted, ok
	})
	if err != nil {
		return unparsedQuery
	}
	if !changed {
		return raw
	}
	return got.Encode()
}

// queryNames are the lower-case names of query parameters whose values are
// masked. The list is exact, so pagination cursors such as nextPageToken stay
// visible.
var queryNames = setOf(
	"access_token",
	"api_key",
	"api_token",
	"apikey",
	"assertion",
	"auth_token",
	"client_assertion",
	"client_secret",
	"id_token",
	"key",
	"password",
	"refresh_token",
	"secret",
	"sig",
	"signature",
	"token",
	"x-amz-security-token",
	"x-amz-signature",
)

// maskBody returns body with the values of the fields named in bodyNames
// masked at any depth. Form and JSON bodies, including newline-delimited and
// concatenated JSON, are parsed to do this, and are returned unchanged if no
// value is masked. A form or JSON body that names a field, but that cannot be
// parsed, is replaced. Bodies of other types are returned as they are.
func maskBody(contentType string, body []byte) []byte {
	if len(body) == 0 || !namesField(body) {
		return body
	}
	switch formatOf(contentType, body) {
	case formBody:
		got, ok := maskForm(body)
		if !ok {
			return []byte(unparsedBody)
		}
		return got
	case jsonBody:
		got, ok := maskJSON(body)
		if !ok {
			return []byte(unparsedBody)
		}
		return got
	default:
		return body
	}
}

// namesField reports whether body contains the name of any credential field,
// ignoring case. The body is lower-cased once and searched for each name,
// which is acceptable because it is only done while tracing is enabled.
func namesField(body []byte) bool {
	lower := bytes.ToLower(body)
	for n := range bodyNames {
		if bytes.Contains(lower, []byte(n)) {
			return true
		}
	}
	return false
}

type bodyFormat int

const (
	otherBody bodyFormat = iota
	formBody
	jsonBody
)

// formatOf returns the format to parse a body as, from its declared content
// type and, if the type is missing or not a known one, from its content. A
// body that starts like a JSON document is treated as JSON even if it is
// declared as a form.
func formatOf(contentType string, body []byte) bodyFormat {
	mt, _, _ := mime.ParseMediaType(contentType)
	t := bytes.TrimSpace(body)
	looksJSON := len(t) != 0 && (t[0] == '{' || t[0] == '[')
	switch {
	case mt == "application/x-www-form-urlencoded":
		if looksJSON {
			return jsonBody
		}
		return formBody
	case mt == "application/json", strings.HasSuffix(mt, "+json"),
		mt == "application/x-ndjson", mt == "application/ndjson", mt == "application/jsonl":
		return jsonBody
	case looksJSON:
		return jsonBody
	case mt == "" && bytes.Contains(t, []byte("=")):
		return formBody
	}
	return otherBody
}

func maskForm(body []byte) ([]byte, bool) {
	q, err := url.ParseQuery(string(body))
	if err != nil {
		return nil, false
	}
	got, changed, err := maskValues(q, maskBodyValue)
	if err != nil {
		return nil, false
	}
	if !changed {
		return body, true
	}
	return []byte(got.Encode()), true
}

func maskJSON(body []byte) ([]byte, bool) {
	var (
		out     bytes.Buffer
		changed bool
	)
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	enc := json.NewEncoder(&out)
	enc.SetEscapeHTML(false)
	for {
		var v any
		err := dec.Decode(&v)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, false
		}
		got, ok, err := maskValues(v, maskBodyValue)
		if err != nil {
			return nil, false
		}
		changed = changed || ok
		err = enc.Encode(got)
		if err != nil {
			return nil, false
		}
	}
	if !changed {
		return body, true
	}
	return bytes.TrimSuffix(out.Bytes(), []byte("\n")), true
}

func maskBodyValue(path []string, _ string) (string, bool) {
	for _, p := range path {
		if _, ok := bodyNames[strings.ToLower(p)]; ok {
			return redacted, true
		}
	}
	return "", false
}

// bodyNames are the lower-case names of body fields whose values are masked,
// at any depth. The list is exact, so token_type and client_assertion_type
// stay visible.
var bodyNames = setOf(
	"access_token",
	"api_key",
	"apikey",
	"assertion",
	"client_assertion",
	"client_secret",
	"id_token",
	"password",
	"refresh_token",
	"secret",
	"token",
)

// maskValues returns a copy of v in which fn has been applied to each string
// value, and reports whether fn replaced any of them.
func maskValues[T any](v T, fn func(path []string, s string) (string, bool)) (_ T, changed bool, err error) {
	got, err := private.RedactFunc(v, "", private.MaskStrings(func(path []string, s string) (string, bool) {
		r, ok := fn(path, s)
		changed = changed || ok
		return r, ok
	}))
	return got, changed, err
}

func setOf(names ...string) map[string]struct{} {
	m := make(map[string]struct{}, len(names))
	for _, n := range names {
		m[n] = struct{}{}
	}
	return m
}
