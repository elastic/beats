// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package logstashexporter

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/plog"

	"github.com/elastic/beats/v7/libbeat/outputs"

	"github.com/elastic/beats/v7/libbeat/otelbeat/otelctx"
	"github.com/elastic/beats/v7/libbeat/outputs/logstash"
	"github.com/elastic/beats/v7/x-pack/otel/exporter/logstashexporter/internal"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/transport"
)

const (
	defaultDeadlockTimeout = 5 * time.Minute
)

type logstashExporter struct {
	config    *logstashOutputConfig
	rawConfig *config.C
	logger    *logp.Logger
	workers   []internal.Worker
	workQueue chan *internal.Work
	settings  exporter.Settings
	mu        sync.RWMutex
}

func newLogstashExporter(settings exporter.Settings, cfg component.Config) (*logstashExporter, error) {
	rawConfig, logstashConfig, err := parseLogstashConfig(&cfg)
	if err != nil {
		return nil, err
	}

	logger, err := logp.ConfigureWithCoreLocal(logp.Config{}, settings.Logger.Core())
	if err != nil {
		return nil, err
	}

	// Same as the number of outputs.Client created by the otelconsumer
	workQueueSize := runtime.NumCPU()

	return &logstashExporter{
		config:    logstashConfig,
		rawConfig: rawConfig,
		logger:    logger,
		workQueue: make(chan *internal.Work, workQueueSize),
		settings:  settings,
	}, nil
}

func (*logstashExporter) Start(context.Context, component.Host) error {
	// Clients are initialized on the first ConsumeLogs call and not here on purpose.
	// The context passed to Start doesn't have the necessary values to create
	// the Logstash clients.
	return nil
}

func (l *logstashExporter) Shutdown(context.Context) error {
	return l.shutdownLogstashWorkers()
}

func (l *logstashExporter) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func (l *logstashExporter) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	_, err := l.makeLogstashWorkers(ctx)
	if err != nil {
		return err
	}

	batch, err := internal.NewLogBatch(ctx, ld)
	if err != nil {
		return err
	}

	work := internal.NewWork(batch)
	if err := l.enqueueWork(ctx, work); err != nil {
		return consumererror.NewLogs(err, ld)
	}

	return l.processWorkResult(ctx, ld, work, batch)
}

func (l *logstashExporter) enqueueWork(ctx context.Context, w *internal.Work) error {
	backoff := 5 * time.Millisecond
	maxBackoff := 250 * time.Millisecond
	attempts := 0

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case l.workQueue <- w:
			return nil

		default:
			attempts++
			l.logger.Debugf("Work queue is full, retrying enqueue (attempt %d, backoff %v)", attempts, backoff)
			time.Sleep(backoff)
			if backoff < maxBackoff {
				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
			}
		}
	}
}

func (l *logstashExporter) processWorkResult(
	ctx context.Context,
	ld plog.Logs,
	work *internal.Work,
	batch *internal.LogBatch,
) error {
	for {
		select {
		case <-ctx.Done():
			return consumererror.NewLogs(ctx.Err(), ld)

		case workRes := <-work.Result():
			complete, res := l.processBatchResult(ctx, workRes, ld, batch, work)
			if complete {
				return res
			}

		case <-time.After(defaultDeadlockTimeout):
			// See logstash.deadlockListener for reasoning behind this log.
			l.logger.Warnf("Logstash worker hasn't complete processing in the last %v", defaultDeadlockTimeout)
		}
	}
}

func (l *logstashExporter) processBatchResult(
	ctx context.Context,
	workRes error,
	ld plog.Logs,
	batch *internal.LogBatch,
	work *internal.Work,
) (bool, error) {
	for {
		select {
		case <-ctx.Done():
			return true, consumererror.NewLogs(ctx.Err(), ld)

		case batchRes := <-batch.Result():
			return l.handleBatchResult(ctx, batchRes, workRes, ld, batch, work)

		case <-time.After(defaultDeadlockTimeout):
			// See logstash.deadlockListener for reasoning behind this log.
			l.logger.Warnf("Logstash batch hasn't complete processing in the last %v.", defaultDeadlockTimeout)
		}
	}
}

