// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

func TestHTTPJSONInput(t *testing.T) {
	testCases := []struct {
		name       string
		setup      func(map[string]interface{}) interface{}
		teardown   func(interface{})
		baseConfig map[string]interface{}
		ssl        bool
		handler    http.HandlerFunc
		expected   []string
		duration   time.Duration
	}{
		{
			name: "Test simple GET request",
			baseConfig: map[string]interface{}{
				"http_method": "GET",
				"interval":    0,
			},
			handler:  defaultHandler("GET", ""),
			expected: []string{`{"hello":[{"world":"moon"},{"space":[{"cake":"pumpkin"}]}]}`},
		},
		{
			name: "Test simple HTTPS GET request",
			baseConfig: map[string]interface{}{
				"http_method":           "GET",
				"interval":              0,
				"ssl.verification_mode": "none",
			},
			handler:  defaultHandler("GET", ""),
			expected: []string{`{"hello":[{"world":"moon"},{"space":[{"cake":"pumpkin"}]}]}`},
			ssl:      true,
		},
		{
			name: "Test request honors rate limit",
			baseConfig: map[string]interface{}{
				"http_method":          "GET",
				"interval":             0,
				"rate_limit.limit":     "X-Rate-Limit-Limit",
				"rate_limit.remaining": "X-Rate-Limit-Remaining",
				"rate_limit.reset":     "X-Rate-Limit-Reset",
			},
			handler:  rateLimitHandler(),
			expected: []string{`{"hello":"world"}`},
		},
		{
			name: "Test request retries when failed",
			baseConfig: map[string]interface{}{
				"http_method": "GET",
				"interval":    0,
			},
			handler:  retryHandler(),
			expected: []string{`{"hello":"world"}`},
		},
		{
			name: "Test POST request with body",
			baseConfig: map[string]interface{}{
				"http_method": "POST",
				"interval":    0,
				"http_request_body": map[string]interface{}{
					"test": "abc",
				},
			},
			handler:  defaultHandler("POST", `{"test":"abc"}`),
			expected: []string{`{"hello":[{"world":"moon"},{"space":[{"cake":"pumpkin"}]}]}`},
		},
		{
			name: "Test repeated POST requests",
			baseConfig: map[string]interface{}{
				"http_method": "POST",
				"interval":    "400ms",
			},
			handler: defaultHandler("POST", ""),
			expected: []string{
				`{"hello":[{"world":"moon"},{"space":[{"cake":"pumpkin"}]}]}`,
				`{"hello":[{"world":"moon"},{"space":[{"cake":"pumpkin"}]}]}`,
			},
			duration: 700 * time.Millisecond,
		},
		{
			name: "Test json objects array",
			baseConfig: map[string]interface{}{
				"http_method":        "GET",
				"interval":           0,
				"json_objects_array": "hello",
			},
			handler:  defaultHandler("GET", ""),
			expected: []string{`{"world":"moon"}`, `{"space":[{"cake":"pumpkin"}]}`},
		},
		{
			name: "Test split events by",
			baseConfig: map[string]interface{}{
				"http_method":     "GET",
				"interval":        0,
				"split_events_by": "hello",
			},
			handler: defaultHandler("GET", ""),
			expected: []string{
				`{"hello":{"world":"moon"}}`,
				`{"hello":{"space":[{"cake":"pumpkin"}]}}`,
			},
		},
		{
			name: "Test split events by with array",
			baseConfig: map[string]interface{}{
				"http_method":        "GET",
				"interval":           0,
				"split_events_by":    "space",
				"json_objects_array": "hello",
			},
			handler: defaultHandler("GET", ""),
			expected: []string{
				`{"world":"moon"}`,
				`{"space":{"cake":"pumpkin"}}`,
			},
		},
		{
			name: "Test split events by not found",
			baseConfig: map[string]interface{}{
				"http_method":     "GET",
				"interval":        0,
				"split_events_by": "unknwown",
			},
			handler:  defaultHandler("GET", ""),
			expected: []string{`{"hello":[{"world":"moon"},{"space":[{"cake":"pumpkin"}]}]}`},
		},
		{
			name: "Test date cursor",
			baseConfig: map[string]interface{}{
				"http_method":                  "GET",
				"interval":                     "400ms",
				"date_cursor.field":            "@timestamp",
				"date_cursor.url_field":        "$filter",
				"date_cursor.value_template":   "alertCreationTime ge {{.}}",
				"date_cursor.initial_interval": "10m",
				"date_cursor.date_format":      "2006-01-02T15:04:05Z",
			},
			handler: dateCursorHandler(),
			expected: []string{
				`{"@timestamp":"2002-10-02T15:00:00Z","foo":"bar"}`,
				`{"@timestamp":"2002-10-02T15:00:01Z","foo":"bar"}`,
			},
			duration: 700 * time.Millisecond,
			setup: func(map[string]interface{}) interface{} {
				timeNow = func() time.Time {
					t, _ := time.Parse(time.RFC3339, "2002-10-02T15:00:00Z")
					return t
				}
				return nil
			},
		},
		{
			name: "Test pagination",
			baseConfig: map[string]interface{}{
				"http_method":          "GET",
				"interval":             0,
				"pagination.id_field":  "nextPageToken",
				"pagination.url_field": "page",
				"json_objects_array":   "items",
			},
			handler:  paginationHandler(),
			expected: []string{`{"foo":"bar"}`, `{"foo":"bar"}`},
		},
		{
			name: "Test oauth2",
			baseConfig: map[string]interface{}{
				"http_method":          "POST",
				"interval":             "0",
				"oauth2.client.id":     "a_client_id",
				"oauth2.client.secret": "a_client_secret",
				"oauth2.endpoint_params": map[string]interface{}{
					"param1": "v1",
				},
				"oauth2.scopes": []string{"scope1", "scope2"},
			},
			setup: func(config map[string]interface{}) interface{} {
				server := httptest.NewServer(http.HandlerFunc(oauth2TokenHandler))
				config["oauth2.token_url"] = server.URL
				return server
			},
			teardown: func(i interface{}) {
				server := i.(*httptest.Server)
				server.Close()
			},
			handler:  oauth2Handler,
			expected: []string{`{"hello": "world"}`},
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			server := func() *httptest.Server {
				if tc.ssl {
					return httptest.NewTLSServer(tc.handler)
				}
				return httptest.NewServer(tc.handler)
			}()
			defer server.Close()

			tc.baseConfig["url"] = server.URL

			var setupResult interface{}
			if tc.setup != nil {
				setupResult = tc.setup(tc.baseConfig)
			}
			if tc.teardown != nil {
				defer tc.teardown(setupResult)
			}

			cfg := common.MustNewConfigFrom(tc.baseConfig)

			input, err := configure(cfg)

			assert.NoError(t, err)
			assert.Equal(t, "httpjson", input.Name())
			assert.NoError(t, input.Test(v2.TestContext{}))

			pub := &publisher{}

			ctx, cancel := newV2Context(tc.duration)
			defer cancel()

			assert.NoError(t, input.Run(ctx, pub))

			assert.Equal(t, len(tc.expected), len(pub.events))
			for i, e := range pub.events {
				val, err := e.Fields.GetValue("message")
				assert.NoError(t, err)
				assert.JSONEq(t, tc.expected[i], val.(string))
			}
		})
	}
}

