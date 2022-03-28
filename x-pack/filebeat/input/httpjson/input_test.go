// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"context"
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	beattest "github.com/elastic/beats/v7/libbeat/publisher/testing"
)

const barConst = "bar"

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
				"request.method": "GET",
			},
			handler:  defaultHandler("GET", ""),
			expected: []string{`{"hello":[{"world":"moon"},{"space":[{"cake":"pumpkin"}]}]}`},
		},
		{
			name:        "Test simple HTTPS GET request",
			setupServer: newTestServer(httptest.NewTLSServer),
			baseConfig: map[string]interface{}{
				"interval":                      1,
				"request.method":                "GET",
				"request.ssl.verification_mode": "none",
			},
			handler:  defaultHandler("GET", ""),
			expected: []string{`{"hello":[{"world":"moon"},{"space":[{"cake":"pumpkin"}]}]}`},
		},
		{
			name:        "Test request honors rate limit",
			setupServer: newTestServer(httptest.NewServer),
			baseConfig: map[string]interface{}{
				"interval":                     1,
				"http_method":                  "GET",
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
				"request.method": "GET",
			},
			handler:  retryHandler(),
			expected: []string{`{"hello":"world"}`},
		},
		{
			name:        "Test POST request with body",
			setupServer: newTestServer(httptest.NewServer),
			baseConfig: map[string]interface{}{
				"interval":       1,
				"request.method": "POST",
				"request.body": map[string]interface{}{
					"test": "abc",
				},
			},
			handler:  defaultHandler("POST", `{"test":"abc"}`),
			expected: []string{`{"hello":[{"world":"moon"},{"space":[{"cake":"pumpkin"}]}]}`},
		},
		{
			name:        "Test repeated POST requests",
			setupServer: newTestServer(httptest.NewServer),
			baseConfig: map[string]interface{}{
				"interval":       "100ms",
				"request.method": "POST",
			},
			handler: defaultHandler("POST", ""),
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
				"request.method": "GET",
				"response.split": map[string]interface{}{
					"target": "body.hello",
				},
			},
			handler:  defaultHandler("GET", ""),
			expected: []string{`{"world":"moon"}`, `{"space":[{"cake":"pumpkin"}]}`},
		},
		{
			name:        "Test split by json objects array with keep parent",
			setupServer: newTestServer(httptest.NewServer),
			baseConfig: map[string]interface{}{
				"interval":       1,
				"request.method": "GET",
				"response.split": map[string]interface{}{
					"target":      "body.hello",
					"keep_parent": true,
				},
			},
			handler: defaultHandler("GET", ""),
			expected: []string{
				`{"hello":{"world":"moon"}}`,
				`{"hello":{"space":[{"cake":"pumpkin"}]}}`,
			},
		},
		{
			name:        "Test nested split",
			setupServer: newTestServer(httptest.NewServer),
			baseConfig: map[string]interface{}{
				"interval":       1,
				"request.method": "GET",
				"response.split": map[string]interface{}{
					"target": "body.hello",
					"split": map[string]interface{}{
						"target":      "body.space",
						"keep_parent": true,
					},
				},
			},
			handler: defaultHandler("GET", ""),
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
				"request.method": "GET",
				"response.split": map[string]interface{}{
					"target": "body.unknown",
				},
			},
			handler:  defaultHandler("GET", ""),
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
				"request.method": "GET",
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
				t.Cleanup(func() { registeredTransforms = newRegistry() })
				server := httptest.NewServer(h)
				config["request.url"] = server.URL
				t.Cleanup(server.Close)
			},
			baseConfig: map[string]interface{}{
				"interval":       time.Second,
				"request.method": "GET",
				"response.split": map[string]interface{}{
					"target": "body.items",
				},
				"response.pagination": []interface{}{
					map[string]interface{}{
						"set": map[string]interface{}{
							"target": "url.params.page",
							"value":  "[[.last_response.body.nextPageToken]]",
						},
					},
				},
			},
			handler:  paginationHandler(),
			expected: []string{`{"foo":"a"}`, `{"foo":"b"}`},
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
				"request.method": "GET",
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
				"request.method": "GET",
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
				"request.method":            "POST",
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
				"request.method": "POST",
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
			handler:  defaultHandler("POST", `{"bar":"foo","url":{"path":"/test-path"}}`),
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
				"request.method": "GET",
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
			handler:  defaultHandler("GET", ""),
			expected: []string{`{"hello":[{"world":"moon"},{"space":[{"cake":"pumpkin"}]}]}`},
		},
		{
			name:        "Test simple Chain GET request",
			setupServer: newChainTestServer(httptest.NewServer),
			baseConfig: map[string]interface{}{
				"interval":       10,
				"request.method": "GET",
				"chain": []interface{}{
					map[string]interface{}{
						"step": map[string]interface{}{
							"request.method": "GET",
							"replace":        "$.records[:].id",
						},
					},
				},
			},
			handler:  defaultHandler("GET", ""),
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
				"request.method": "GET",
				"chain": []interface{}{
					map[string]interface{}{
						"step": map[string]interface{}{
							"request.method": "GET",
							"replace":        "$.records[:].id",
						},
					},
					map[string]interface{}{
						"step": map[string]interface{}{
							"request.method": "GET",
							"replace":        "$.file_name",
						},
					},
				},
			},
			handler:  defaultHandler("GET", ""),
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
				"request.method": "GET",
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
							"request.method": "GET",
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
				"request.method": "GET",
				"chain": []interface{}{
					map[string]interface{}{
						"step": map[string]interface{}{
							"request.method": "GET",
							"replace":        "$.records[:].id",
							"response.split": map[string]interface{}{
								"target": "body.hello",
							},
						},
					},
				},
			},
			handler:  defaultHandler("GET", ""),
			expected: []string{`{"world":"moon"}`, `{"space":[{"cake":"pumpkin"}]}`},
		},
		{
			name:        "Test split by json objects array with keep parent in chain",
			setupServer: newChainTestServer(httptest.NewServer),
			baseConfig: map[string]interface{}{
				"interval":       1,
				"request.method": "GET",
				"chain": []interface{}{
					map[string]interface{}{
						"step": map[string]interface{}{
							"request.method": "GET",
							"replace":        "$.records[:].id",
							"response.split": map[string]interface{}{
								"target":      "body.hello",
								"keep_parent": true,
							},
						},
					},
				},
			},
			handler: defaultHandler("GET", ""),
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
				"request.method": "GET",
				"response.split": map[string]interface{}{
					"target": "body.hello",
				},
				"chain": []interface{}{
					map[string]interface{}{
						"step": map[string]interface{}{
							"request.method": "GET",
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
			handler: defaultHandler("GET", ""),
			expected: []string{
				`{"world":"moon"}`,
				`{"space":{"cake":"pumpkin"}}`,
			},
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			tc.setupServer(t, tc.handler, tc.baseConfig)

			cfg := common.MustNewConfigFrom(tc.baseConfig)

			conf := defaultConfig()
			assert.NoError(t, cfg.Unpack(&conf))

			input, err := newStatelessInput(conf)

			assert.NoError(t, err)
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

func newV2Context() (v2.Context, func()) {
	ctx, cancel := context.WithCancel(context.Background())
	return v2.Context{
		Logger:      logp.NewLogger("httpjson_test"),
		ID:          "test_id",
		Cancelation: ctx,
	}, cancel
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
		n, err := rand.Int(rand.Reader, big.NewInt(100))
		if err != nil {
			return
		}
		w.WriteHeader(int(n.Int64()) + 500)
		count += 1
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
	if r.URL.Path == "/token" {
		oauth2TokenHandler(w, r)
		return
	}

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
			if r.URL.Query().Get("page") != barConst {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":"wrong page token value"}`))
				return
			}
			_, _ = w.Write([]byte(`{"@timestamp":"2002-10-02T15:00:01Z","items":[{"foo":"b"}]}`))
		case 2:
			_, _ = w.Write([]byte(`{"@timestamp":"2002-10-02T15:00:02Z","items":[{"foo":"c"}]}`))
		case 3:
			_, _ = w.Write([]byte(`{"@timestamp":"2002-10-02T15:00:03Z","items":[{"foo":"d"}]}`))
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
			if r.URL.Query().Get("page") != barConst {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":"wrong page token value"}`))
				return
			}
			_, _ = w.Write([]byte(`[{"foo":"bar"}]`))
		}
		count += 1
	}
}
