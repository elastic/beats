// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package entityanalytics

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"

	"go.uber.org/zap/exp/zapslog"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/features"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/pkg/logp"
	"github.com/elastic/beats/v7/pkg/paths"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/internal/kvstore"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider"
	"github.com/elastic/entcollect"
	"github.com/elastic/go-concert/ctxtool"
)

// minimalStateInput implements v2.Input for the minimal-state path.
type minimalStateInput struct {
	provider         entcollect.Provider
	providerName     string
	fullSyncInterval time.Duration
	incrSyncInterval time.Duration

	// client is the provider's HTTP client, or nil. Run sets its
	// transport to transport, wrapped with the request tracer that
	// tracer configures, except for requests to the URL prefixes in
	// untraced.
	client    *http.Client
	transport http.RoundTripper
	tracer    *provider.TracerConfig
	untraced  []string

	logger *logp.Logger
	store  statestore.States
	path   *paths.Path
}

var _ v2.Input = (*minimalStateInput)(nil)

func (n *minimalStateInput) Name() string {
	return Name
}

func (n *minimalStateInput) Test(_ v2.TestContext) error {
	return nil
}

func (n *minimalStateInput) Run(runCtx v2.Context, connector beat.PipelineConnector) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("minimal input %s panic with: %+v\n%s", runCtx.ID, r, debug.Stack())
			runCtx.Logger.Errorf("Minimal input %s panic: %+v", runCtx.ID, err)
		}
	}()

	runCtx.UpdateStatus(status.Starting, "")
	log := runCtx.Logger.With("provider", n.providerName)

	if n.client != nil {
		ctx := ctxtool.FromCanceller(runCtx.Cancelation)
		var traced http.RoundTripper
		traced, err = provider.TraceRequests(ctx, n.transport, n.tracer, runCtx.Agent.Paths, n.providerName, runCtx.IDWithoutName, log)
		if err != nil {
			return fmt.Errorf("could not set up request tracer: %w", err)
		}
		n.client.Transport, err = provider.SkipTracing(traced, n.transport, n.untraced)
		if err != nil {
			return fmt.Errorf("could not set up request tracer: %w", err)
		}
	}

	client, err := connector.ConnectWith(beat.ClientConfig{
		EventListener: kvstore.NewTxACKHandler(),
	})
	if err != nil {
		return fmt.Errorf("could not connect to publishing pipeline: %w", err)
	}
	defer client.Close()

	syncer, err := n.newSyncer(runCtx, log)
	if err != nil {
		return err
	}
	defer syncer.close()

	slogger := slogLogger(log)

	syncTimer := time.NewTimer(0) // fire immediately on first run
	incrTimer := time.NewTimer(n.incrSyncInterval)
	defer syncTimer.Stop()
	defer incrTimer.Stop()

	// The most recent full and incremental sync errors. The input is
	// degraded while either is set. A successful full sync clears both.
	var fullErr, incrErr error
	retry := retryBackoff{initial: n.incrSyncInterval, max: n.fullSyncInterval}

	runCtx.UpdateStatus(status.Running, "")
	for {
		select {
		case <-runCtx.Cancelation.Done():
			if !errors.Is(runCtx.Cancelation.Err(), context.Canceled) {
				return runCtx.Cancelation.Err()
			}
			return nil

		case <-syncTimer.C:
			err := recoverSync(runCtx, log, syncer, n.provider, client, slogger, true)
			wait := n.fullSyncInterval
			switch {
			case err == nil:
				fullErr, incrErr = nil, nil
				retry.reset()
			case provider.SyncInterrupted(runCtx, err):
				log.Infow(provider.SyncInterruptedMsg, "error", err)
				continue // The input is stopping.
			default:
				log.Errorw("Error running full sync", "error", err)
				fullErr = err
				wait = retry.failed()
			}
			updateSyncStatus(runCtx, fullErr, incrErr, "Successful full sync")
			syncTimer.Reset(wait)
			log.Debugf("Next full sync expected at: %v", time.Now().Add(wait))

			if !incrTimer.Stop() {
				select {
				case <-incrTimer.C:
				default:
				}
			}
			incrTimer.Reset(n.incrSyncInterval)

		case <-incrTimer.C:
			err := recoverSync(runCtx, log, syncer, n.provider, client, slogger, false)
			switch {
			case err == nil:
				incrErr = nil
			case provider.SyncInterrupted(runCtx, err):
				log.Infow(provider.SyncInterruptedMsg, "error", err)
				continue // The input is stopping.
			default:
				log.Errorw("Error running incremental sync", "error", err)
				incrErr = err
			}
			updateSyncStatus(runCtx, fullErr, incrErr, "Successful incremental sync")
			incrTimer.Reset(n.incrSyncInterval)
			log.Debugf("Next incremental sync expected at: %v", time.Now().Add(n.incrSyncInterval))
		}
	}
}

// retryBackoff tracks the wait before retrying a failed full sync. It starts
// at initial and doubles after each further failure, up to max.
type retryBackoff struct {
	initial, max time.Duration
	next         time.Duration // zero means initial
}

// failed returns the wait before the next retry, and doubles the wait
// for the retry after it.
func (b *retryBackoff) failed() time.Duration {
	if b.next == 0 {
		b.next = b.initial
	}
	wait := min(b.next, b.max)
	b.next = min(2*b.next, b.max)
	return wait
}

// reset restarts the backoff after a successful full sync.
func (b *retryBackoff) reset() {
	b.next = 0
}

