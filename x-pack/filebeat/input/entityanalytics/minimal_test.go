// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package entityanalytics

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/beats/v7/pkg/config"
	"github.com/elastic/beats/v7/pkg/logp/logptest"
	"github.com/elastic/beats/v7/pkg/paths"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/internal/kvstore"
	"github.com/elastic/entcollect"
)

// fakeProvider is an in-memory entcollect.Provider for testing.
type fakeProvider struct {
	mu       sync.Mutex
	fullFunc func(ctx context.Context, store entcollect.Store, pub entcollect.Publisher, log *slog.Logger) error
	incrFunc func(ctx context.Context, store entcollect.Store, pub entcollect.Publisher, log *slog.Logger) error
}

func (p *fakeProvider) FullSync(ctx context.Context, store entcollect.Store, pub entcollect.Publisher, log *slog.Logger) error {
	p.mu.Lock()
	fn := p.fullFunc
	p.mu.Unlock()
	if fn != nil {
		return fn(ctx, store, pub, log)
	}
	return nil
}

func (p *fakeProvider) IncrementalSync(ctx context.Context, store entcollect.Store, pub entcollect.Publisher, log *slog.Logger) error {
	p.mu.Lock()
	fn := p.incrFunc
	p.mu.Unlock()
	if fn != nil {
		return fn(ctx, store, pub, log)
	}
	return nil
}

type fakeClient struct {
	mu     sync.Mutex
	events []beat.Event
}

func (c *fakeClient) Publish(event beat.Event) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, event)
}

func (c *fakeClient) PublishAll(events []beat.Event) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, events...)
}

func (c *fakeClient) Close() error { return nil }

func (c *fakeClient) Events() []beat.Event {
	c.mu.Lock()
	defer c.mu.Unlock()
	dst := make([]beat.Event, len(c.events))
	copy(dst, c.events)
	return dst
}

type fakeConnector struct {
	client beat.Client
}

func (f *fakeConnector) ConnectWith(cfg beat.ClientConfig) (beat.Client, error) {
	return f.client, nil
}

func (f *fakeConnector) Connect() (beat.Client, error) {
	return f.client, nil
}

func (f *fakeConnector) Disconnect(ctx context.Context) error {
	return f.client.Close()
}
func TestMinimalInput_RunSync_FullCycle(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	dbFile := filepath.Join(tmpDir, "test.db")

	log := logptest.NewTestingLogger(t, "test")
	store, err := kvstore.NewStore(log, dbFile, 0o600)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer store.Close()

	client := &fakeClient{}
	bucketName := "entcollect.testprov"
	slogger := slogLogger(log)

	prov := &fakeProvider{
		fullFunc: func(ctx context.Context, s entcollect.Store, pub entcollect.Publisher, log *slog.Logger) error {
			if err := s.Set("cursor", "page-2"); err != nil {
				return err
			}
			return pub(ctx, entcollect.Document{
				ID:        "user-1",
				Kind:      entcollect.KindUser,
				Action:    entcollect.ActionDiscovered,
				Timestamp: time.Now(),
				Fields:    map[string]any{"user.name": "alice"},
			})
		},
	}

	ctx := t.Context()

	ackClient := &ackingClient{inner: client}
	s := &bboltSyncer{store: store, bucketName: bucketName}

	err = s.runSync(
		v2.Context{
			Logger:      log,
			ID:          "test-input-1",
			Cancelation: v2.GoContextFromCanceler(ctx),
		},
		prov,
		ackClient,
		slogger,
		true,
	)
	if err != nil {
		t.Fatalf("runSync: %v", err)
	}

	events := client.Events()
	if got := len(events); got != 1 {
		t.Fatalf("len(events) = %d; want 1", got)
	}

	action, _ := events[0].Fields.GetValue("event.action")
	if action != "user-discovered" {
		t.Errorf("event.action = %v; want user-discovered", action)
	}

	var cursor string
	err = store.RunTransaction(false, func(tx *kvstore.Transaction) error {
		return tx.Get([]byte(bucketName), []byte("cursor"), &cursor)
	})
	if err != nil {
		t.Fatalf("reading cursor: %v", err)
	}
	if cursor != "page-2" {
		t.Errorf("cursor = %q; want %q", cursor, "page-2")
	}
}