func newV2Context(d time.Duration) (v2.Context, func()) {
	ctx, cancel := func() (context.Context, func()) {
		if d == 0 {
			return context.WithCancel(context.Background())
		}
		return context.WithTimeout(context.Background(), d)
	}()
	return v2.Context{
		Logger:      logp.NewLogger("httpjson_test"),
		ID:          "test_id",
		Cancelation: ctx,
	}, cancel
}

type publisher struct {
	events []beat.Event
}

func (p *publisher) Publish(e beat.Event) {
	p.events = append(p.events, e)
}

func defaultHandler(expectedMethod, expectedBody string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		msg := `{"hello":[{"world":"moon"},{"space":[{"cake":"pumpkin"}]}]}`
		switch {
		case r.Method != expectedMethod:
			w.WriteHeader(http.StatusBadRequest)
			msg = fmt.Sprintf(`{"error":"expected method was %q"}`, expectedMethod)
		case expectedBody != "":
			body, _ := ioutil.ReadAll(r.Body)
			r.Body.Close()
			if expectedBody != string(body) {
				w.WriteHeader(http.StatusBadRequest)
				msg = fmt.Sprintf(`{"error":"expected body was %q"}`, expectedBody)
			}
		}

		_, _ = w.Write([]byte(msg))
	}
}

