// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package scenarios

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/heartbeat/hbtest"
)

var testWsOnce = &sync.Once{}

// Starting this thing up is expensive, let's just do it once
func startTestWebserver(t *testing.T) *httptest.Server {
	testWsOnce.Do(func() {
		testWs = httptest.NewServer(hbtest.HelloWorldHandler(200))

		// wait for ws to become available
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

func startStatefulTestWS(t *testing.T, statuses []int) *httptest.Server {
	mtx := sync.Mutex{}
	statusIdx := 0
	testWs = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mtx.Lock()
		defer mtx.Unlock()

		statusIdx++
		if statusIdx > len(statuses)-1 {
			statusIdx = 0
		}

		status := statuses[statusIdx]
		w.WriteHeader(status)
		_, _ = w.Write([]byte(fmt.Sprintf("Status: %d", status)))
	}))

	// wait for ws to become available
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

	return testWs
}
