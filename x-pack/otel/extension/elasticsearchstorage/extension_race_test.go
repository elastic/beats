// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package elasticsearchstorage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"

	"github.com/elastic/beats/v7/libbeat/statestore/backend"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

// TestElasticStorage_Access_Concurrent_Race documents and guards against the
// data race in the elasticsearch_storage OTel extension: a single
// eslegclient.Connection (with its single reused *bytes.Buffer body encoder)
// is shared across every backend.Store returned by Access. Concurrent state
// ops from different streams therefore race on the shared buffer, producing
// either:
//
//   - "http: ContentLength=N with Body length M" (Go stdlib aborts the
//     request locally — buffer was overwritten between
//     http.NewRequestWithContext capturing ContentLength and the transport
//     reading the live body), or
//   - the wrong body landing on the wire, e.g. a {"v":...} state doc reaching
//     the _search endpoint, which ES rejects with
//     "Unknown key for a START_OBJECT in [v]".
//
// This test stands up a fake ES (httptest.Server) and drives concurrent Set
// and Each from multiple goroutines against multiple stores produced by one
// extension. It is expected to FAIL today, and should pass once Access
// returns stores that serialize access to the underlying connection.
//
// Under `go test -race` the data race is detected directly on the encoder's
// shared *bytes.Buffer. Without -race the test still catches the bug
// functionally — the fake ES rejects bodies that don't match the path
// (PUT must carry a {"v":...} state doc, _search must carry a match_all
// query), and the Go transport's own ContentLength/body-length mismatch
// surfaces as an error returned from Set/Each.
func TestElasticStorage_Access_Concurrent_Race(t *testing.T) {
	srv := newFakeES(t)
	defer srv.Close()

	cfg := &Config{
		ElasticsearchConfig: map[string]interface{}{
			"hosts":    []string{srv.URL},
			"username": "elastic",
			"password": "changeme",
		},
	}
	ext := &elasticStorage{cfg: cfg, logger: logptest.NewTestingLogger(t, "")}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	require.NoError(t, ext.Start(ctx, componenttest.NewNopHost()))
	t.Cleanup(func() { _ = ext.Shutdown(context.Background()) })

	const (
		numStores       = 4
		opsPerStore     = 50
		writersPerStore = 2 // one Each loop + one Set loop per store == higher race odds
	)

	stores := make([]backend.Store, numStores)
	for i := 0; i < numStores; i++ {
		s, err := ext.Access(fmt.Sprintf("stream-%d", i))
		require.NoError(t, err)
		stores[i] = s
	}

	var (
		wg     sync.WaitGroup
		failed atomic.Int64
	)

	report := func(op string, err error) {
		// Any per-op error is sufficient to fail the test; capture the
		// first one in detail and silence the rest to keep output sane.
		if failed.Add(1) == 1 {
			t.Errorf("concurrent %s failed: %v", op, err)
		}
	}

	for i := 0; i < numStores; i++ {
		s := stores[i]
		key := fmt.Sprintf("cursor-%d", i)
		for w := 0; w < writersPerStore; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < opsPerStore; j++ {
					if err := s.Set(key, map[string]any{
						"cursor":  nil,
						"ttl":     0,
						"updated": time.Now().UTC().Format(time.RFC3339Nano),
					}); err != nil {
						report("Set", err)
						return
					}
				}
			}()
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < opsPerStore; j++ {
					if err := s.Each(func(string, backend.ValueDecoder) (bool, error) {
						return true, nil
					}); err != nil {
						report("Each", err)
						return
					}
				}
			}()
		}
	}

	wg.Wait()

	if failed.Load() != 0 {
		t.Fatalf("%d concurrent ops returned errors (see first reported above) — likely the shared-encoder race in elasticStorage.Access", failed.Load())
	}
	if srvErrs := srv.errs(); len(srvErrs) > 0 {
		t.Fatalf("fake ES detected corrupted request bodies (signature A/B): %v", srvErrs)
	}
}

// fakeES is a minimal stand-in for Elasticsearch that:
//   - answers Ping with a valid version response,
//   - validates that PUT _doc bodies look like state docs ({"v":...}),
//   - validates that _search bodies look like search queries (match_all),
//   - records any body that arrives on the wrong endpoint (Signature A of
//     the shared-buffer race) for the test to assert on.
type fakeES struct {
	*httptest.Server

	mu      sync.Mutex
	corrupt []string
}

func newFakeES(t *testing.T) *fakeES {
	t.Helper()
	f := &fakeES{}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/":
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"version":{"number":"8.10.0","build_flavor":"default"},"name":"fake"}`)

		case strings.HasSuffix(r.URL.Path, "/_search"):
			body, _ := io.ReadAll(r.Body)
			if !bytes.Contains(body, []byte(`match_all`)) {
				f.record(fmt.Sprintf("search got non-search body on %s: %q", r.URL.Path, body))
				http.Error(w, `{"error":{"type":"parsing_exception","reason":"Unknown key for a START_OBJECT in [v]"}}`, http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"took":1,"hits":{"total":{"value":0,"relation":"eq"},"hits":[]}}`)

		case r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/_doc/"):
			body, _ := io.ReadAll(r.Body)
			// A correctly-marshalled state doc starts with {"v":
			trimmed := bytes.TrimSpace(body)
			if !bytes.HasPrefix(trimmed, []byte(`{"v":`)) {
				f.record(fmt.Sprintf("PUT got non-state-doc body on %s: %q", r.URL.Path, body))
				http.Error(w, `{"error":"bad body"}`, http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"result":"created"}`)

		default:
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{}`)
		}
	}))
	return f
}

func (f *fakeES) record(msg string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.corrupt = append(f.corrupt, msg)
}

func (f *fakeES) errs() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.corrupt))
	copy(out, f.corrupt)
	return out
}
