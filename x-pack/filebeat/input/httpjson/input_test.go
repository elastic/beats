// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	beattest "github.com/elastic/beats/v7/libbeat/publisher/testing"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

var testCases = []struct {
	name         string
	setupServer  func(testing.TB, http.HandlerFunc, map[string]interface{})
	baseConfig   map[string]interface{}
	handler      http.HandlerFunc
	expected     []string
	expectedFile string

	skipReason string
}{
	{
		name:        "simple_GET_request",
		setupServer: newTestServer(httptest.NewServer),
		baseConfig: map[string]interface{}{
			"interval":       1,
			"request.method": http.MethodGet,
		},
		handler:  defaultHandler(http.MethodGet, "", ""),
		expected: []string{`{"hello":[{"world":"moon"},{"space":[{"cake":"pumpkin"}]}]}`},
	},
	{
		name:        "simple_HTTPS_GET_request",
		setupServer: newTestServer(httptest.NewTLSServer),
		baseConfig: map[string]interface{}{
			"interval":                      1,
			"request.method":                http.MethodGet,
			"request.ssl.verification_mode": "none",
		},
		handler:  defaultHandler(http.MethodGet, "", ""),
		expected: []string{`{"hello":[{"world":"moon"},{"space":[{"cake":"pumpkin"}]}]}`},
	},
	{
		name:        "request_honors_rate_limit",
		setupServer: newTestServer(httptest.NewServer),
		baseConfig: map[string]interface{}{
			"interval":                     1,
			"http_method":                  http.MethodGet,
			"request.rate_limit.limit":     `[[.last_response.header.Get "X-Rate-Limit-Limit"]]`,
			"request.rate_limit.remaining": `[[.last_response.header.Get "X-Rate-Limit-Remaining"]]`,
			"request.rate_limit.reset":     `[[.last_response.header.Get "X-Rate-Limit-Reset"]]`,
		},
		handler:  rateLimitHandler(),
		expected: []string{`{"hello":"world"}`},
	},
	{
		name:        "request_retries_when_failed",
		setupServer: newTestServer(httptest.NewServer),
		baseConfig: map[string]interface{}{
			"interval":       1,
			"request.method": http.MethodGet,
		},
		handler:  retryHandler(),
		expected: []string{`{"hello":"world"}`},
	},
	{
		name:        "POST_request_with_body",
		setupServer: newTestServer(httptest.NewServer),
		baseConfig: map[string]interface{}{
			"interval":       1,
			"request.method": http.MethodPost,
			"request.body": map[string]interface{}{
				"test": "abc",
			},
		},
		handler:  defaultHandler(http.MethodPost, `{"test":"abc"}`, ""),
		expected: []string{`{"hello":[{"world":"moon"},{"space":[{"cake":"pumpkin"}]}]}`},
	},
	{
		name:        "POST_request_with_empty_object_body",
		setupServer: newTestServer(httptest.NewServer),
		baseConfig: map[string]interface{}{
			"interval":       1,
			"request.method": http.MethodPost,
			"request.body":   map[string]interface{}{},
		},
		handler:  defaultHandler(http.MethodPost, `{}`, ""),
		expected: []string{`{"hello":[{"world":"moon"},{"space":[{"cake":"pumpkin"}]}]}`},
	},
	{
		name:        "repeated_POST_requests",
		setupServer: newTestServer(httptest.NewServer),
		baseConfig: map[string]interface{}{
			"interval":       "100ms",
			"request.method": http.MethodPost,
		},
		handler: defaultHandler(http.MethodPost, "", ""),
		expected: []string{
			`{"hello":[{"world":"moon"},{"space":[{"cake":"pumpkin"}]}]}`,
			`{"hello":[{"world":"moon"},{"space":[{"cake":"pumpkin"}]}]}`,
		},
	},
	{
		name:        "split_by_json_objects_array",
		setupServer: newTestServer(httptest.NewServer),
		baseConfig: map[string]interface{}{
			"interval":       1,
			"request.method": http.MethodGet,
			"response.split": map[string]interface{}{
				"target": "body.hello",
			},
		},
		handler:  defaultHandler(http.MethodGet, "", ""),
		expected: []string{`{"world":"moon"}`, `{"space":[{"cake":"pumpkin"}]}`},
	},
	{
		name:        "split_by_json_objects_array_with_keep_parent",
		setupServer: newTestServer(httptest.NewServer),
		baseConfig: map[string]interface{}{
			"interval":       1,
			"request.method": http.MethodGet,
			"response.split": map[string]interface{}{
				"target":      "body.hello",
				"keep_parent": true,
			},
		},
		handler: defaultHandler(http.MethodGet, "", ""),
		expected: []string{
			`{"hello":{"world":"moon"}}`,
			`{"hello":{"space":[{"cake":"pumpkin"}]}}`,
		},
	},
	{
		name:        "split_on_empty_array_without_ignore_empty_value",
		setupServer: newTestServer(httptest.NewServer),
		baseConfig: map[string]interface{}{
			"interval":       1,
			"request.method": http.MethodGet,
			"response.split": map[string]interface{}{
				"target": "body.response.empty",
			},
		},
		handler:  defaultHandler(http.MethodGet, "", `{"response":{"empty":[]}}`),
		expected: []string{`{"response":{"empty":[]}}`},
	},
	{
		name:        "split_on_empty_array_with_ignore_empty_value",
		setupServer: newTestServer(httptest.NewServer),
		baseConfig: map[string]interface{}{
			"interval":       1,
			"request.method": http.MethodGet,
			"response.split": map[string]interface{}{
				"target":             "body.response.empty",
				"ignore_empty_value": true,
			},
		},
		handler:  defaultHandler(http.MethodGet, "", `{"response":{"empty":[]}}`),
		expected: nil,
	},
	{
		name:        "split_on_null_field_with_ignore_empty_value_keeping_parent",
		setupServer: newTestServer(httptest.NewServer),
		baseConfig: map[string]interface{}{
			"interval":       1,
			"request.method": http.MethodGet,
			"response.split": map[string]interface{}{
				"target":             "body.response.empty",
				"ignore_empty_value": true,
				"keep_parent":        true,
			},
		},
		handler:  defaultHandler(http.MethodGet, "", `{"response":{"empty":null}}`),
		expected: []string{`{"response":{"empty":null}}`},
	},
	{
		name:        "split_on_empty_array_with_ignore_empty_value_keeping_parent",
		setupServer: newTestServer(httptest.NewServer),
		baseConfig: map[string]interface{}{
			"interval":       1,
			"request.method": http.MethodGet,
			"response.split": map[string]interface{}{
				"target":             "body.response.empty",
				"ignore_empty_value": true,
				"keep_parent":        true,
			},
		},
		handler:  defaultHandler(http.MethodGet, "", `{"response":{"empty":[]}}`),
		expected: []string{`{"response":{"empty":[]}}`},
	},
	{
		name:        "split_on_null_field_at_root_with_ignore_empty_value_keeping_parent",
		setupServer: newTestServer(httptest.NewServer),
		baseConfig: map[string]interface{}{
			"interval":       1,
			"request.method": http.MethodGet,
			"response.split": map[string]interface{}{
				"target":             "body.response",
				"ignore_empty_value": true,
				"keep_parent":        true,
			},
		},
		handler:  defaultHandler(http.MethodGet, "", `{"response":null,"other":"data"}`),
		expected: []string{`{"other":"data","response":null}`},
	},
	{
		name:        "split_on_empty_array_at_root_with_ignore_empty_value_keeping_parent",
		setupServer: newTestServer(httptest.NewServer),
		baseConfig: map[string]interface{}{
			"interval":       1,
			"request.method": http.MethodGet,
			"response.split": map[string]interface{}{
				"target":             "body.response",
				"ignore_empty_value": true,
				"keep_parent":        true,
			},
		},
		handler:  defaultHandler(http.MethodGet, "", `{"response":[],"other":"data"}`),
		expected: []string{`{"other":"data","response":[]}`},
	},
	{
		name:        "nested_split",
		setupServer: newTestServer(httptest.NewServer),
		baseConfig: map[string]interface{}{
			"interval":       1,
			"request.method": http.MethodGet,
			"response.split": map[string]interface{}{
				"target": "body.hello",
				"split": map[string]interface{}{
					"target":      "body.space",
					"keep_parent": true,
				},
			},
		},
		handler: defaultHandler(http.MethodGet, "", ""),
		expected: []string{
			`{"world":"moon"}`,
			`{"space":{"cake":"pumpkin"}}`,
		},
	},
	{
		name:        "split_events_by_not_found",
		setupServer: newTestServer(httptest.NewServer),
		baseConfig: map[string]interface{}{
			"interval":       1,
			"request.method": http.MethodGet,
			"response.split": map[string]interface{}{
				"target": "body.unknown",
			},
		},
		handler:  defaultHandler(http.MethodGet, "", ""),
		expected: []string{},
	},
	{
		name: "date_cursor",
		setupServer: func(t testing.TB, h http.HandlerFunc, config map[string]interface{}) {
			// mock timeNow func to return a fixed value
			timeNow = func() time.Time {
				t, _ := time.Parse(time.RFC3339, "2002-10-02T15:00:00Z")
				return t
			}

			server := httptest.NewServer(h)
			config["request.url"] = server.URL
			t.Cleanup(server.Close)
			t.Cleanup(func() { timeNow = time.Now })
		},
		baseConfig: map[string]interface{}{
			"interval":       1,
			"request.method": http.MethodGet,
			"request.transforms": []interface{}{
				map[string]interface{}{
					"set": map[string]interface{}{
						"target":  "url.params.$filter",
						"value":   "alertCreationTime ge [[.cursor.timestamp]]",
						"default": `alertCreationTime ge [[formatDate (now (parseDuration "-10m")) "2006-01-02T15:04:05Z"]]`,
					},
				},
			},
			"cursor": map[string]interface{}{
				"timestamp": map[string]interface{}{
					"value": `[[index .last_response.body "@timestamp"]]`,
				},
			},
		},
		handler: dateCursorHandler(),
		expected: []string{
			`{"@timestamp":"2002-10-02T15:00:00Z","foo":"bar"}`,
			`{"@timestamp":"2002-10-02T15:00:01Z","foo":"bar"}`,
			`{"@timestamp":"2002-10-02T15:00:02Z","foo":"bar"}`,
		},
	},
	{
		name: "tracer_filename_sanitization",
		setupServer: func(t testing.TB, h http.HandlerFunc, config map[string]interface{}) {
			// mock timeNow func to return a fixed value
			timeNow = func() time.Time {
				t, _ := time.Parse(time.RFC3339, "2002-10-02T15:00:00Z")
				return t
			}

			server := httptest.NewServer(h)
			config["request.url"] = server.URL
			t.Cleanup(server.Close)
			t.Cleanup(func() { timeNow = time.Now })
		},
		baseConfig: map[string]interface{}{
			"interval":       1,
			"request.method": http.MethodGet,
			"request.transforms": []interface{}{
				map[string]interface{}{
					"set": map[string]interface{}{
						"target":  "url.params.$filter",
						"value":   "alertCreationTime ge [[.cursor.timestamp]]",
						"default": `alertCreationTime ge [[formatDate (now (parseDuration "-10m")) "2006-01-02T15:04:05Z"]]`,
					},
				},
			},
			"cursor": map[string]interface{}{
				"timestamp": map[string]interface{}{
					"value": `[[index .last_response.body "@timestamp"]]`,
				},
			},
			"request.tracer.filename": "logs/http-request-trace-*.ndjson",
		},
		handler: dateCursorHandler(),
		expected: []string{
			`{"@timestamp":"2002-10-02T15:00:00Z","foo":"bar"}`,
			`{"@timestamp":"2002-10-02T15:00:01Z","foo":"bar"}`,
			`{"@timestamp":"2002-10-02T15:00:02Z","foo":"bar"}`,
		},
		expectedFile: filepath.Join("logs", "http-request-trace-httpjson-foo-eb837d4c-5ced-45ed-b05c-de658135e248_https_somesource_someapi.ndjson"),
	},
	{
		name: "pagination",
		setupServer: func(t testing.TB, h http.HandlerFunc, config map[string]interface{}) {
			server := httptest.NewServer(h)
			config["request.url"] = server.URL
			t.Cleanup(server.Close)
		},
		baseConfig: map[string]interface{}{
			"interval":       time.Millisecond,
			"request.method": http.MethodGet,
			"response.split": map[string]interface{}{
				"target": "body.items",
				"transforms": []interface{}{
					map[string]interface{}{
						"set": map[string]interface{}{
							"target": "body.page",
							"value":  "[[.last_response.page]]",
						},
					},
				},
			},
			"response.pagination": []interface{}{
				map[string]interface{}{
					"set": map[string]interface{}{
						"target":                 "url.params.page",
						"value":                  "[[.last_response.body.nextPageToken]]",
						"fail_on_template_error": true,
					},
				},
			},
		},
		handler: paginationHandler(),
		expected: []string{
			`{"foo":"a","page":"0"}`, `{"foo":"b","page":"1"}`, `{"foo":"c","page":"0"}`, `{"foo":"d","page":"0"}`,
			`{"foo":"a","page":"0"}`, `{"foo":"b","page":"1"}`, `{"foo":"c","page":"0"}`, `{"foo":"d","page":"0"}`,
		},
	},
	{
		name: "first_event",
		setupServer: func(t testing.TB, h http.HandlerFunc, config map[string]interface{}) {
			server := httptest.NewServer(h)
			config["request.url"] = server.URL
			t.Cleanup(server.Close)
		},
		baseConfig: map[string]interface{}{
			"interval":       1,
			"request.method": http.MethodGet,
			"response.split": map[string]interface{}{
				"target": "body.items",
				"transforms": []interface{}{
					map[string]interface{}{
						"set": map[string]interface{}{
							"target":  "body.first",
							"value":   "[[.cursor.first]]",
							"default": "none",
						},
					},
				},
			},
			"response.pagination": []interface{}{
				map[string]interface{}{
					"set": map[string]interface{}{
						"target":                 "url.params.page",
						"value":                  "[[.last_response.body.nextPageToken]]",
						"fail_on_template_error": true,
					},
				},
			},
			"cursor": map[string]interface{}{
				"first": map[string]interface{}{
					"value": "[[.first_event.foo]]",
				},
			},
		},
		handler:  paginationHandler(),
		expected: []string{`{"first":"none", "foo":"a"}`, `{"first":"a", "foo":"b"}`, `{"first":"a", "foo":"c"}`, `{"first":"c", "foo":"d"}`},
	},
	{
		name: "pagination_with_array_response",
		setupServer: func(t testing.TB, h http.HandlerFunc, config map[string]interface{}) {
			server := httptest.NewServer(h)
			config["request.url"] = server.URL
			t.Cleanup(server.Close)
		},
		baseConfig: map[string]interface{}{
			"interval":       1,
			"request.method": http.MethodGet,
			"response.pagination": []interface{}{
				map[string]interface{}{
					"set": map[string]interface{}{
						"target": "url.params.page",
						"value":  `[[index (index .last_response.body 0) "nextPageToken"]]`,
					},
				},
			},
		},
		handler:  paginationArrayHandler(),
		expected: []string{`{"nextPageToken":"bar","foo":"bar"}`, `{"foo":"bar"}`, `{"foo":"bar"}`},
	},
	{
		name: "oauth2",
		setupServer: func(t testing.TB, h http.HandlerFunc, config map[string]interface{}) {
			server := httptest.NewServer(h)
			config["request.url"] = server.URL
			config["auth.oauth2.token_url"] = server.URL + "/token"
			t.Cleanup(server.Close)
		},
		baseConfig: map[string]interface{}{
			"interval":                  1,
			"request.method":            http.MethodPost,
			"auth.oauth2.client.id":     "a_client_id",
			"auth.oauth2.client.secret": "a_client_secret",
			"auth.oauth2.endpoint_params": map[string]interface{}{
				"param1": "v1",
			},
			"auth.oauth2.scopes": []string{"scope1", "scope2"},
		},
		handler:  oauth2Handler,
		expected: []string{`{"hello": "world"}`},
	},
	{
		name: "request_transforms_can_access_state_from_previous_transforms",
		setupServer: func(t testing.TB, h http.HandlerFunc, config map[string]interface{}) {
			server := httptest.NewServer(h)
			config["request.url"] = server.URL + "/test-path"
			t.Cleanup(server.Close)
		},
		baseConfig: map[string]interface{}{
			"interval":       1,
			"request.method": http.MethodPost,
			"request.transforms": []interface{}{
				map[string]interface{}{
					"set": map[string]interface{}{
						"target": "header.X-Foo",
						"value":  "foo",
					},
				},
				map[string]interface{}{
					"set": map[string]interface{}{
						"target": "body.bar",
						"value":  `[[.header.Get "X-Foo"]]`,
					},
				},
				map[string]interface{}{
					"set": map[string]interface{}{
						"target": "body.url.path",
						"value":  `[[.url.Path]]`,
					},
				},
			},
		},
		handler:  defaultHandler(http.MethodPost, `{"bar":"foo","url":{"path":"/test-path"}}`, ""),
		expected: []string{`{"hello":[{"world":"moon"},{"space":[{"cake":"pumpkin"}]}]}`},
	},
	{
		name: "response_transforms_can't_access_request_state_from_previous_transforms",
		setupServer: func(t testing.TB, h http.HandlerFunc, config map[string]interface{}) {
			server := httptest.NewServer(h)
			config["request.url"] = server.URL
			t.Cleanup(server.Close)
		},
		baseConfig: map[string]interface{}{
			"interval":       10,
			"request.method": http.MethodGet,
			"request.transforms": []interface{}{
				map[string]interface{}{
					"set": map[string]interface{}{
						"target": "header.X-Foo",
						"value":  "foo",
					},
				},
			},
			"response.transforms": []interface{}{
				map[string]interface{}{
					"set": map[string]interface{}{
						"target": "body.bar",
						"value":  `[[.header.Get "X-Foo"]]`,
					},
				},
			},
		},
		handler:  defaultHandler(http.MethodGet, "", ""),
		expected: []string{`{"hello":[{"world":"moon"},{"space":[{"cake":"pumpkin"}]}]}`},
	},
	{
		name:        "simple_Chain_GET_request",
		setupServer: newChainTestServer(httptest.NewServer),
		baseConfig: map[string]interface{}{
			"interval":       10,
			"request.method": http.MethodGet,
			"chain": []interface{}{
				map[string]interface{}{
					"step": map[string]interface{}{
						"request.method": http.MethodGet,
						"replace":        "$.records[:].id",
					},
				},
			},
		},
		handler:  defaultHandler(http.MethodGet, "", ""),
		expected: []string{`{"hello":[{"world":"moon"},{"space":[{"cake":"pumpkin"}]}]}`},
	},
	{
		name:        "simple_naked_Chain_GET_request",
		setupServer: newNakedChainTestServer(httptest.NewServer),
		baseConfig: map[string]interface{}{
			"interval":       10,
			"request.method": http.MethodGet,
			"chain": []interface{}{
				map[string]interface{}{
					"step": map[string]interface{}{
						"request.url":    "placeholder:$.records[:]",
						"request.method": http.MethodGet,
						"replace":        "$.records[:]",
					},
				},
			},
		},
		handler:  defaultHandler(http.MethodGet, "", ""),
		expected: []string{`{"hello":[{"world":"moon"},{"space":[{"cake":"pumpkin"}]}]}`},
	},
	{
		name: "multiple_Chain_GET_request",
		setupServer: func(t testing.TB, h http.HandlerFunc, config map[string]interface{}) {
			r := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/":
					fmt.Fprintln(w, `{"records":[{"id":1}]}`)
				case "/1":
					fmt.Fprintln(w, `{"file_name": "file_1"}`)
				case "/file_1":
					fmt.Fprintln(w, `{"hello":[{"world":"moon"},{"space":[{"cake":"pumpkin"}]}]}`)
				}
			})
			server := httptest.NewServer(r)
			config["request.url"] = server.URL
			config["chain.0.step.request.url"] = server.URL + "/$.records[:].id"
			config["chain.1.step.request.url"] = server.URL + "/$.file_name"
			t.Cleanup(server.Close)
		},
		baseConfig: map[string]interface{}{
			"interval":       10,
			"request.method": http.MethodGet,
			"chain": []interface{}{
				map[string]interface{}{
					"step": map[string]interface{}{
						"request.method": http.MethodGet,
						"replace":        "$.records[:].id",
					},
				},
				map[string]interface{}{
					"step": map[string]interface{}{
						"request.method": http.MethodGet,
						"replace":        "$.file_name",
					},
				},
			},
		},
		handler:  defaultHandler(http.MethodGet, "", ""),
		expected: []string{`{"hello":[{"world":"moon"},{"space":[{"cake":"pumpkin"}]}]}`},
	},
	{
		name: "date_cursor_while_using_chain",
		setupServer: func(t testing.TB, h http.HandlerFunc, config map[string]interface{}) {
			// mock timeNow func to return a fixed value
			timeNow = func() time.Time {
				t, _ := time.Parse(time.RFC3339, "2002-10-02T15:00:00Z")
				return t
			}

			r := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/":
					fmt.Fprintln(w, `{"records":[{"id":1}]}`)
				case "/1":
					fmt.Fprintln(w, `{"hello":[{"world":"moon"},{"space":[{"cake":"pumpkin"}]}]}`)
				}
			})
			server := httptest.NewServer(r)
			config["request.url"] = server.URL
			config["chain.0.step.request.url"] = server.URL + "/$.records[:].id"
			t.Cleanup(server.Close)
			t.Cleanup(func() { timeNow = time.Now })
		},
		baseConfig: map[string]interface{}{
			"interval":       1,
			"request.method": http.MethodGet,
			"request.transforms": []interface{}{
				map[string]interface{}{
					"set": map[string]interface{}{
						"target":  "url.params.$filter",
						"value":   "alertCreationTime ge [[.cursor.timestamp]]",
						"default": `alertCreationTime ge [[formatDate (now (parseDuration "-10m")) "2006-01-02T15:04:05Z"]]`,
					},
				},
			},
			"chain": []interface{}{
				map[string]interface{}{
					"step": map[string]interface{}{
						"request.method": http.MethodGet,
						"replace":        "$.records[:].id",
					},
				},
			},
			"cursor": map[string]interface{}{
				"timestamp": map[string]interface{}{
					"value": `[[index .last_response.body "@timestamp"]]`,
				},
			},
		},
		handler:  dateCursorHandler(),
		expected: []string{`{"hello":[{"world":"moon"},{"space":[{"cake":"pumpkin"}]}]}`},
	},
	{
		name:        "split_by_json_objects_array_in_chain",
		setupServer: newChainTestServer(httptest.NewServer),
		baseConfig: map[string]interface{}{
			"interval":       1,
			"request.method": http.MethodGet,
			"chain": []interface{}{
				map[string]interface{}{
					"step": map[string]interface{}{
						"request.method": http.MethodGet,
						"replace":        "$.records[:].id",
						"response.split": map[string]interface{}{
							"target": "body.hello",
						},
					},
				},
			},
		},
		handler:  defaultHandler(http.MethodGet, "", ""),
		expected: []string{`{"world":"moon"}`, `{"space":[{"cake":"pumpkin"}]}`},
	},
	{
		name:        "split_by_json_objects_array_with_keep_parent_in_chain",
		setupServer: newChainTestServer(httptest.NewServer),
		baseConfig: map[string]interface{}{
			"interval":       1,
			"request.method": http.MethodGet,
			"chain": []interface{}{
				map[string]interface{}{
					"step": map[string]interface{}{
						"request.method": http.MethodGet,
						"replace":        "$.records[:].id",
						"response.split": map[string]interface{}{
							"target":      "body.hello",
							"keep_parent": true,
						},
					},
				},
			},
		},
		handler: defaultHandler(http.MethodGet, "", ""),
		expected: []string{
			`{"hello":{"world":"moon"}}`,
			`{"hello":{"space":[{"cake":"pumpkin"}]}}`,
		},
	},
	{
		name:        "nested_split_in_chain",
		setupServer: newChainTestServer(httptest.NewServer),
		baseConfig: map[string]interface{}{
			"interval":       1,
			"request.method": http.MethodGet,
			"response.split": map[string]interface{}{
				"target": "body.hello",
			},
			"chain": []interface{}{
				map[string]interface{}{
					"step": map[string]interface{}{
						"request.method": http.MethodGet,
						"replace":        "$.records[:].id",
						"response.split": map[string]interface{}{
							"target": "body.hello",
							"split": map[string]interface{}{
								"target":      "body.space",
								"keep_parent": true,
							},
						},
					},
				},
			},
		},
		handler: defaultHandler(http.MethodGet, "", ""),
		expected: []string{
			`{"world":"moon"}`,
			`{"space":{"cake":"pumpkin"}}`,
		},
	},
	{
		name:        "pagination_when_used_with_chaining",
		setupServer: newChainPaginationTestServer(httptest.NewServer),
		baseConfig: map[string]interface{}{
			"interval":       1,
			"request.method": http.MethodGet,
			"response.pagination": []interface{}{
				map[string]interface{}{
					"set": map[string]interface{}{
						"target":                 "url.value",
						"value":                  "[[.last_response.body.nextLink]]",
						"fail_on_template_error": true,
					},
				},
			},
			"chain": []interface{}{
				map[string]interface{}{
					"step": map[string]interface{}{
						"request.method": http.MethodGet,
						"replace":        "$.records[:].id",
					},
				},
			},
		},
		handler: defaultHandler(http.MethodGet, "", ""),
		expected: []string{
			`{"hello":{"world":"moon"}}`,
			`{"space":{"cake":"pumpkin"}}`,
		},
	},
	{
		name: "replace_with_clause_and_first_response_object",
		setupServer: func(t testing.TB, h http.HandlerFunc, config map[string]interface{}) {
			r := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/":
					fmt.Fprintln(w, `{"exportId":"2212"}`)
				case "/2212":
					fmt.Fprintln(w, `{"files":[{"id":"1"},{"id":"2"}]}`)
				case "/2212/1":
					fmt.Fprintln(w, `{"hello":{"world":"moon"}}`)
				case "/2212/2":
					fmt.Fprintln(w, `{"space":{"cake":"pumpkin"}}`)
				}
			})
			server := httptest.NewServer(r)
			config["request.url"] = server.URL
			config["chain.0.step.request.url"] = server.URL + "/$.exportId"
			config["chain.1.step.request.url"] = server.URL + "/$.exportId/$.files[:].id"
			t.Cleanup(server.Close)
		},
		baseConfig: map[string]interface{}{
			"interval":                     1,
			"request.method":               http.MethodGet,
			"response.save_first_response": true,
			"chain": []interface{}{
				map[string]interface{}{
					"step": map[string]interface{}{
						"request.method": http.MethodGet,
						"replace":        "$.exportId",
					},
				},
				map[string]interface{}{
					"step": map[string]interface{}{
						"request.method": http.MethodGet,
						"replace":        "$.files[:].id",
						"replace_with":   "$.exportId,.first_response.body.exportId",
					},
				},
			},
		},
		expected: []string{
			`{"hello":{"world":"moon"}}`,
			`{"space":{"cake":"pumpkin"}}`,
		},
	},
	{
		name: "replace_with_clause_with_hardcoded_value_1",
		setupServer: func(t testing.TB, h http.HandlerFunc, config map[string]interface{}) {
			r := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/":
					fmt.Fprintln(w, `{"files":[{"id":"1"},{"id":"2"}]}`)
				case "/2212/1":
					fmt.Fprintln(w, `{"hello":{"world":"moon"}}`)
				case "/2212/2":
					fmt.Fprintln(w, `{"space":{"cake":"pumpkin"}}`)
				}
			})
			server := httptest.NewServer(r)
			config["request.url"] = server.URL
			config["chain.0.step.request.url"] = server.URL + "/$.exportId/$.files[:].id"
			t.Cleanup(server.Close)
		},
		baseConfig: map[string]interface{}{
			"interval":       1,
			"request.method": http.MethodGet,
			"chain": []interface{}{
				map[string]interface{}{
					"step": map[string]interface{}{
						"request.method": http.MethodGet,
						"replace":        "$.files[:].id",
						"replace_with":   "$.exportId,2212",
					},
				},
			},
		},
		expected: []string{
			`{"hello":{"world":"moon"}}`,
			`{"space":{"cake":"pumpkin"}}`,
		},
	},
	{
		name: "replace_with_clause_with_hardcoded_value_(no_dot_prefix)",
		setupServer: func(t testing.TB, h http.HandlerFunc, config map[string]interface{}) {
			r := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/":
					fmt.Fprintln(w, `{"files":[{"id":"1"},{"id":"2"}]}`)
				case "/first_response.body.id/1":
					fmt.Fprintln(w, `{"hello":{"world":"moon"}}`)
				case "/first_response.body.id/2":
					fmt.Fprintln(w, `{"space":{"cake":"pumpkin"}}`)
				}
			})
			server := httptest.NewServer(r)
			config["request.url"] = server.URL
			config["chain.0.step.request.url"] = server.URL + "/$.exportId/$.files[:].id"
			t.Cleanup(server.Close)
		},
		baseConfig: map[string]interface{}{
			"interval":                     1,
			"request.method":               http.MethodGet,
			"response.save_first_response": true,
			"chain": []interface{}{
				map[string]interface{}{
					"step": map[string]interface{}{
						"request.method": http.MethodGet,
						"replace":        "$.files[:].id",
						"replace_with":   "$.exportId,first_response.body.id",
					},
				},
			},
		},
		expected: []string{
			`{"hello":{"world":"moon"}}`,
			`{"space":{"cake":"pumpkin"}}`,
		},
	},
	{
		name: "replace_with_clause_with_hardcoded_value_(more_than_one_dot_prefix)",
		setupServer: func(t testing.TB, h http.HandlerFunc, config map[string]interface{}) {
			r := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/":
					fmt.Fprintln(w, `{"files":[{"id":"1"},{"id":"2"}]}`)
				case "/..first_response.body.id/1":
					fmt.Fprintln(w, `{"hello":{"world":"moon"}}`)
				case "/..first_response.body.id/2":
					fmt.Fprintln(w, `{"space":{"cake":"pumpkin"}}`)
				}
			})
			server := httptest.NewServer(r)
			config["request.url"] = server.URL
			config["chain.0.step.request.url"] = server.URL + "/$.exportId/$.files[:].id"
			t.Cleanup(server.Close)
		},
		baseConfig: map[string]interface{}{
			"interval":                     1,
			"request.method":               http.MethodGet,
			"response.save_first_response": true,
			"chain": []interface{}{
				map[string]interface{}{
					"step": map[string]interface{}{
						"request.method": http.MethodGet,
						"replace":        "$.files[:].id",
						"replace_with":   "$.exportId,..first_response.body.id",
					},
				},
			},
		},
		expected: []string{
			`{"hello":{"world":"moon"}}`,
			`{"space":{"cake":"pumpkin"}}`,
		},
	},
	{
		name: "replace_with_clause_with_hardcoded_value_containing_'.'_(dots)",
		setupServer: func(t testing.TB, h http.HandlerFunc, config map[string]interface{}) {
			r := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/":
					fmt.Fprintln(w, `{"files":[{"id":"1"},{"id":"2"}]}`)
				case "/.xyz.2212.abc./1":
					fmt.Fprintln(w, `{"hello":{"world":"moon"}}`)
				case "/.xyz.2212.abc./2":
					fmt.Fprintln(w, `{"space":{"cake":"pumpkin"}}`)
				}
			})
			server := httptest.NewServer(r)
			config["request.url"] = server.URL
			config["chain.0.step.request.url"] = server.URL + "/$.exportId/$.files[:].id"
			t.Cleanup(server.Close)
		},
		baseConfig: map[string]interface{}{
			"interval":       1,
			"request.method": http.MethodGet,
			"chain": []interface{}{
				map[string]interface{}{
					"step": map[string]interface{}{
						"request.method": http.MethodGet,
						"replace":        "$.files[:].id",
						"replace_with":   "$.exportId,.xyz.2212.abc.",
					},
				},
			},
		},
		expected: []string{
			`{"hello":{"world":"moon"}}`,
			`{"space":{"cake":"pumpkin"}}`,
		},
	},
	{
		name: "global_transform_context_separation_with_parent_last_response_object",
		setupServer: func(t testing.TB, h http.HandlerFunc, config map[string]interface{}) {
			var serverURL string
			r := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/":
					fmt.Fprintf(w, `{"files":[{"id":"1"},{"id":"2"}],"exportId":"2212", "nextLink":"%s/link1"}`, serverURL)
				case "/link1":
					fmt.Fprintln(w, `{"files":[{"id":"3"},{"id":"4"}], "exportId":"2213"}`)
				case "/2212/1":
					matchBody(w, r, `{"exportId":"2212"}`, `{"hello":{"world":"moon"}}`)
				case "/2212/2":
					matchBody(w, r, `{"exportId":"2212"}`, `{"space":{"cake":"pumpkin"}}`)
				case "/2213/3":
					matchBody(w, r, `{"exportId":"2213"}`, `{"hello":{"cake":"pumpkin"}}`)
				case "/2213/4":
					matchBody(w, r, `{"exportId":"2213"}`, `{"space":{"world":"moon"}}`)
				}
			})
			server := httptest.NewServer(r)
			config["request.url"] = server.URL
			serverURL = server.URL
			config["chain.0.step.request.url"] = server.URL + "/$.exportId/$.files[:].id"
			t.Cleanup(server.Close)
		},
		baseConfig: map[string]interface{}{
			"interval":                            1,
			"request.method":                      http.MethodPost,
			"response.request_body_on_pagination": true,
			"response.pagination": []interface{}{
				map[string]interface{}{
					"set": map[string]interface{}{
						"target":                 "url.value",
						"value":                  "[[.last_response.body.nextLink]]",
						"fail_on_template_error": true,
					},
				},
			},
			"chain": []interface{}{
				map[string]interface{}{
					"step": map[string]interface{}{
						"request.method": http.MethodPost,
						"replace":        "$.files[:].id",
						"replace_with":   "$.exportId,.parent_last_response.body.exportId",
						"request.transforms": []interface{}{
							map[string]interface{}{
								"set": map[string]interface{}{
									"target": "body.exportId",
									"value":  "[[ .parent_last_response.body.exportId ]]",
								},
							},
						},
					},
				},
			},
		},
		expected: []string{
			`{"hello":{"world":"moon"}}`,
			`{"space":{"cake":"pumpkin"}}`,
			`{"hello":{"cake":"pumpkin"}}`,
			`{"space":{"world":"moon"}}`,
		},
	},
	{
		name: "cursor_value_is_updated_for_root_response_with_chaining_&_pagination",
		setupServer: func(t testing.TB, h http.HandlerFunc, config map[string]interface{}) {
			var serverURL string
			r := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/":
					fmt.Fprintf(w, `{"files":[{"id":"1"},{"id":"2"}],"exportId":"2212", "createdAt":"22/02/2022",
						"nextLink":"%s/link1"}`, serverURL)
				case "/link1":
					fmt.Fprintln(w, `{"files":[{"id":"3"},{"id":"4"}], "exportId":"2213", "createdAt":"24/04/2022"}`)
				case "/2212/1":
					matchBody(w, r, `{"createdAt":"22/02/2022","exportId":"2212"}`, `{"hello":{"world":"moon"}}`)
				case "/2212/2":
					matchBody(w, r, `{"createdAt":"22/02/2022","exportId":"2212"}`, `{"space":{"cake":"pumpkin"}}`)
				case "/2213/3":
					matchBody(w, r, `{"createdAt":"24/04/2022","exportId":"2213"}`, `{"hello":{"cake":"pumpkin"}}`)
				case "/2213/4":
					matchBody(w, r, `{"createdAt":"24/04/2022","exportId":"2213"}`, `{"space":{"world":"moon"}}`)
				}
			})
			server := httptest.NewServer(r)
			config["request.url"] = server.URL
			serverURL = server.URL
			config["chain.0.step.request.url"] = server.URL + "/$.exportId/$.files[:].id"
			t.Cleanup(server.Close)
		},
		baseConfig: map[string]interface{}{
			"interval":                            1,
			"request.method":                      http.MethodPost,
			"response.request_body_on_pagination": true,
			"response.pagination": []interface{}{
				map[string]interface{}{
					"set": map[string]interface{}{
						"target":                 "url.value",
						"value":                  "[[.last_response.body.nextLink]]",
						"fail_on_template_error": true,
					},
				},
			},
			"chain": []interface{}{
				map[string]interface{}{
					"step": map[string]interface{}{
						"request.method": http.MethodPost,
						"replace":        "$.files[:].id",
						"replace_with":   "$.exportId,.parent_last_response.body.exportId",
						"request.transforms": []interface{}{
							map[string]interface{}{
								"set": map[string]interface{}{
									"target": "body.exportId",
									"value":  "[[ .parent_last_response.body.exportId ]]",
								},
							},
							map[string]interface{}{
								"set": map[string]interface{}{
									"target": "body.createdAt",
									"value":  "[[ .cursor.last_published_login ]]",
								},
							},
						},
					},
				},
			},
			"cursor": map[string]interface{}{
				"last_published_login": map[string]interface{}{
					"value": "[[ .last_event.createdAt ]]",
				},
			},
		},
		expected: []string{
			`{"hello":{"world":"moon"}}`,
			`{"space":{"cake":"pumpkin"}}`,
			`{"hello":{"cake":"pumpkin"}}`,
			`{"space":{"world":"moon"}}`,
		},
	},
	{
		name: "cursor_value_is_updated_for_root_response_with_chaining_&_pagination_along_with_split_operator",
		setupServer: func(t testing.TB, h http.HandlerFunc, config map[string]interface{}) {
			var serverURL string
			r := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/":
					fmt.Fprintf(w, `{"files":[{"id":"1"},{"id":"2"}],"exportId":"2212","time":[{"timeStamp":"22/02/2022"}],
						"nextLink":"%s/link1"}`, serverURL)
				case "/link1":
					fmt.Fprintln(w, `{"files":[{"id":"3"},{"id":"4"}], "exportId":"2213","time":[{"timeStamp":"24/04/2022"}]}`)
				case "/2212/1":
					matchBody(w, r, `{"createdAt":"22/02/2022","exportId":"2212"}`, `{"hello":{"world":"moon"}}`)
				case "/2212/2":
					matchBody(w, r, `{"createdAt":"22/02/2022","exportId":"2212"}`, `{"space":{"cake":"pumpkin"}}`)
				case "/2213/3":
					matchBody(w, r, `{"createdAt":"24/04/2022","exportId":"2213"}`, `{"hello":{"cake":"pumpkin"}}`)
				case "/2213/4":
					matchBody(w, r, `{"createdAt":"24/04/2022","exportId":"2213"}`, `{"space":{"world":"moon"}}`)
				}
			})
			server := httptest.NewServer(r)
			config["request.url"] = server.URL
			serverURL = server.URL
			config["chain.0.step.request.url"] = server.URL + "/$.exportId/$.files[:].id"
			t.Cleanup(server.Close)
		},
		baseConfig: map[string]interface{}{
			"interval":                            1,
			"request.method":                      http.MethodPost,
			"response.request_body_on_pagination": true,
			"response.pagination": []interface{}{
				map[string]interface{}{
					"set": map[string]interface{}{
						"target":                 "url.value",
						"value":                  "[[.last_response.body.nextLink]]",
						"fail_on_template_error": true,
					},
				},
			},
			"response.split": map[string]interface{}{
				"target":      "body.time",
				"type":        "array",
				"keep_parent": true,
			},
			"chain": []interface{}{
				map[string]interface{}{
					"step": map[string]interface{}{
						"request.method": http.MethodPost,
						"replace":        "$.files[:].id",
						"replace_with":   "$.exportId,.parent_last_response.body.exportId",
						"request.transforms": []interface{}{
							map[string]interface{}{
								"set": map[string]interface{}{
									"target": "body.exportId",
									"value":  "[[ .parent_last_response.body.exportId ]]",
								},
							},
							map[string]interface{}{
								"set": map[string]interface{}{
									"target": "body.createdAt",
									"value":  "[[ .cursor.last_published_login ]]",
								},
							},
						},
					},
				},
			},
			"cursor": map[string]interface{}{
				"last_published_login": map[string]interface{}{
					"value": "[[ .last_event.time.timeStamp ]]",
				},
			},
		},
		expected: []string{
			`{"hello":{"world":"moon"}}`,
			`{"space":{"cake":"pumpkin"}}`,
			`{"hello":{"cake":"pumpkin"}}`,
			`{"space":{"world":"moon"}}`,
		},
	},
	{
		name: "Test simple XML decode",
		setupServer: func(t testing.TB, h http.HandlerFunc, config map[string]interface{}) {
			r := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				const text = `<?xml version="1.0" encoding="UTF-8"?>
<order orderid="56733" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:noNamespaceSchemaLocation="sales.xsd">
  <sender>Ástríðr Ragnar</sender>
  <address>
    <name>Joord Lennart</name>
    <company>Sydøstlige Gruppe</company>
    <address>Beekplantsoen 594, 2 hoog, 6849 IG</address>
    <city>Boekend</city>
    <country>Netherlands</country>
  </address>
  <item>
    <name>Egil's Saga</name>
    <note>Free Sample</note>
    <number>1</number>
    <cost>99.95</cost>
    <sent>FALSE</sent>
  </item>
</order>
`
				io.ReadAll(r.Body)
				r.Body.Close()
				w.Write([]byte(text))
			})
			server := httptest.NewServer(r)
			config["request.url"] = server.URL
			t.Cleanup(server.Close)
		},
		baseConfig: map[string]interface{}{
			"interval":       1,
			"request.method": http.MethodGet,
			"response.xsd": `<?xml version="1.0" encoding="UTF-8" ?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="order">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="sender" type="xs:string"/>
        <xs:element name="address">
          <xs:complexType>
            <xs:sequence>
              <xs:element name="name" type="xs:string"/>
              <xs:element name="company" type="xs:string"/>
              <xs:element name="address" type="xs:string"/>
              <xs:element name="city" type="xs:string"/>
              <xs:element name="country" type="xs:string"/>
            </xs:sequence>
          </xs:complexType>
        </xs:element>
        <xs:element name="item" maxOccurs="unbounded">
          <xs:complexType>
            <xs:sequence>
              <xs:element name="name" type="xs:string"/>
              <xs:element name="note" type="xs:string" minOccurs="0"/>
              <xs:element name="number" type="xs:positiveInteger"/>
              <xs:element name="cost" type="xs:decimal"/>
              <xs:element name="sent" type="xs:boolean"/>
            </xs:sequence>
          </xs:complexType>
        </xs:element>
      </xs:sequence>
      <xs:attribute name="orderid" type="xs:string" use="required"/>
    </xs:complexType>
  </xs:element>
</xs:schema>
`,
		},
		handler: defaultHandler(http.MethodGet, "", ""),
		expected: []string{mapstr.M{
			"order": map[string]interface{}{
				"address": map[string]interface{}{
					"address": "Beekplantsoen 594, 2 hoog, 6849 IG",
					"city":    "Boekend",
					"company": "Sydøstlige Gruppe",
					"country": "Netherlands",
					"name":    "Joord Lennart",
				},
				"item": []interface{}{
					map[string]interface{}{
						"cost":   99.95,
						"name":   "Egil's Saga",
						"note":   "Free Sample",
						"number": 1,
						"sent":   false,
					},
				},
				"noNamespaceSchemaLocation": "sales.xsd",
				"orderid":                   "56733",
				"sender":                    "Ástríðr Ragnar",
				"xsi":                       "http://www.w3.org/2001/XMLSchema-instance",
			},
		}.String()},
	},
}