func TestMinimalInput_RunSync_ErrorDiscardsBuffer(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	dbFile := filepath.Join(tmpDir, "test-err.db")

	log := logptest.NewTestingLogger(t, "test")
	store, err := kvstore.NewStore(log, dbFile, 0o600)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer store.Close()

	client := &fakeClient{}
	ackClient := &ackingClient{inner: client}
	bucketName := "entcollect.testprov"
	slogger := slogLogger(log)

	syncErr := errors.New("provider failure")
	prov := &fakeProvider{
		fullFunc: func(ctx context.Context, s entcollect.Store, pub entcollect.Publisher, log *slog.Logger) error {
			if err := s.Set("cursor", "bad"); err != nil {
				return err
			}
			_ = pub(ctx, entcollect.Document{
				ID:        "user-1",
				Kind:      entcollect.KindUser,
				Action:    entcollect.ActionDiscovered,
				Timestamp: time.Now(),
			})
			return syncErr
		},
	}

	ctx := context.Background()
	s := &bboltSyncer{store: store, bucketName: bucketName}

	err = s.runSync(
		v2.Context{
			Logger:      log,
			ID:          "test-input-err",
			Cancelation: v2.GoContextFromCanceler(ctx),
		},
		prov,
		ackClient,
		slogger,
		true,
	)
	if !errors.Is(err, syncErr) {
		t.Fatalf("runSync = %v; want %v", err, syncErr)
	}

	// The cursor must NOT be committed (buffer was discarded, tx rolled back).
	var cursor string
	txErr := store.RunTransaction(false, func(tx *kvstore.Transaction) error {
		return tx.Get([]byte(bucketName), []byte("cursor"), &cursor)
	})
	if txErr == nil {
		t.Fatal("cursor should not exist after failed sync, but Get succeeded")
	}
}

func TestMinimalInput_RunCancelation(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	prov := &fakeProvider{}
	mi := &minimalStateInput{
		provider:         prov,
		providerName:     "testprov",
		fullSyncInterval: time.Hour,
		incrSyncInterval: time.Hour,
		logger:           logptest.NewTestingLogger(t, "test"),
		path:             &paths.Path{Data: tmpDir},
	}

	ctx, cancel := context.WithCancel(context.Background())

	client := &fakeClient{}
	connector := &fakeConnector{client: client}

	done := make(chan error, 1)
	go func() {
		done <- mi.Run(
			v2.Context{
				Logger:      logptest.NewTestingLogger(t, "test"),
				ID:          "test-cancel",
				Cancelation: v2.GoContextFromCanceler(ctx),
			},
			connector,
		)
	}()

	// Let the first sync fire (timer starts at 0).
	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run = %v; want nil", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after cancel")
	}
}

