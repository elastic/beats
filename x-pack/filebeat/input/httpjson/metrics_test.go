// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"

	"github.com/elastic/beats/v7/libbeat/monitoring/inputmon"
	beattest "github.com/elastic/beats/v7/libbeat/publisher/testing"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

func TestMetrics(t *testing.T) {
	testCases := []struct {
		name           string
		setupServer    func(*testing.T, http.HandlerFunc, map[string]interface{})
		baseConfig     map[string]interface{}
		handler        http.HandlerFunc
		expectedEvents []string
		assertMetrics  func(reg *monitoring.Registry) error
	}{
		{
			name: "Test pagination metrics",
			setupServer: func(t *testing.T, h http.HandlerFunc, config map[string]interface{}) {
				server := httptest.NewServer(h)
				config["request.url"] = server.URL
				t.Cleanup(server.Close)
			},
			baseConfig: map[string]interface{}{
				"interval":       time.Millisecond,
				"request.method": http.MethodPost,
				"request.body": map[string]interface{}{
					"field": "value",
				},
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
			expectedEvents: []string{
				`{"foo":"a","page":"0"}`, `{"foo":"b","page":"1"}`, `{"foo":"c","page":"0"}`, `{"foo":"d","page":"0"}`,
				`{"foo":"a","page":"0"}`, `{"foo":"b","page":"1"}`, `{"foo":"c","page":"0"}`, `{"foo":"d","page":"0"}`,
			},
			assertMetrics: func(reg *monitoring.Registry) error {
				checkHasValue := func(v interface{}) bool {
					switch t := v.(type) {
					case int64:
						return t > 0
					case map[string]interface{}:
						h := t["histogram"].(map[string]interface{})
						return h["count"].(int64) > 0 && h["max"].(int64) > 0
					}
					return false
				}

				snapshot := monitoring.CollectStructSnapshot(reg, monitoring.Full, true)

				for _, m := range []string{
					"http_request_body_bytes", "http_request_post_total",
					"http_request_total", "http_response_2xx_total",
					"http_response_body_bytes", "http_response_total",
					"http_round_trip_time", "httpjson_interval_execution_time",
					"httpjson_interval_pages_execution_time", "httpjson_interval_total",
				} {
					if !checkHasValue(snapshot[m]) {
						return fmt.Errorf("expected non zero value for metric %s", m)
					}
				}

				return nil
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

			chanClient := beattest.NewChanClient(len(tc.expectedEvents))
			t.Cleanup(func() { _ = chanClient.Close() })

			ctx, cancel := newV2Context("httpjson-foo-eb837d4c-5ced-45ed-b05c-de658135e248::https://somesource/someapi")
			t.Cleanup(cancel)

			reg, unreg := inputmon.NewInputRegistry("httpjson-test", ctx.ID, nil)
			t.Cleanup(unreg)

			var g errgroup.Group
			g.Go(func() error {
				pub := statelessPublisher{wrapped: chanClient}
				return run(ctx, conf, pub, nil, reg)
			})

			timeout := time.NewTimer(5 * time.Second)
			t.Cleanup(func() { _ = timeout.Stop() })

			if len(tc.expectedEvents) == 0 {
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
					t.Errorf("timed out waiting for %d events", len(tc.expectedEvents))
					cancel()
					return
				case got := <-chanClient.Channel:
					val, err := got.Fields.GetValue("message")
					assert.NoError(t, err)
					assert.JSONEq(t, tc.expectedEvents[receivedCount], val.(string))
					receivedCount += 1
					if receivedCount == len(tc.expectedEvents) {
						cancel()
						break wait
					}
				}
			}

			assert.NoError(t, g.Wait())
			assert.NoError(t, tc.assertMetrics(reg))
		})
	}
}