func TestInput(t *testing.T) {
	logp.TestingSetup()

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			if test.skipReason != "" {
				t.Skipf("skip: %s", test.skipReason)
			}

			test.setupServer(t, test.handler, test.baseConfig)

			cfg := conf.MustNewConfigFrom(test.baseConfig)

			conf := defaultConfig()
			assert.NoError(t, cfg.Unpack(&conf))

			var tempDir string
			if conf.Request.Tracer != nil {
				tempDir = t.TempDir()
				conf.Request.Tracer.Filename = filepath.Join(tempDir, conf.Request.Tracer.Filename)
			}

			input := newStatelessInput(conf)

			assert.Equal(t, "httpjson-stateless", input.Name())
			assert.NoError(t, input.Test(v2.TestContext{}))

			chanClient := beattest.NewChanClient(len(test.expected))
			t.Cleanup(func() { _ = chanClient.Close() })

			ctx, cancel := newV2Context("httpjson-foo-eb837d4c-5ced-45ed-b05c-de658135e248::https://somesource/someapi")
			t.Cleanup(cancel)

			var g errgroup.Group
			g.Go(func() error {
				return input.Run(ctx, chanClient)
			})

			timeout := time.NewTimer(5 * time.Second)
			t.Cleanup(func() { _ = timeout.Stop() })

			if len(test.expected) == 0 {
				select {
				case <-timeout.C:
				case got := <-chanClient.Channel:
					t.Errorf("unexpected event: %v", got)
				}
				cancel()
				assert.NoError(t, g.Wait())
				return
			}

			var receivedCount int
		wait:
			for {
				select {
				case <-timeout.C:
					t.Errorf("timed out waiting for %d events", len(test.expected))
					cancel()
					return
				case got := <-chanClient.Channel:
					val, err := got.Fields.GetValue("message")
					assert.NoError(t, err)
					assert.JSONEq(t, test.expected[receivedCount], val.(string))
					receivedCount += 1
					if receivedCount == len(test.expected) {
						cancel()
						break wait
					}
				}
			}
			if test.expectedFile != "" {
				if _, err := os.Stat(filepath.Join(tempDir, test.expectedFile)); err == nil {
					assert.NoError(t, g.Wait())
				} else {
					t.Errorf("Expected log filename not found")
				}
			}
			assert.NoError(t, g.Wait())
		})
	}
}