// TestMinimalInput_RunFullSyncRetry checks that a failed or panicking full
// sync is retried well before the full sync interval, and that the input
// is reported degraded until a full sync succeeds.
func TestMinimalInput_RunFullSyncRetry(t *testing.T) {
	t.Parallel()

	syncErr := errors.New("provider failure")
	var calls atomic.Int32
	prov := &fakeProvider{
		fullFunc: func(ctx context.Context, s entcollect.Store, pub entcollect.Publisher, log *slog.Logger) error {
			switch calls.Add(1) {
			case 1:
				panic("bad response")
			case 2:
				return syncErr
			}
			return nil
		},
	}
	mi := newTestMinimalInput(t, prov, time.Hour, 20*time.Millisecond)

	recovered := statusUpdate{status.Running, "Successful full sync"}
	updates := runUntil(t, mi, func(u []statusUpdate) bool {
		return slices.Contains(u, recovered)
	})

	start := []statusUpdate{{status.Starting, ""}, {status.Running, ""}}
	if len(updates) < 2 || !slices.Equal(updates[:2], start) {
		t.Errorf("first updates = %v; want %v", updates[:min(2, len(updates))], start)
	}
	var (
		sawPanic, sawErr bool
		failed           = -1
		success          = slices.Index(updates, recovered)
	)
	for i, u := range updates {
		switch {
		case u.status == status.Degraded && strings.Contains(u.msg, "sync panic: bad response"):
			sawPanic = true
		case u.status == status.Degraded && strings.Contains(u.msg, syncErr.Error()):
			sawErr = true
		}
		if u.status == status.Degraded && failed < 0 {
			failed = i
		}
	}
	if !sawPanic {
		t.Error("no degraded status for the panicking full sync")
	}
	if !sawErr {
		t.Error("no degraded status for the failed full sync")
	}
	if failed < 0 || success < failed {
		t.Fatalf("want degraded then running after a successful full sync; got %v", updates)
	}
	for _, u := range updates[failed:success] {
		if u.status != status.Degraded {
			t.Errorf("status %v %q reported before a full sync succeeded; want degraded", u.status, u.msg)
		}
	}
}

// TestMinimalInput_RunFullSyncBackoff checks that the wait between full
// sync retries grows, and that it starts again from the incremental sync
// interval after a successful full sync.
func TestMinimalInput_RunFullSyncBackoff(t *testing.T) {
	t.Parallel()

	// Full syncs 1 to 6 fail, 7 succeeds, 8 fails and 9 succeeds.
	var (
		mu    sync.Mutex
		times []time.Time
	)
	prov := &fakeProvider{
		fullFunc: func(ctx context.Context, s entcollect.Store, pub entcollect.Publisher, log *slog.Logger) error {
			mu.Lock()
			defer mu.Unlock()
			times = append(times, time.Now())
			switch len(times) {
			case 7, 9:
				return nil
			}
			return errors.New("provider failure")
		},
	}
	const incr = 10 * time.Millisecond
	mi := newTestMinimalInput(t, prov, 700*time.Millisecond, incr)

	success := statusUpdate{status.Running, "Successful full sync"}
	runUntil(t, mi, func(u []statusUpdate) bool {
		n := 0
		for _, v := range u {
			if v == success {
				n++
			}
		}
		return n == 2
	})

	mu.Lock()
	defer mu.Unlock()
	if len(times) < 9 {
		t.Fatalf("got %d full syncs; want 9", len(times))
	}
	var gaps []time.Duration
	for i := 1; i < len(times); i++ {
		gaps = append(gaps, times[i].Sub(times[i-1]))
	}
	// A timer never fires early, so each retry waits at least its nominal
	// wait: 10, 20, 40, 80, 160 and 320ms.
	for i, want := 0, incr; i < 6; i, want = i+1, 2*want {
		if gaps[i] < want {
			t.Errorf("retry %d waited %v; want at least %v (gaps %v)", i+1, gaps[i], want, gaps)
		}
	}
	// After the success the backoff starts again, so the retry after the
	// next failure waits a nominal 10ms, not 640ms.
	if gaps[7] >= gaps[5] {
		t.Errorf("retry after a success waited %v, no shorter than the last retry before it %v; want reset (gaps %v)", gaps[7], gaps[5], gaps)
	}
}

// TestMinimalInput_RunIncrementalSyncStatus checks that a failed
// incremental sync is reported as degraded, and that the next successful
// one reports running.
func TestMinimalInput_RunIncrementalSyncStatus(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	prov := &fakeProvider{
		incrFunc: func(ctx context.Context, s entcollect.Store, pub entcollect.Publisher, log *slog.Logger) error {
			if calls.Add(1) == 1 {
				return errors.New("incremental failure")
			}
			return nil
		},
	}
	mi := newTestMinimalInput(t, prov, time.Hour, 20*time.Millisecond)

	failed := statusUpdate{status.Degraded, "Error running incremental sync: incremental failure"}
	recovered := statusUpdate{status.Running, "Successful incremental sync"}
	updates := runUntil(t, mi, func(u []statusUpdate) bool {
		i := slices.Index(u, failed)
		return i >= 0 && slices.Contains(u[i:], recovered)
	})
	for _, u := range updates[slices.Index(updates, failed):slices.Index(updates, recovered)] {
		if u.status != status.Degraded {
			t.Errorf("status %v %q reported before an incremental sync succeeded; want degraded", u.status, u.msg)
		}
	}
}

