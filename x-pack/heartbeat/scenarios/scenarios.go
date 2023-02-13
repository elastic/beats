// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package scenarios

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/elastic/beats/v7/heartbeat/hbtest"
	"github.com/elastic/beats/v7/x-pack/heartbeat/scenarios/framework"
)

var scenarioDB = framework.NewScenarioDB()
var testWs *httptest.Server

var testWsOnce = &sync.Once{}

// Starting this thing up is expensive, let's just do it once
func startTestWebserver(t *testing.T) *httptest.Server {
	testWsOnce.Do(func() {
		testWs = httptest.NewServer(hbtest.HelloWorldHandler(200))
		var err error
		for i := 0; i < 20; i++ {
			var resp *http.Response
			req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, testWs.URL, nil)
			resp, err = http.DefaultClient.Do(req)
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode == 200 {
					break
				}
			}

			time.Sleep(time.Millisecond * 250)
		}

		if err != nil {
			require.NoError(t, err, "could not retrieve successful response from test webserver")
		}
	})

	return testWs
}

// Note, no browser scenarios here, those all go in browserscenarios.go
// since they have different build tags
func init() {
	scenarioDB.Add(
		framework.Scenario{
			Name: "http-simple",
			Type: "http",
			Tags: []string{"lightweight", "http"},
			Runner: func(t *testing.T) (config mapstr.M, close func(), err error) {
				server := startTestWebserver(t)
				config = mapstr.M{
					"id":       "http-test-id",
					"name":     "http-test-name",
					"type":     "http",
					"schedule": "@every 1m",
					"urls":     []string{server.URL},
				}
				return config, nil, nil
			},
		},
		framework.Scenario{
			Name: "tcp-simple",
			Type: "tcp",
			Tags: []string{"lightweight", "tcp"},
			Runner: func(t *testing.T) (config mapstr.M, close func(), err error) {
				server := startTestWebserver(t)
				parsedUrl, err := url.Parse(server.URL)
				if err != nil {
					panic(fmt.Sprintf("URL %s should always be parsable: %s", server.URL, err))
				}
				config = mapstr.M{
					"id":       "tcp-test-id",
					"name":     "tcp-test-name",
					"type":     "tcp",
					"schedule": "@every 1m",
					"hosts":    []string{parsedUrl.Host}, // Host includes host:port
				}
				return config, nil, nil
			},
		},
		framework.Scenario{
			Name: "simple-icmp",
			Type: "icmp",
			Tags: []string{"icmp"},
			Runner: func(t *testing.T) (config mapstr.M, close func(), err error) {
				return mapstr.M{
					"id":       "icmp-test-id",
					"name":     "icmp-test-name",
					"type":     "icmp",
					"schedule": "@every 1m",
					"hosts":    []string{"127.0.0.1"},
				}, func() {}, nil
			},
		},
	)
}