func (l *logstashExporter) handleBatchResult(
	ctx context.Context,
	batchRes internal.LogBatchResult,
	workRes error,
	ld plog.Logs,
	batch *internal.LogBatch,
	work *internal.Work,
) (bool, error) {
	switch batchRes {
	case internal.LogBatchResultACK:
		// Batch was acknowledged, processing complete
		return true, nil

	case internal.LogBatchResultDrop:
		// Batch was explicitly dropped, report permanent error
		return true, consumererror.NewPermanent(fmt.Errorf("batch was dropped: %w", workRes))

	case internal.LogBatchResultCancelled:
		if err := l.enqueueWork(ctx, work); err != nil {
			return true, consumererror.NewLogs(fmt.Errorf("failed to requeue cancelled batch: %w", err), ld)
		}
		return false, nil

	case internal.LogBatchResultRetry:
		return l.handleRetry(ctx, workRes, ld, batch, work)

	default:
		return true, consumererror.NewPermanent(fmt.Errorf("unexpected batch result: %v", batchRes))
	}
}

func (l *logstashExporter) handleRetry(
	ctx context.Context,
	workRes error,
	ld plog.Logs,
	batch *internal.LogBatch,
	work *internal.Work,
) (bool, error) {
	// Connection errors don't count against retry limit. The Logstash clients might close
	// the connection for different reasons, and workers don't have access to the internal
	// client's state to properly determined when the connection was closed.
	if workRes != nil && errors.Is(workRes, transport.ErrNotConnected) {
		batch.AddRetry(-1)
		if err := l.enqueueWork(ctx, work); err != nil {
			return true, consumererror.NewLogs(fmt.Errorf("failed to requeue batch after connection error: %w", err), ld)
		}
		return false, nil
	}

	//nolint:gosec //G115: MaxRetries is positive.
	if l.config.MaxRetries > 0 && batch.NumRetries() >= uint64(l.config.MaxRetries) {
		return true, consumererror.NewLogs(
			fmt.Errorf("max number of retries exceeded: %d", l.config.MaxRetries),
			ld,
		)
	}

	l.logger.Debugf("Attempt %d of %d to publish events", batch.NumRetries()+1, l.config.MaxRetries)
	if err := l.enqueueWork(ctx, work); err != nil {
		return true, consumererror.NewLogs(fmt.Errorf("failed to requeue batch for retry: %w", err), ld)
	}

	return false, nil
}

func (l *logstashExporter) makeLogstashWorkers(ctx context.Context) ([]internal.Worker, error) {
	if w := l.getWorkers(); w != nil {
		return w, nil
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Re-check after acquiring write lock
	if l.workers != nil {
		return l.workers, nil
	}

	beatVersion := otelctx.GetBeatVersion(ctx)
	beatIndexPrefix := otelctx.GetBeatIndexPrefix(ctx)
	group, err := logstash.MakeLogstashClients(beatVersion, l.logger, outputs.NewNilObserver(), l.rawConfig, beatIndexPrefix)
	if err != nil {
		return nil, err
	}

	workers := make([]internal.Worker, 0, len(group.Clients))
	for _, cli := range group.Clients {
		workers = append(workers, internal.MakeClientWorker(l.workQueue, cli, *l.logger))
	}

	l.workers = workers
	return workers, nil
}

func (l *logstashExporter) getWorkers() []internal.Worker {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.workers
}

func (l *logstashExporter) shutdownLogstashWorkers() error {
	l.mu.Lock()
	closingWorkers := l.workers
	l.workers = nil
	l.mu.Unlock()

	var errs error
	for _, cw := range closingWorkers {
		err := cw.Close()
		if err != nil {
			errs = errors.Join(errs, err)
		}
	}

	return errs
}