// TestMinimalInput_RunFullSyncClearsIncrementalError checks that a
// successful full sync reports running even if the last incremental sync
// failed.
func TestMinimalInput_RunFullSyncClearsIncrementalError(t *testing.T) {
	t.Parallel()

	prov := &fakeProvider{
		incrFunc: func(ctx context.Context, s entcollect.Store, pub entcollect.Publisher, log *slog.Logger) error {
			return errors.New("incremental failure")
		},
	}
	mi := newTestMinimalInput(t, prov, 100*time.Millisecond, 20*time.Millisecond)

	failed := statusUpdate{status.Degraded, "Error running incremental sync: incremental failure"}
	recovered := statusUpdate{status.Running, "Successful full sync"}
	runUntil(t, mi, func(u []statusUpdate) bool {
		i := slices.Index(u, failed)
		return i >= 0 && slices.Contains(u[i:], recovered)
	})
}

// TestRetryBackoff checks the doubling, the cap and the reset of the full
// sync retry wait.
func TestRetryBackoff(t *testing.T) {
	t.Parallel()

	b := retryBackoff{initial: 10 * time.Second, max: time.Minute}
	steps := []struct {
		reset bool
		want  time.Duration
	}{
		{want: 10 * time.Second},
		{want: 20 * time.Second},
		{want: 40 * time.Second},
		{want: time.Minute},
		{want: time.Minute},
		{reset: true, want: 10 * time.Second},
		{want: 20 * time.Second},
	}
	for i, step := range steps {
		if step.reset {
			b.reset()
		}
		if got := b.failed(); got != step.want {
			t.Errorf("step %d: failed() = %v; want %v", i, got, step.want)
		}
	}
}

// newTestMinimalInput returns a minimal-state input using prov with a
// bbolt store in a temporary directory.
func newTestMinimalInput(t *testing.T, prov entcollect.Provider, full, incr time.Duration) *minimalStateInput {
	return &minimalStateInput{
		provider:         prov,
		providerName:     "testprov",
		fullSyncInterval: full,
		incrSyncInterval: incr,
		logger:           logptest.NewTestingLogger(t, "test"),
		path:             &paths.Path{Data: t.TempDir()},
	}
}

// runUntil runs mi until the status updates it reports satisfy done, then
// stops it and returns the updates.
func runUntil(t *testing.T, mi *minimalStateInput, done func([]statusUpdate) bool) []statusUpdate {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	reporter := &statusRecorder{done: done, seen: make(chan struct{})}
	runErr := make(chan error, 1)
	go func() {
		runErr <- mi.Run(
			v2.Context{
				Logger:      logptest.NewTestingLogger(t, "test"),
				ID:          "test-" + t.Name(),
				Cancelation: v2.GoContextFromCanceler(ctx),
			}.WithStatusReporter(reporter),
			&fakeConnector{client: &ackingClient{inner: &fakeClient{}}},
		)
	}()

	timedOut := false
	select {
	case <-reporter.seen:
	case err := <-runErr:
		t.Fatalf("Run returned %v early; updates: %v", err, reporter.updates())
	case <-time.After(5 * time.Second):
		timedOut = true
	}
	// Stop Run before failing, so it does not log after the test ends.
	cancel()
	select {
	case err := <-runErr:
		if err != nil {
			t.Errorf("Run = %v; want nil", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after cancel")
	}
	if timedOut {
		t.Fatalf("timed out waiting for status updates; got %v", reporter.updates())
	}
	return reporter.updates()
}

// statusRecorder records status updates. It closes seen the first time
// the recorded updates satisfy done.
type statusRecorder struct {
	done func([]statusUpdate) bool
	seen chan struct{}

	mu     sync.Mutex
	upds   []statusUpdate
	closed bool
}

type statusUpdate struct {
	status status.Status
	msg    string
}

func (r *statusRecorder) UpdateStatus(s status.Status, msg string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.upds = append(r.upds, statusUpdate{s, msg})
	if !r.closed && r.done(r.upds) {
		close(r.seen)
		r.closed = true
	}
}

func (r *statusRecorder) updates() []statusUpdate {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]statusUpdate(nil), r.upds...)
}

