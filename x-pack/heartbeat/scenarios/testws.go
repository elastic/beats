package scenarios

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/elastic/beats/v7/heartbeat/hbtest"
	"github.com/stretchr/testify/require"
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
		w.Write([]byte(fmt.Sprintf("Status: %d", status)))

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
