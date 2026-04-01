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

	dataDir := n.path.Resolve(paths.Data, "kvstore")
	if err = os.MkdirAll(dataDir, 0700); err != nil {
		return fmt.Errorf("kvstore: unable to make data directory: %w", err)
	}
	filename := filepath.Join(dataDir, runCtx.ID+".db")
	store, err := kvstore.NewStore(log, filename, 0600)
	if err != nil {
		return err
	}
	defer store.Close()

	slogger := slogLogger(log)
	bucketName := "entcollect." + n.providerName

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
			if err := n.runSync(runCtx, store, client, slogger, bucketName, true); err != nil {
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
			if err := n.runSync(runCtx, store, client, slogger, bucketName, false); err != nil {
				log.Errorw("Error running incremental sync", "error", err)
			}
			incrTimer.Reset(n.incrSyncInterval)
			log.Debugf("Next incremental sync expected at: %v", time.Now().Add(n.incrSyncInterval))
		}
	}
}

func (n *minimalStateInput) runSync(
	runCtx v2.Context,
	store *kvstore.Store,
	client beat.Client,
	slogger *slog.Logger,
	bucketName string,
	full bool,
) error {
	ctx := v2.GoContextFromCanceler(runCtx.Cancelation)
	tx, err := store.BeginTx(true)
	if err != nil {
		return fmt.Errorf("unable to begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // best-effort cleanup

	es := kvstore.NewEntcollectStore(tx, bucketName)
	buf := entcollect.NewBuffer(es)

	tracker := kvstore.NewTxTracker(ctx)
	pub := kvstore.NewPublisher(client, runCtx.ID, tracker)

	if full {
		err = n.provider.FullSync(ctx, buf, pub, slogger)
	} else {
		err = n.provider.IncrementalSync(ctx, buf, pub, slogger)
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

// slogLogger bridges logp.Logger to *slog.Logger using zapslog.
func slogLogger(l *logp.Logger) *slog.Logger {
	opts := []zapslog.HandlerOption{zapslog.WithCaller(true)}
	if name := l.Name(); name != "" {
		opts = append(opts, zapslog.WithName(name))
	}
	return slog.New(zapslog.NewHandler(l.Core(), opts...))
}