func BenchmarkInput(b *testing.B) {
	for _, test := range testCases {
		b.Run(test.name, func(b *testing.B) {
			test.setupServer(b, test.handler, test.baseConfig)

			cfg := conf.MustNewConfigFrom(test.baseConfig)

			conf := defaultConfig()
			assert.NoError(b, cfg.Unpack(&conf))

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				input := newStatelessInput(conf)

				chanClient := beattest.NewChanClient(len(test.expected))
				b.Cleanup(func() { _ = chanClient.Close() })

				ctx, cancel := newV2Context(fmt.Sprintf("%s-%d", test.name, i))
				b.Cleanup(cancel)

				var g errgroup.Group
				g.Go(func() error {
					return input.Run(ctx, chanClient)
				})

				timeout := time.NewTimer(5 * time.Second)
				b.Cleanup(func() { _ = timeout.Stop() })

				if len(test.expected) == 0 {
					select {
					case <-timeout.C:
					case got := <-chanClient.Channel:
						b.Errorf("unexpected event: %v", got)
					}
					cancel()
					assert.NoError(b, g.Wait())
					return
				}

				var receivedCount int
			wait:
				for {
					select {
					case <-timeout.C:
						b.Errorf("timed out waiting for %d events", len(test.expected))
						cancel()
						return
					case <-chanClient.Channel:
						receivedCount += 1
						if receivedCount == len(test.expected) {
							cancel()
							break wait
						}
					}
				}
			}
		})
	}
}