// ackingClient wraps a fakeClient and immediately ACKs each event by
// calling tracker.Ack() after delegating to the inner client.
type ackingClient struct {
	inner *fakeClient
}

func (c *ackingClient) Publish(event beat.Event) {
	c.inner.Publish(event)
	if t, ok := event.Private.(*kvstore.TxTracker); ok {
		t.Ack()
	}
}

func (c *ackingClient) PublishAll(events []beat.Event) {
	for _, e := range events {
		c.Publish(e)
	}
}

func (c *ackingClient) Close() error { return nil }

// TestMinimalInput_RequestTracer creates minimal-state inputs from
// configurations like those users and the Fleet integrations send, runs
// each against test TLS servers that need ssl.verification_mode: none,
// and checks that the configured client is used, that requests are
// written to the input's trace file in the logs directory, and that
// credentials are not.
func TestMinimalInput_RequestTracer(t *testing.T) {
	t.Run("jamf", func(t *testing.T) {
		fixture := newAtomicFixture(makeComputersJSON(t,
			testComputer{Name: "laptop", UDID: "TRACE-1", Managed: true},
		))
		srv := startJamfIntegServer(t, fixture)
		cfg := fmt.Sprintf(`
provider: jamf
use_minimal_state: true
jamf_tenant: %s
jamf_username: testuser
jamf_password: testpass
request.ssl.verification_mode: none
tracer.filename: "../../logs/jamf/http-request-trace-*.ndjson"
`, hostFromURL(t, srv.URL))
		// The input may be run again after a restart of its runner;
		// each run must trace each request once.
		trace := runTracedInput(t, cfg, "jamf", "/api/preview/computers", 2)
		// Jamf sends the credentials as HTTP basic auth.
		creds := base64.StdEncoding.EncodeToString([]byte("testuser:testpass"))
		if bytes.Contains(trace, []byte(creds)) {
			t.Errorf("trace records the credentials:\n%s", trace)
		}
	})

	t.Run("azure-ad", func(t *testing.T) {
		login := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "ACCESS-TOKEN-VALUE", "token_type": "Bearer", "expires_in": 3600})
		}))
		defer login.Close()
		var graphURL string
		graph := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := map[string]any{"value": []any{}}
			if strings.HasSuffix(r.URL.Path, "/delta") {
				resp["@odata.deltaLink"] = graphURL + r.URL.Path + "?$deltatoken=x"
			}
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer graph.Close()
		graphURL = graph.URL
		// Rendered from the entityanalytics_entra_id template with
		// minimal state, the request tracer, an ssl block and the
		// direct reports option enabled.
		cfg := fmt.Sprintf(`
provider: azure-ad
use_minimal_state: true
tracer.filename: "../../logs/azure-ad/http-request-trace-*.ndjson"
tracer.maxbackups: 5
tracer.maxsize: 1
client_id: test-client
tenant_id: test-tenant
secret: SUPER-SECRET-VALUE
login_endpoint: %s
api_endpoint: %s/v1.0
dataset: all
login_scopes:
  - https://graph.microsoft.com/.default
expand.users:
  - directReports
ssl:
  verification_mode: none
timeout: 30s
`, login.URL, graph.URL)
		trace := runTracedInput(t, cfg, "azure-ad", "/v1.0/users/delta", 1)
		for _, secret := range []string{"SUPER-SECRET-VALUE", "ACCESS-TOKEN-VALUE"} {
			if bytes.Contains(trace, []byte(secret)) {
				t.Errorf("trace records %s:\n%s", secret, trace)
			}
		}
	})

	t.Run("okta", func(t *testing.T) {
		srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("[]"))
		}))
		defer srv.Close()
		// Rendered from the entityanalytics_okta template with minimal
		// state, the request tracer and an ssl block.
		cfg := fmt.Sprintf(`
provider: okta
use_minimal_state: true
tracer.enabled: true
tracer.filename: "../../logs/okta/http-request-trace-*.ndjson"
tracer.maxbackups: 5
okta_domain: %s
okta_token: SUPER-SECRET-TOKEN
dataset: users
enrich_with:
  - groups
request.ssl.verification_mode: none
`, hostFromURL(t, srv.URL))
		trace := runTracedInput(t, cfg, "okta", "/api/v1/users", 1)
		if bytes.Contains(trace, []byte("SUPER-SECRET-TOKEN")) {
			t.Errorf("trace records the API token:\n%s", trace)
		}
	})
}

