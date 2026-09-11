// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package elasticsearchclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"

	"github.com/elastic/elastic-agent-libs/logp"
)

// TestExtension_ConcurrentRequests_Race issues concurrent Request calls through
// one extension and checks that path and body stay paired. eslegclient.Connection
// reuses a single encoder buffer and response buffer; without the
// extension-owned mutex those buffers race and either:
//   - mix one call's JSON onto another call's path, or
//   - return a cloned-too-late response body that belongs to a later call.
//
// Run with `go test -race` so the detector can flag unsynchronized buffer use.
func TestExtension_ConcurrentRequests_Race(t *testing.T) {
	srv := newRaceFakeES()
	t.Cleanup(srv.Close)

	ext := newTestExtension(map[string]any{
		"hosts":    []string{srv.URL},
		"username": "elastic",
		"password": "changeme",
	}, logp.NewNopLogger())
	require.NoError(t, ext.Start(t.Context(), componenttest.NewNopHost()), "Start must succeed before concurrent requests")
	t.Cleanup(func() { _ = ext.Shutdown(context.Background()) })

	const (
		workers = 8
		ops     = 40
	)

	var (
		wg     sync.WaitGroup
		failed atomic.Int64
	)
	report := func(msg string, args ...any) {
		if failed.Add(1) == 1 {
			t.Errorf(msg, args...)
		}
	}

	for w := range workers {
		wg.Go(func() {
			for n := range ops {
				token := fmt.Sprintf("w%02d-n%04d", w, n)
				path := "/" + token + "/_search"
				body := map[string]any{"token": token, "query": map[string]any{"term": map[string]any{"id": token}}}

				status, resp, err := ext.Request(http.MethodPost, path, "", map[string]string{"routing": token}, body)
				if err != nil {
					report("concurrent Request %s failed: %v", token, err)
					return
				}
				if status != http.StatusOK {
					report("concurrent Request %s status=%d, want 200", token, status)
					return
				}
				var got struct {
					Token string `json:"token"`
				}
				if err := json.Unmarshal(resp, &got); err != nil {
					report("concurrent Request %s returned non-JSON %q: %v", token, resp, err)
					return
				}
				if got.Token != token {
					report("response/body integrity failed: sent token %s, got %s (body %q)", token, got.Token, resp)
					return
				}
			}
		})
	}

	wg.Wait()
	assert.EqualValues(t, 0, failed.Load(), "concurrent Request calls must succeed with matching tokens")
	assert.Empty(t, srv.errs(), "fake ES must not observe path/body mismatches")
}

type raceFakeES struct {
	*httptest.Server
	mu      sync.Mutex
	corrupt []string
}

func newRaceFakeES() *raceFakeES {
	f := &raceFakeES{}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/" {
			pingOK(w)
			return
		}

		body, _ := io.ReadAll(r.Body)
		token := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/"), "/_search")
		if token == "" || strings.Contains(token, "/") {
			f.record(fmt.Sprintf("unexpected path %s", r.URL.Path))
			http.Error(w, `{"error":"bad path"}`, http.StatusBadRequest)
			return
		}
		if r.URL.Query().Get("routing") != token {
			f.record(fmt.Sprintf("path %s routing param %q did not match token", r.URL.Path, r.URL.Query().Get("routing")))
			http.Error(w, `{"error":"bad routing"}`, http.StatusBadRequest)
			return
		}
		if !bytes.Contains(body, []byte(`"token":"`+token+`"`)) {
			f.record(fmt.Sprintf("path %s got body %q", r.URL.Path, body))
			http.Error(w, `{"error":"body/path mismatch"}`, http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"ok":true,"token":%q}`, token)
	}))
	return f
}

func (f *raceFakeES) record(msg string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.corrupt = append(f.corrupt, msg)
}

func (f *raceFakeES) errs() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.corrupt))
	copy(out, f.corrupt)
	return out
}