func newTestServer(
	newServer func(http.Handler) *httptest.Server,
) func(testing.TB, http.HandlerFunc, map[string]interface{}) {
	return func(t testing.TB, h http.HandlerFunc, config map[string]interface{}) {
		server := newServer(h)
		config["request.url"] = server.URL
		t.Cleanup(server.Close)
	}
}

func newChainTestServer(
	newServer func(http.Handler) *httptest.Server,
) func(testing.TB, http.HandlerFunc, map[string]interface{}) {
	return func(t testing.TB, h http.HandlerFunc, config map[string]interface{}) {
		r := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/":
				fmt.Fprintln(w, `{"records":[{"id":1}]}`)
			case "/1":
				fmt.Fprintln(w, `{"hello":[{"world":"moon"},{"space":[{"cake":"pumpkin"}]}]}`)
			}
		})
		server := httptest.NewServer(r)
		config["request.url"] = server.URL
		config["chain.0.step.request.url"] = server.URL + "/$.records[:].id"
		t.Cleanup(server.Close)
	}
}

func newNakedChainTestServer(
	newServer func(http.Handler) *httptest.Server,
) func(testing.TB, http.HandlerFunc, map[string]interface{}) {
	return func(t testing.TB, h http.HandlerFunc, config map[string]interface{}) {
		var server *httptest.Server
		r := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/":
				fmt.Fprintln(w, `{"records":["`+server.URL+`/1"]}`)
			case "/1":
				fmt.Fprintln(w, `{"hello":[{"world":"moon"},{"space":[{"cake":"pumpkin"}]}]}`)
			}
		})
		server = httptest.NewServer(r)
		config["request.url"] = server.URL
		t.Cleanup(server.Close)
	}
}

