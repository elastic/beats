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
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/extension/xextension/storage"

	"github.com/elastic/entcollect"

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
// extension.
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
		ElasticsearchConfig: map[string]any{
			"hosts":    []string{srv.URL},
			"username": "elastic",
			"password": "changeme",
		},
	}
	ext := &elasticStorage{cfg: cfg, logger: logptest.NewTestingLogger(t, "")}

	ctx := t.Context()

	require.NoError(t, ext.Start(ctx, componenttest.NewNopHost()))
	t.Cleanup(func() { _ = ext.Shutdown(context.Background()) })

	const (
		numStores       = 4
		opsPerStore     = 50
		writersPerStore = 2 // one Each loop + one Set loop per store == higher race odds
	)

	stores := make([]backend.Store, numStores)
	for i := range numStores {
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

	for i := range numStores {
		s := stores[i]
		key := fmt.Sprintf("cursor-%d", i)
		for range writersPerStore {
			wg.Go(func() {
				for range opsPerStore {
					if err := s.Set(key, map[string]any{
						"cursor":  nil,
						"ttl":     0,
						"updated": time.Now().UTC().Format(time.RFC3339Nano),
					}); err != nil {
						report("Set", err)
						return
					}
				}
			})
			wg.Go(func() {
				for range opsPerStore {
					if err := s.Each(func(string, backend.ValueDecoder) (bool, error) {
						return true, nil
					}); err != nil {
						report("Each", err)
						return
					}
				}
			})
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

// TestElasticStorage_MixedRegistry_Concurrent_Race covers the cross-path
// race between the two registry interfaces the extension implements:
// backend.Registry (used by Filebeat inputs via Access → *lockedStore) and
// entcollect.Registry (used by entcollect providers via Store →
// *entcollectStore). Both factories return stores backed by the same
// e.client, but only *lockedStore takes the extension's clientMu. When
// the two paths are exercised on the same extension instance, the mutex
// on the Access side does not protect against unsynchronized ops from the
// Store side — both still hit the shared encoder buffer, response
// buffer, and JSON visitor state on eslegclient.Connection.
//
// This is the realistic deployment shape: an EDOT/hybrid agent runs
// Filebeat httpjson streams (Access path) alongside entcollect identity
// providers (Store path), pointing at the same elasticsearch_storage
// extension.
func TestElasticStorage_MixedRegistry_Concurrent_Race(t *testing.T) {
	srv := newFakeES(t)
	defer srv.Close()

	cfg := &Config{
		ElasticsearchConfig: map[string]any{
			"hosts":    []string{srv.URL},
			"username": "elastic",
			"password": "changeme",
		},
	}
	ext := &elasticStorage{cfg: cfg, logger: logptest.NewTestingLogger(t, "")}

	ctx := t.Context()

	require.NoError(t, ext.Start(ctx, componenttest.NewNopHost()))
	t.Cleanup(func() { _ = ext.Shutdown(context.Background()) })

	// Compile-time check that the extension still satisfies the two
	// registry interfaces this test exercises.
	var (
		_ backend.Registry    = ext
		_ entcollect.Registry = ext
	)

	const (
		numStores   = 4
		opsPerStore = 50
	)

	beatStores := make([]backend.Store, numStores)
	entStores := make([]entcollect.Store, numStores)
	otelClients := make([]storage.Client, numStores)
	for i := range numStores {
		bs, err := ext.Access(fmt.Sprintf("stream-%d", i))
		require.NoError(t, err)
		beatStores[i] = bs

		es, err := ext.Store(fmt.Sprintf("provider-%d", i))
		require.NoError(t, err)
		entStores[i] = es

		// OTel storage.Client path: the third face reaching the shared
		// connection. It must serialize on the same clientMu as the other two.
		oc, err := ext.GetClient(ctx, component.KindReceiver, component.MustNewIDWithName("otelrcv", fmt.Sprintf("r%d", i)), "")
		require.NoError(t, err)
		otelClients[i] = oc
	}
	t.Cleanup(func() {
		for _, oc := range otelClients {
			_ = oc.Close(context.Background())
		}
	})

	var (
		wg     sync.WaitGroup
		failed atomic.Int64
	)
	report := func(op string, err error) {
		if failed.Add(1) == 1 {
			t.Errorf("concurrent %s failed: %v", op, err)
		}
	}

	for i := range numStores {
		bs := beatStores[i]
		es := entStores[i]
		key := fmt.Sprintf("cursor-%d", i)

		// Access path: Set + Each, same shape as the original test.
		wg.Go(func() {
			for range opsPerStore {
				if err := bs.Set(key, map[string]any{
					"cursor":  nil,
					"ttl":     0,
					"updated": time.Now().UTC().Format(time.RFC3339Nano),
				}); err != nil {
					report("Access.Set", err)
					return
				}
			}
		})
		wg.Go(func() {
			for range opsPerStore {
				if err := bs.Each(func(string, backend.ValueDecoder) (bool, error) {
					return true, nil
				}); err != nil {
					report("Access.Each", err)
					return
				}
			}
		})

		// Store path: same op pattern via entcollect.Store. These
		// goroutines do not hold clientMu in the current code, so
		// they race against the Access goroutines on the shared
		// eslegclient.Connection.
		wg.Go(func() {
			for range opsPerStore {
				if err := es.Set(key, map[string]any{
					"cursor":  nil,
					"ttl":     0,
					"updated": time.Now().UTC().Format(time.RFC3339Nano),
				}); err != nil {
					report("Store.Set", err)
					return
				}
			}
		})
		wg.Go(func() {
			for range opsPerStore {
				if err := es.Each(func(string, func(any) error) (bool, error) {
					return true, nil
				}); err != nil {
					report("Store.Each", err)
					return
				}
			}
		})

		// OTel path: Set + Get + Walk via the storage.Client, exercising the
		// same shared connection as the two Beats paths above.
		oc := otelClients[i]
		wg.Go(func() {
			val := []byte(fmt.Sprintf(`{"cursor":null,"n":%d}`, i))
			for range opsPerStore {
				if err := oc.Set(ctx, key, val); err != nil {
					report("OTel.Set", err)
					return
				}
			}
		})
		wg.Go(func() {
			for range opsPerStore {
				if _, err := oc.Get(ctx, key); err != nil {
					report("OTel.Get", err)
					return
				}
			}
		})
		wg.Go(func() {
			for range opsPerStore {
				if err := oc.(*esStorageClient).Walk(ctx, func(string, []byte) ([]*storage.Operation, error) {
					return nil, nil
				}); err != nil {
					report("OTel.Walk", err)
					return
				}
			}
		})
	}

	wg.Wait()

	if failed.Load() != 0 {
		t.Fatalf("%d concurrent ops returned errors (see first reported above) — the entcollect path does not share clientMu with the Access path", failed.Load())
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

		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/_pit"):
			// Point-in-time open for the OTel client's Walk enumeration.
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"id":"fakepit"}`)

		case strings.HasSuffix(r.URL.Path, "/_search"):
			body, _ := io.ReadAll(r.Body)
			if !bytes.Contains(body, []byte(`match_all`)) {
				f.record(fmt.Sprintf("search got non-search body on %s: %q", r.URL.Path, body))
				http.Error(w, `{"error":{"type":"parsing_exception","reason":"Unknown key for a START_OBJECT in [v]"}}`, http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"took":1,"pit_id":"fakepit","hits":{"total":{"value":0,"relation":"eq"},"hits":[]}}`)

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