func rateLimitHandler() http.HandlerFunc {
	var isRetry bool
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		if isRetry {
			_, _ = w.Write([]byte(`{"hello":"world"}`))
			return
		}
		w.Header().Set("X-Rate-Limit-Limit", "0")
		w.Header().Set("X-Rate-Limit-Remaining", "0")
		w.Header().Set("X-Rate-Limit-Reset", fmt.Sprint(time.Now().Unix()))
		w.WriteHeader(http.StatusTooManyRequests)
		isRetry = true
		_, _ = w.Write([]byte(`{"error":"too many requests"}`))
	}
}

func retryHandler() http.HandlerFunc {
	count := 0
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		if count == 2 {
			_, _ = w.Write([]byte(`{"hello":"world"}`))
			return
		}
		w.WriteHeader(rand.Intn(100) + 500)
		count += 1
		_, _ = w.Write([]byte(`{"error":"failed"}`))
	}
}

func oauth2TokenHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	_ = r.ParseForm()
	switch {
	case r.Method != "POST":
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"wrong method"}`))
	case r.FormValue("grant_type") != "client_credentials":
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"wrong grant_type"}`))
	case r.FormValue("client_id") != "a_client_id":
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"wrong client_id"}`))
	case r.FormValue("client_secret") != "a_client_secret":
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"wrong client_secret"}`))
	case r.FormValue("scope") != "scope1 scope2":
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"wrong scope"}`))
	case r.FormValue("param1") != "v1":
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"wrong param1"}`))
	default:
		_, _ = w.Write([]byte(`{"token_type": "Bearer", "expires_in": "60", "access_token": "abcd"}`))
	}
}

func oauth2Handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	switch {
	case r.Method != "POST":
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"wrong method"}`))
	case r.Header.Get("Authorization") != "Bearer abcd":
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"wrong bearer"}`))
	default:
		_, _ = w.Write([]byte(`{"hello":"world"}`))
	}
}

func dateCursorHandler() http.HandlerFunc {
	var count int
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		switch count {
		case 0:
			if r.URL.Query().Get("$filter") != "alertCreationTime ge 2002-10-02T14:50:00Z" {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":"wrong initial cursor value"`))
				return
			}
			_, _ = w.Write([]byte(`{"@timestamp":"2002-10-02T15:00:00Z","foo":"bar"}`))
		case 1:
			if r.URL.Query().Get("$filter") != "alertCreationTime ge 2002-10-02T15:00:00Z" {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":"wrong cursor value"`))
				return
			}
			_, _ = w.Write([]byte(`{"@timestamp":"2002-10-02T15:00:01Z","foo":"bar"}`))
		}
		count += 1
	}
}

func paginationHandler() http.HandlerFunc {
	var count int
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		switch count {
		case 0:
			_, _ = w.Write([]byte(`{"@timestamp":"2002-10-02T15:00:00Z","nextPageToken":"bar","items":[{"foo":"bar"}]}`))
		case 1:
			if r.URL.Query().Get("page") != "bar" {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":"wrong page token value"}`))
				return
			}
			_, _ = w.Write([]byte(`{"@timestamp":"2002-10-02T15:00:01Z","items":[{"foo":"bar"}]}`))
		}
		count += 1
	}
}