func newChainPaginationTestServer(
	newServer func(http.Handler) *httptest.Server,
) func(testing.TB, http.HandlerFunc, map[string]interface{}) {
	return func(t testing.TB, h http.HandlerFunc, config map[string]interface{}) {
		var serverURL string
		r := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/":
				link := serverURL + "/link2"
				value := fmt.Sprintf(`{"records":[{"id":1}], "nextLink":"%s"}`, link)
				fmt.Fprintln(w, value)
			case "/1":
				fmt.Fprintln(w, `{"hello":{"world":"moon"}}`)
			case "/link2":
				fmt.Fprintln(w, `{"records":[{"id":2}]}`)
			case "/2":
				fmt.Fprintln(w, `{"space":{"cake":"pumpkin"}}`)
			}
		})
		server := httptest.NewServer(r)
		config["request.url"] = server.URL
		serverURL = server.URL
		config["chain.0.step.request.url"] = server.URL + "/$.records[:].id"
	}
}

func newV2Context(id string) (v2.Context, func()) {
	ctx, cancel := context.WithCancel(context.Background())
	return v2.Context{
		Logger:      logp.NewLogger("httpjson_test"),
		ID:          id,
		Cancelation: ctx,
	}, cancel
}

//nolint:errcheck // We can safely ignore errors here
func matchBody(w io.Writer, req *http.Request, match, response string) {
	body, _ := io.ReadAll(req.Body)
	req.Body.Close()
	if string(body) == match {
		w.Write([]byte(response))
	}
}

