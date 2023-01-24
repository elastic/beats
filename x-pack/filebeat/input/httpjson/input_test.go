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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	beattest "github.com/elastic/beats/v7/libbeat/publisher/testing"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

func TestInput(t *testing.T) {
	testCases := []struct {
		name        string
		setupServer func(*testing.T, http.HandlerFunc, map[string]interface{})
		baseConfig  map[string]interface{}
		handler     http.HandlerFunc
		expected    []string
	}{
		{
			name:        "Test simple GET request",
			setupServer: newTestServer(httptest.NewServer),
			baseConfig: map[string]interface{}{
				"interval":       1,
				"request.method": http.MethodGet,
			},
			handler:  defaultHandler(http.MethodGet, "", ""),
			expected: []string{`{"hello":[{"world":"moon"},{"space":[{"cake":"pumpkin"}]}]}`},
		},
		{
			name:        "Test simple HTTPS GET request",
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
			name:        "Test request honors rate limit",
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
			name:        "Test request retries when failed",
			setupServer: newTestServer(httptest.NewServer),
			baseConfig: map[string]interface{}{
				"interval":       1,
				"request.method": http.MethodGet,
			},
			handler:  retryHandler(),
			expected: []string{`{"hello":"world"}`},
		},
		{
			name:        "Test POST request with body",
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
			name:        "Test repeated POST requests",
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
			name:        "Test split by json objects array",
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
			name:        "Test split by json objects array with keep parent",
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
			name:        "Test split on empty array without ignore_empty_value",
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
			name:        "Test split on empty array with ignore_empty_value",
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
			name:        "Test split on null field with ignore_empty_value keeping parent",
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
			name:        "Test split on empty array with ignore_empty_value keeping parent",
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
			name:        "Test split on null field at root with ignore_empty_value keeping parent",
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
			name:        "Test split on empty array at root with ignore_empty_value keeping parent",
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
			name:        "Test nested split",
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
			name:        "Test split events by not found",
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
			name: "Test date cursor",
			setupServer: func(t *testing.T, h http.HandlerFunc, config map[string]interface{}) {
				registerRequestTransforms()
				t.Cleanup(func() { registeredTransforms = newRegistry() })
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
			name: "Test pagination",
			setupServer: func(t *testing.T, h http.HandlerFunc, config map[string]interface{}) {
				registerPaginationTransforms()
				registerResponseTransforms()
				t.Cleanup(func() { registeredTransforms = newRegistry() })
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
			name: "Test first event",
			setupServer: func(t *testing.T, h http.HandlerFunc, config map[string]interface{}) {
				registerPaginationTransforms()
				registerResponseTransforms()
				t.Cleanup(func() { registeredTransforms = newRegistry() })
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
			name: "Test pagination with array response",
			setupServer: func(t *testing.T, h http.HandlerFunc, config map[string]interface{}) {
				registerPaginationTransforms()
				t.Cleanup(func() { registeredTransforms = newRegistry() })
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
			name: "Test oauth2",
			setupServer: func(t *testing.T, h http.HandlerFunc, config map[string]interface{}) {
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
			name: "Test request transforms can access state from previous transforms",
			setupServer: func(t *testing.T, h http.HandlerFunc, config map[string]interface{}) {
				registerRequestTransforms()
				t.Cleanup(func() { registeredTransforms = newRegistry() })
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
			name: "Test response transforms can't access request state from previous transforms",
			setupServer: func(t *testing.T, h http.HandlerFunc, config map[string]interface{}) {
				registerRequestTransforms()
				registerResponseTransforms()
				t.Cleanup(func() { registeredTransforms = newRegistry() })
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
			name:        "Test simple Chain GET request",
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
			name: "Test multiple Chain GET request",
			setupServer: func(t *testing.T, h http.HandlerFunc, config map[string]interface{}) {
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
			name: "Test date cursor while using chain",
			setupServer: func(t *testing.T, h http.HandlerFunc, config map[string]interface{}) {
				registerRequestTransforms()
				t.Cleanup(func() { registeredTransforms = newRegistry() })
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
			name:        "Test split by json objects array in chain",
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
			name:        "Test split by json objects array with keep parent in chain",
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
			name:        "Test nested split in chain",
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
			name:        "Test pagination when used with chaining",
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
			name: "Test replace_with clause and first_response object",
			setupServer: func(t *testing.T, h http.HandlerFunc, config map[string]interface{}) {
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
				"interval":       1,
				"request.method": http.MethodGet,
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
			name: "Test replace_with clause with hardcoded value_1",
			setupServer: func(t *testing.T, h http.HandlerFunc, config map[string]interface{}) {
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
			name: "Test replace_with clause with hardcoded value (no dot prefix)",
			setupServer: func(t *testing.T, h http.HandlerFunc, config map[string]interface{}) {
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
				"interval":       1,
				"request.method": http.MethodGet,
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
			name: "Test replace_with clause with hardcoded value (more than one dot prefix)",
			setupServer: func(t *testing.T, h http.HandlerFunc, config map[string]interface{}) {
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
				"interval":       1,
				"request.method": http.MethodGet,
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
			name: "Test replace_with clause with hardcoded value containing '.' (dots)",
			setupServer: func(t *testing.T, h http.HandlerFunc, config map[string]interface{}) {
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
			name: "Test global transform context separation with parent_last_response object",
			setupServer: func(t *testing.T, h http.HandlerFunc, config map[string]interface{}) {
				var serverURL string
				registerPaginationTransforms()
				registerRequestTransforms()
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
				t.Cleanup(func() { registeredTransforms = newRegistry() })
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
			name: "Test if cursor value is updated for root response with chaining & pagination",
			setupServer: func(t *testing.T, h http.HandlerFunc, config map[string]interface{}) {
				var serverURL string
				registerPaginationTransforms()
				registerRequestTransforms()
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
				t.Cleanup(func() { registeredTransforms = newRegistry() })
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
			name: "Test if cursor value is updated for root response with chaining & pagination along with split operator",
			setupServer: func(t *testing.T, h http.HandlerFunc, config map[string]interface{}) {
				var serverURL string
				registerPaginationTransforms()
				registerRequestTransforms()
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
				t.Cleanup(func() { registeredTransforms = newRegistry() })
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
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			tc.setupServer(t, tc.handler, tc.baseConfig)

			cfg := conf.MustNewConfigFrom(tc.baseConfig)

			conf := defaultConfig()
			assert.NoError(t, cfg.Unpack(&conf))

			input := newStatelessInput(conf)

			assert.Equal(t, "httpjson-stateless", input.Name())
			assert.NoError(t, input.Test(v2.TestContext{}))

			chanClient := beattest.NewChanClient(len(tc.expected))
			t.Cleanup(func() { _ = chanClient.Close() })

			ctx, cancel := newV2Context()
			t.Cleanup(cancel)

			var g errgroup.Group
			g.Go(func() error {
				return input.Run(ctx, chanClient)
			})

			timeout := time.NewTimer(5 * time.Second)
			t.Cleanup(func() { _ = timeout.Stop() })

			if len(tc.expected) == 0 {
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
					t.Errorf("timed out waiting for %d events", len(tc.expected))
					cancel()
					return
				case got := <-chanClient.Channel:
					val, err := got.Fields.GetValue("message")
					assert.NoError(t, err)
					assert.JSONEq(t, tc.expected[receivedCount], val.(string))
					receivedCount += 1
					if receivedCount == len(tc.expected) {
						cancel()
						break wait
					}
				}
			}
			assert.NoError(t, g.Wait())
		})
	}
}

func newTestServer(
	newServer func(http.Handler) *httptest.Server,
) func(*testing.T, http.HandlerFunc, map[string]interface{}) {
	return func(t *testing.T, h http.HandlerFunc, config map[string]interface{}) {
		server := newServer(h)
		config["request.url"] = server.URL
		t.Cleanup(server.Close)
	}
}

func newChainTestServer(
	newServer func(http.Handler) *httptest.Server,
) func(*testing.T, http.HandlerFunc, map[string]interface{}) {
	return func(t *testing.T, h http.HandlerFunc, config map[string]interface{}) {
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

func newChainPaginationTestServer(
	newServer func(http.Handler) *httptest.Server,
) func(*testing.T, http.HandlerFunc, map[string]interface{}) {
	return func(t *testing.T, h http.HandlerFunc, config map[string]interface{}) {
		registerPaginationTransforms()
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
		t.Cleanup(func() { registeredTransforms = newRegistry() })
	}
}

func newV2Context() (v2.Context, func()) {
	ctx, cancel := context.WithCancel(context.Background())
	return v2.Context{
		Logger:      logp.NewLogger("httpjson_test"),
		ID:          "test_id",
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