// recoverSync runs a single sync with s. A panic during the sync is
// logged with its stack and returned as an error, so that one failed
// sync does not stop the input.
func recoverSync(runCtx v2.Context, log *logp.Logger, s syncer, p entcollect.Provider, client beat.Client, slogger *slog.Logger, full bool) (err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("Minimal input %s sync panic: %+v\n%s", runCtx.ID, r, debug.Stack())
			err = fmt.Errorf("sync panic: %+v", r)
		}
	}()
	return s.runSync(runCtx, p, client, slogger, full)
}

// updateSyncStatus reports the input as degraded if the most recent full
// or incremental sync failed, and as running with msg otherwise.
func updateSyncStatus(runCtx v2.Context, fullErr, incrErr error, msg string) {
	switch {
	case fullErr != nil:
		runCtx.UpdateStatus(status.Degraded, fmt.Sprintf("Error running full sync: %v", fullErr))
	case incrErr != nil:
		runCtx.UpdateStatus(status.Degraded, fmt.Sprintf("Error running incremental sync: %v", incrErr))
	default:
		runCtx.UpdateStatus(status.Running, msg)
	}
}

// syncer abstracts state storage for sync operations. The bbolt
// implementation uses transactions for atomicity; the ES-backed
// implementation relies on entcollect.Buffer for batching without
// transactional guarantees.
type syncer interface {
	runSync(runCtx v2.Context, provider entcollect.Provider, client beat.Client, slogger *slog.Logger, full bool) error
	close()
}

func (n *minimalStateInput) newSyncer(runCtx v2.Context, log *logp.Logger) (syncer, error) {
	if features.IsElasticsearchStateStoreEnabledForInput(Name) {
		if n.store == nil {
			return nil, errors.New("ES state store enabled but no statestore was injected")
		}
		return n.newESSyncer(runCtx, log)
	}
	return n.newBBoltSyncer(runCtx, log)
}

// bboltSyncer uses local bbolt storage with transactional semantics.
type bboltSyncer struct {
	store      *kvstore.Store
	bucketName string
}

func (n *minimalStateInput) newBBoltSyncer(runCtx v2.Context, log *logp.Logger) (*bboltSyncer, error) {
	dataDir := n.path.Resolve(paths.Data, "kvstore")
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, fmt.Errorf("kvstore: unable to make data directory: %w", err)
	}
	filename := filepath.Join(dataDir, runCtx.ID+".db")
	store, err := kvstore.NewStore(log, filename, 0o600)
	if err != nil {
		return nil, err
	}
	return &bboltSyncer{
		store:      store,
		bucketName: "entcollect." + n.providerName,
	}, nil
}

func (s *bboltSyncer) runSync(runCtx v2.Context, provider entcollect.Provider, client beat.Client, slogger *slog.Logger, full bool) error {
	ctx := v2.GoContextFromCanceler(runCtx.Cancelation)
	tx, err := s.store.BeginTx(true)
	if err != nil {
		return fmt.Errorf("unable to begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // best-effort cleanup

	es := kvstore.NewEntcollectStore(tx, s.bucketName)
	buf := entcollect.NewBuffer(es)

	tracker := kvstore.NewTxTracker(ctx)
	pub := kvstore.NewPublisher(client, runCtx.ID, tracker)

	if full {
		err = provider.FullSync(ctx, buf, pub, slogger)
	} else {
		err = provider.IncrementalSync(ctx, buf, pub, slogger)
	}

	// Always wait for in-flight events to be ACKed, even on error.
	tracker.Wait()

	if err != nil {
		buf.Discard()
		return err
	}

	if ctx.Err() != nil {
		buf.Discard()
		return ctx.Err()
	}

	if err := buf.Commit(); err != nil {
		return fmt.Errorf("unable to commit buffer: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("unable to commit transaction: %w", err)
	}
	return nil
}

func (s *bboltSyncer) close() {
	s.store.Close()
}

// esSyncer uses the Elasticsearch-backed state store for agentless
// deployments. No transactional atomicity — entcollect.Buffer
// provides application-level batching.
type esSyncer struct {
	store *statestore.Store
}

func (n *minimalStateInput) newESSyncer(runCtx v2.Context, log *logp.Logger) (*esSyncer, error) {
	s, err := n.store.StoreFor(Name, runCtx.ID)
	if err != nil {
		return nil, fmt.Errorf("unable to open ES state store: %w", err)
	}
	log.Info("Using Elasticsearch-backed state store")
	return &esSyncer{store: s}, nil
}

func (s *esSyncer) runSync(runCtx v2.Context, provider entcollect.Provider, client beat.Client, slogger *slog.Logger, full bool) error {
	ctx := v2.GoContextFromCanceler(runCtx.Cancelation)

	es := kvstore.NewStateStoreAdapter(s.store)
	buf := entcollect.NewBuffer(es)

	tracker := kvstore.NewTxTracker(ctx)
	pub := kvstore.NewPublisher(client, runCtx.ID, tracker)

	var err error
	if full {
		err = provider.FullSync(ctx, buf, pub, slogger)
	} else {
		err = provider.IncrementalSync(ctx, buf, pub, slogger)
	}

	// Always wait for in-flight events to be ACKed, even on error.
	tracker.Wait()

	if err != nil {
		buf.Discard()
		return err
	}

	if ctx.Err() != nil {
		buf.Discard()
		return ctx.Err()
	}

	if err := buf.Commit(); err != nil {
		return fmt.Errorf("unable to commit buffer: %w", err)
	}
	return nil
}

func (s *esSyncer) close() {
	s.store.Close()
}

// slogLogger bridges logp.Logger to *slog.Logger using zapslog.
func slogLogger(l *logp.Logger) *slog.Logger {
	opts := []zapslog.HandlerOption{zapslog.WithCaller(true)}
	if name := l.Name(); name != "" {
		opts = append(opts, zapslog.WithName(name))
	}
	return slog.New(zapslog.NewHandler(l.Core(), opts...))
}