func defaultHandler(expectedMethod, expectedBody, msg string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		if msg == "" {
			msg = `{"hello":[{"world":"moon"},{"space":[{"cake":"pumpkin"}]}]}`
		}
		switch {
		case r.Method != expectedMethod:
			w.WriteHeader(http.StatusBadRequest)
			msg = fmt.Sprintf(`{"error":"expected method was %q"}`, expectedMethod)
		case expectedBody != "":
			body, _ := io.ReadAll(r.Body)
			r.Body.Close()
			if expectedBody != string(body) {
				w.WriteHeader(http.StatusBadRequest)
				msg = fmt.Sprintf(`{"error":"expected body was %q, but got %q"}`, expectedBody, body)
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
	}
}

func oauth2TokenHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	_ = r.ParseForm()
	switch {
	case r.Method != http.MethodPost:
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
	if r.URL.Path == "/token" {
		oauth2TokenHandler(w, r)
		return
	}

	w.Header().Set("content-type", "application/json")
	switch {
	case r.Method != http.MethodPost:
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
		case 2:
			if r.URL.Query().Get("$filter") != "alertCreationTime ge 2002-10-02T15:00:01Z" {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":"wrong cursor value"`))
				return
			}
			_, _ = w.Write([]byte(`{"@timestamp":"2002-10-02T15:00:02Z","foo":"bar"}`))
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
			_, _ = w.Write([]byte(`{"@timestamp":"2002-10-02T15:00:00Z","nextPageToken":"bar","items":[{"foo":"a"}]}`))
		case 1:
			if r.URL.Query().Get("page") != "bar" {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":"wrong page token value"}`))
				return
			}
			_, _ = w.Write([]byte(`{"@timestamp":"2002-10-02T15:00:01Z","items":[{"foo":"b"}]}`))
		case 2:
			_, _ = w.Write([]byte(`{"@timestamp":"2002-10-02T15:00:02Z","items":[{"foo":"c"}]}`))
		case 3:
			_, _ = w.Write([]byte(`{"@timestamp":"2002-10-02T15:00:03Z","items":[{"foo":"d"}]}`))
			count = 0
			return
		}
		count += 1
	}
}

func paginationArrayHandler() http.HandlerFunc {
	var count int
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		switch count {
		case 0:
			_, _ = w.Write([]byte(`[{"nextPageToken":"bar","foo":"bar"},{"foo":"bar"}]`))
		case 1:
			if r.URL.Query().Get("page") != "bar" {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":"wrong page token value"}`))
				return
			}
			_, _ = w.Write([]byte(`[{"foo":"bar"}]`))
		}
		count += 1
	}
}
