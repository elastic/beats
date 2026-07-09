// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package entityanalytics

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"

	"go.uber.org/zap/exp/zapslog"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/features"
	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/internal/kvstore"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/paths"
	"github.com/elastic/entcollect"
)

// minimalStateInput implements v2.Input for the minimal-state path.
type minimalStateInput struct {
	provider         entcollect.Provider
	providerName     string
	fullSyncInterval time.Duration
	incrSyncInterval time.Duration
	logger           *logp.Logger
	store            statestore.States
	path             *paths.Path
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

	log := runCtx.Logger.With("provider", n.providerName)

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

	for {
		select {
		case <-runCtx.Cancelation.Done():
			if !errors.Is(runCtx.Cancelation.Err(), context.Canceled) {
				return runCtx.Cancelation.Err()
			}
			return nil

		case <-syncTimer.C:
			if err := syncer.runSync(runCtx, n.provider, client, slogger, true); err != nil {
				log.Errorw("Error running full sync", "error", err)
			}
			syncTimer.Reset(n.fullSyncInterval)
			log.Debugf("Next full sync expected at: %v", time.Now().Add(n.fullSyncInterval))

			if !incrTimer.Stop() {
				select {
				case <-incrTimer.C:
				default:
				}
			}
			incrTimer.Reset(n.incrSyncInterval)

		case <-incrTimer.C:
			if err := syncer.runSync(runCtx, n.provider, client, slogger, false); err != nil {
				log.Errorw("Error running incremental sync", "error", err)
			}
			incrTimer.Reset(n.incrSyncInterval)
			log.Debugf("Next incremental sync expected at: %v", time.Now().Add(n.incrSyncInterval))
		}
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
	s, err := n.store.StoreFor(Name)
	if err != nil {
		return nil, fmt.Errorf("unable to open ES state store: %w", err)
	}
	s.SetID(runCtx.ID)
	log.Infof("Using Elasticsearch-backed state store (index: agentless-state-%s)", runCtx.ID)
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