// runTracedInput creates an input from the YAML configuration cfg and runs
// it n times. Each run waits until the trace file for the named provider
// records one more successful response to a request whose path starts
// with want, and checks that no sync failed. It returns the trace file
// contents.
func runTracedInput(t *testing.T, cfg, name, want string, n int) []byte {
	t.Helper()
	c, err := config.NewConfigWithYAML([]byte(cfg), "test")
	if err != nil {
		t.Fatalf("parsing config: %v", err)
	}
	home := t.TempDir()
	logs := filepath.Join(home, "logs")
	log, obs := logptest.NewTestingLoggerWithObserver(t, "")
	plugin := Plugin(log, nil, &paths.Path{Home: home, Data: filepath.Join(home, "data"), Logs: logs})
	inp, err := plugin.Manager.Create(c)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	const id = "trace-test"
	trace := filepath.Join(logs, name, "http-request-trace-"+id+".ndjson")
	var b []byte
	for run := 1; run <= n; run++ {
		ctx, cancel := context.WithCancel(t.Context())
		done := make(chan error, 1)
		go func() {
			done <- inp.Run(
				v2.Context{
					Logger:        log,
					ID:            id,
					IDWithoutName: id,
					Agent:         beat.Info{Paths: &paths.Path{Logs: logs}},
					Cancelation:   v2.GoContextFromCanceler(ctx),
				},
				&fakeConnector{client: &fakeClient{}},
			)
		}()
		deadline := time.Now().Add(10 * time.Second)
		for time.Now().Before(deadline) {
			b, _ = os.ReadFile(trace)
			if tracedRequests(b, want) >= run {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
		cancel()
		if err := <-done; err != nil {
			t.Errorf("run %d: Run = %v; want nil", run, err)
		}
		if errs := obs.FilterMessageSnippet("Error running").All(); len(errs) != 0 {
			t.Fatalf("run %d: sync failed: %v", run, errs)
		}
		if got := tracedRequests(b, want); got != run {
			t.Fatalf("run %d: trace %s records %d requests to %s; want %d\n%s", run, trace, got, want, run, b)
		}
	}
	return b
}

// tracedRequests returns the number of 200 responses in trace to requests
// whose path starts with path. The tracer logs each response after the
// round trip completes, so a request that failed, for example on a TLS
// error, is not counted. Responses are matched to their requests by
// transaction ID.
func tracedRequests(trace []byte, path string) int {
	type entry struct {
		Message string `json:"message"`
		TxID    string `json:"transaction.id"`
		Path    string `json:"url.path"`
		Status  int    `json:"http.response.status_code"`
	}
	wanted := make(map[string]bool)
	var n int
	for line := range bytes.Lines(trace) {
		var e entry
		if json.Unmarshal(line, &e) != nil {
			continue
		}
		switch e.Message {
		case "HTTP request":
			wanted[e.TxID] = strings.HasPrefix(e.Path, path)
		case "HTTP response":
			if wanted[e.TxID] && e.Status == http.StatusOK {
				n++
			}
		}
	}
	return n
}
