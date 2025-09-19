// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package logstashexporter

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pdata/plog"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/otelbeat/otelctx"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/beats/v7/x-pack/otel/exporter/logstashexporter/internal"
	"github.com/elastic/elastic-agent-libs/transport"
)

const (
	exporterTestDefaultTimeout = 10 * time.Second
)

func TestNewLogstashExporterCreatesValidInstance(t *testing.T) {
	exp := newExporterWithDefaults(t)
	assert.NotNil(t, exp)
	assert.NotNil(t, exp.config)
	assert.NotNil(t, exp.rawConfig)
	assert.NotNil(t, exp.logger)
	assert.Empty(t, exp.workers)
	assert.NotNil(t, exp.workQueue)
	assert.NotNil(t, exp.settings)
	assert.Equal(t, runtime.NumCPU(), cap(exp.workQueue))
}

func TestNewLogstashExporterReturnsErrorOnInvalidConfig(t *testing.T) {
	settings := exporter.Settings{}

	// missing required "hosts" field
	invalidCfg := map[string]any{}

	_, err := newLogstashExporter(settings, invalidCfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing required field")
}

func TestCapabilitiesReturnsMutatesDataFalse(t *testing.T) {
	exp := newExporterWithDefaults(t)
	assert.False(t, exp.Capabilities().MutatesData)
}

func TestConsumeLogs(t *testing.T) {
	exp := newExporterWithDefaults(t)
	tests := []struct {
		name              string
		wantEnqueued      int
		wantErrPermanent  bool
		wantErrRetryable  bool
		wantErrContaining string
		wantBatchRetries  uint64
		publishFn         func(publishCall int, batch publisher.Batch) error
	}{
		{
			name: "ACK",
			publishFn: func(_ int, batch publisher.Batch) error {
				batch.ACK()
				return nil
			},
			wantEnqueued: 1,
		},
		{
			name: "Drop",
			publishFn: func(_ int, batch publisher.Batch) error {
				batch.Drop()
				return nil
			},
			wantEnqueued:      1,
			wantErrPermanent:  true,
			wantErrContaining: "batch was dropped",
		},
		{
			name: "Retry",
			publishFn: func(publishCall int, batch publisher.Batch) error {
				if publishCall == 1 {
					batch.Retry()
				} else {
					batch.ACK()
				}
				return nil
			},
			wantEnqueued:     2,
			wantBatchRetries: 1,
		},
		{
			name: "Retry with max retries exceeded",
			publishFn: func(publishCall int, batch publisher.Batch) error {
				batch.Retry()
				return nil
			},
			wantEnqueued:      10,
			wantBatchRetries:  10,
			wantErrContaining: "max number of retries exceeded",
			wantErrRetryable:  true,
		},
		{
			name: "Retry with ErrNotConnected error",
			publishFn: func(publishCall int, batch publisher.Batch) error {
				switch publishCall {
				case 1, 3:
					batch.Retry()
					return errors.New("some error")
				case 2, 4:
					batch.Retry()
					return transport.ErrNotConnected
				default:
					batch.ACK()
					return nil
				}
			},
			wantEnqueued:     5,
			wantBatchRetries: 2, // ErrNotConnected does not count against retries
		},
		{
			name: "Retry specific events",
			publishFn: func(publishCall int, batch publisher.Batch) error {
				if publishCall == 1 {
					batch.RetryEvents([]publisher.Event{})
				} else {
					batch.ACK()
				}
				return nil
			},
			wantEnqueued:     2,
			wantBatchRetries: 1,
		},
		{
			name: "Cancelled",
			publishFn: func(publishCall int, batch publisher.Batch) error {
				if publishCall == 5 {
					batch.ACK()
				} else {
					batch.Cancelled()
				}
				return nil
			},
			wantEnqueued: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var publishCallCount atomic.Int32
			worker := &mockClientWorker{
				PublishFn: func(ctx context.Context, batch publisher.Batch) error {
					return tt.publishFn(int(publishCallCount.Add(1)), batch)
				},
			}

			clientCtx := newTestBeatsClientContext(t.Context())
			exp.workers = append(exp.workers, worker)

			workerCtx, workerCtxCancel := context.WithCancel(t.Context())
			t.Cleanup(workerCtxCancel)
			worker.run(workerCtx, exp)

			logs := newTestLogs()
			ok, err := runWithTimeout(clientCtx, func(timeoutCtx context.Context) error {
				return exp.ConsumeLogs(timeoutCtx, logs)
			})

			require.True(t, ok, "test timed out")
			assert.Len(t, worker.Enqueued, tt.wantEnqueued)

			if tt.wantEnqueued > 0 {
				// All enqueued batch should be the same instance
				lb, ok := worker.Enqueued[len(worker.Enqueued)-1].Batch().(*internal.LogBatch)
				require.True(t, ok, "expected batch to be of type *internal.LogBatch")
				assert.Equal(t, tt.wantBatchRetries, lb.NumRetries())
			}

			if tt.wantErrRetryable || tt.wantErrPermanent || tt.wantErrContaining != "" {
				require.Error(t, err)
				if tt.wantErrContaining != "" {
					assert.Contains(t, err.Error(), tt.wantErrContaining)
				}
				if tt.wantErrPermanent {
					assert.True(t, consumererror.IsPermanent(err))
				}
				if tt.wantErrRetryable {
					assert.False(t, consumererror.IsPermanent(err))
					var retryableErr consumererror.Logs
					if assert.ErrorAs(t, err, &retryableErr) {
						assert.Equal(t, logs, retryableErr.Data())
					}
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConsumeLogsWithInvalidLogs(t *testing.T) {
	clientCtx := newTestBeatsClientContext(t.Context())
	exp := newExporterWithDefaults(t)
	exp.workers = append(exp.workers, &mockClientWorker{})

	logs := newTestLogs()
	invalidRecord := logs.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().AppendEmpty()
	invalidRecord.Body().SetStr("invalid") // body must be a map

	ok, err := runWithTimeout(clientCtx, func(timeoutCtx context.Context) error {
		return exp.ConsumeLogs(timeoutCtx, logs)
	})

	require.True(t, ok, "test timed out")
	require.Error(t, err)
	assert.True(t, consumererror.IsPermanent(err))
	assert.ErrorContains(t, err, "invalid beats event body")
}

func TestConsumeLogsHandlesCancelledContext(t *testing.T) {
	cancelledCtx, cancel := context.WithCancel(newTestBeatsClientContext(t.Context()))
	cancel()

	exp := newExporterWithDefaults(t)
	exp.workers = append(exp.workers, &mockClientWorker{})

	logs := newTestLogs()
	ok, err := runWithTimeout(t.Context(), func(context.Context) error {
		return exp.ConsumeLogs(cancelledCtx, logs)
	})

	require.True(t, ok, "test timed out")
	require.Error(t, err)
	assert.False(t, consumererror.IsPermanent(err))
	var retryableErr consumererror.Logs
	if assert.ErrorAs(t, err, &retryableErr) {
		assert.Equal(t, logs, retryableErr.Data())
		assert.ErrorIs(t, retryableErr, context.Canceled)
	}
}

func TestGetWorkersReturnsNilWhenNotInitialized(t *testing.T) {
	exp := newExporterWithDefaults(t)
	assert.Nil(t, exp.getWorkers())
}

func TestHasWorkersForReturnsWorkersAfterCreation(t *testing.T) {
	expectedWorkers := []internal.Worker{&mockClientWorker{}}
	exp := newExporterWithDefaults(t)
	exp.workers = expectedWorkers
	assert.Equal(t, expectedWorkers, exp.getWorkers())
}

func TestShutdownClosesWorkers(t *testing.T) {
	exp := newExporterWithDefaults(t)
	worker := &mockClientWorker{CloseErr: errors.New("close error")}
	worker2 := &mockClientWorker{}
	exp.workers = append(exp.workers, worker, worker2)

	_ = exp.Shutdown(t.Context())
	assert.Empty(t, exp.workers)
	assert.True(t, worker.Closed)
	assert.True(t, worker2.Closed)
}

func TestHandleBatchResultReturnsErrorForUnexpectedResult(t *testing.T) {
	exp := newExporterWithDefaults(t)

	done, err := exp.handleBatchResult(
		t.Context(),
		999, // Invalid/unexpected result value
		nil,
		plog.NewLogs(),
		nil,
		nil,
	)

	require.True(t, done)
	require.Error(t, err)
	assert.True(t, consumererror.IsPermanent(err))
	assert.ErrorContains(t, err, "unexpected batch result")
}

func TestMakeLogstashWorkers(t *testing.T) {
	exp := newExporterWithDefaultsWith(t, map[string]any{
		"loadbalance": true,
		"hosts":       []string{"localhost:9999", "localhost:8888"},
	})
	t.Cleanup(func() { _ = exp.Shutdown(t.Context()) })
	clientCtx := newTestBeatsClientContext(t.Context())

	// Initial worker creation
	initialWorkers, err := exp.makeLogstashWorkers(clientCtx)
	require.NoError(t, err)
	assert.Len(t, initialWorkers, 2)
	assert.Equal(t, initialWorkers, exp.getWorkers())

	// Call it again and verify if it returns initial workers
	workers2, err := exp.makeLogstashWorkers(clientCtx)
	require.NoError(t, err)
	assert.Equal(t, initialWorkers, workers2)

	// Test if concurrent calls don't create duplicates
	var wg sync.WaitGroup
	results := make([][]internal.Worker, 10)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			ws, err := exp.makeLogstashWorkers(clientCtx)
			require.NoError(t, err)
			results[idx] = ws
		}(i)
	}
	wg.Wait()

	// All results should be the same
	for i := 1; i < len(results); i++ {
		assert.Equal(t, initialWorkers, results[i])
	}
}

func TestConsumeLogsConcurrency(t *testing.T) {
	exp := newExporterWithDefaults(t)
	clientCtx := newTestBeatsClientContext(t.Context())
	worker := &mockClientWorker{
		PublishFn: func(ctx context.Context, batch publisher.Batch) error {
			time.Sleep(10 * time.Millisecond)
			batch.ACK()
			return nil
		},
	}
	exp.workers = append(exp.workers, worker)
	workerCtx, workerCtxCancel := context.WithCancel(t.Context())
	t.Cleanup(workerCtxCancel)
	worker.run(workerCtx, exp)

	// Run multiple ConsumeLogs calls concurrently
	var wg sync.WaitGroup
	const numConsumers = 10
	errsChan := make(chan error, numConsumers)
	for i := 0; i < numConsumers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			logs := newTestLogs()
			ok, err := runWithTimeout(clientCtx, func(timeoutCtx context.Context) error {
				return exp.ConsumeLogs(timeoutCtx, logs)
			})
			if !ok {
				errsChan <- errors.New("test timed out")
			} else {
				errsChan <- err
			}
		}()
	}

	wg.Wait()
	close(errsChan)

	for err := range errsChan {
		assert.NoError(t, err)
	}

	assert.Len(t, worker.Enqueued, numConsumers)
}

func TestProcessBatchResultHandlesCancelledContext(t *testing.T) {
	cancelledCtx, cancel := context.WithCancel(newTestBeatsClientContext(t.Context()))
	cancel()

	exp := newExporterWithDefaults(t)
	logs := newTestLogs()
	batch, err := internal.NewLogBatch(cancelledCtx, logs)
	require.NoError(t, err)

	ok, err := runWithTimeout(t.Context(), func(context.Context) error {
		_, err := exp.processBatchResult(cancelledCtx, nil, logs, batch, nil)
		return err
	})

	require.True(t, ok, "test timed out")
	assert.False(t, consumererror.IsPermanent(err))
	var retryableErr consumererror.Logs
	if assert.ErrorAs(t, err, &retryableErr) {
		assert.Equal(t, logs, retryableErr.Data())
		assert.ErrorIs(t, retryableErr, context.Canceled)
	}
}

func newExporterWithDefaults(t *testing.T) *logstashExporter {
	return newExporterWithDefaultsWith(t, nil)
}

func newExporterWithDefaultsWith(t *testing.T, extraConfig map[string]any) *logstashExporter {
	settings := exportertest.NewNopSettings(Type)
	defaultConfig := createDefaultConfig()
	var cfg Config
	if c, ok := defaultConfig.(*Config); ok {
		cfg = *c
	} else {
		t.Fatal("default config is not of type *Config")
	}
	cfg["hosts"] = []string{"localhost:9999"}
	cfg["max_retries"] = 10

	for k, v := range extraConfig {
		cfg[k] = v
	}

	exp, err := newLogstashExporter(settings, defaultConfig)
	require.NoError(t, err)
	return exp
}

func newTestLogs() plog.Logs {
	logs := plog.NewLogs()
	resourceLogs := logs.ResourceLogs().AppendEmpty()
	scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()
	logRecord := scopeLogs.LogRecords().AppendEmpty()
	logBody := logRecord.Body().SetEmptyMap()
	logBody.PutStr("value", "test log message")
	return logs
}

func newTestBeatsClientContext(ctx context.Context) context.Context {
	return otelctx.NewConsumerContext(ctx, beat.Info{
		Beat:        "test-beat",
		Version:     "1.0.0",
		IndexPrefix: "test-index",
	})
}

func runWithTimeout(ctx context.Context, fn func(context.Context) error) (bool, error) {
	result := make(chan error, 1)
	timeoutCtx, cancel := context.WithTimeout(ctx, exporterTestDefaultTimeout)
	defer cancel()

	go func() {
		result <- fn(timeoutCtx)
	}()

	select {
	case <-timeoutCtx.Done():
		return false, errors.New("timed out")
	case err := <-result:
		return true, err
	}
}

type mockClientWorker struct {
	Closed     bool
	Enqueued   []*internal.Work
	EnqueueErr error
	CloseErr   error
	PublishFn  func(ctx context.Context, batch publisher.Batch) error
}

func (m *mockClientWorker) Publish(ctx context.Context, batch publisher.Batch) error {
	if m.PublishFn != nil {
		return m.PublishFn(ctx, batch)
	}
	return nil
}

func (m *mockClientWorker) String() string {
	return "mockClientWorker"
}

func (m *mockClientWorker) run(ctx context.Context, exp *logstashExporter) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case work := <-exp.workQueue:
				m.Enqueued = append(m.Enqueued, work)
				work.Result() <- m.Publish(ctx, work.Batch())
			}
		}
	}()
}

func (m *mockClientWorker) Close() error {
	m.Closed = true
	return m.CloseErr
}
