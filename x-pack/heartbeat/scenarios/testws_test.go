// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package scenarios

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v9/heartbeat/hbtest"
)

var testWsOnce = &sync.Once{}
var failingTestWsOnce = &sync.Once{}

// Starting this thing up is expensive, let's just do it once
func startTestWebserver(t *testing.T) *httptest.Server {
	testWsOnce.Do(func() {
		testWs = httptest.NewServer(hbtest.HelloWorldHandler(200))

		waitForWs(t, testWs.URL, 200)
	})

	return testWs
}

func startFailingTestWebserver(t *testing.T) *httptest.Server {
	failingTestWsOnce.Do(func() {
		failingTestWs = httptest.NewServer(hbtest.HelloWorldHandler(400))

		waitForWs(t, failingTestWs.URL, 400)
	})

	return failingTestWs
}

func waitForWs(t *testing.T, url string, statusCode int) {
	require.Eventuallyf(
		t,
		func() bool {
			req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
			resp, _ := http.DefaultClient.Do(req)
			resp.Body.Close()
			return resp.StatusCode == statusCode
		},
		10*time.Second, 250*time.Millisecond, "could not start webserver",
	)
}
