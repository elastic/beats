// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"
	"text/tabwriter"
	"time"

	sqsTypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/dustin/go-humanize"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

func TestBenchmarkInputSQSV2(t *testing.T) {
	log := logptest.NewTestingLogger(t, inputName)
	results := []testing.BenchmarkResult{
		benchmarkInputSQSV2(t, log, 1),
		benchmarkInputSQSV2(t, log, 2),
		benchmarkInputSQSV2(t, log, 4),
		benchmarkInputSQSV2(t, log, 8),
		benchmarkInputSQSV2(t, log, 16),
		benchmarkInputSQSV2(t, log, 32),
		benchmarkInputSQSV2(t, log, 64),
		benchmarkInputSQSV2(t, log, 128),
		benchmarkInputSQSV2(t, log, 256),
		benchmarkInputSQSV2(t, log, 512),
		benchmarkInputSQSV2(t, log, 1024),
	}

	headers := []string{
		"WORKERS",
		"EVENTS PER SEC",
		"S3 BYTES PER SEC",
		"SQS MSGS PER SEC",
		"TIME (SEC)",
		"CPUS",
	}
	data := make([][]string, 0)
	for _, r := range results {
		data = append(data, []string{
			fmt.Sprintf("%.0f", r.Extra["number_of_workers"]),
			fmt.Sprintf("%.0f", r.Extra["events_per_sec"]),
			fmt.Sprintf("%v", humanize.Bytes(uint64(r.Extra["s3_bytes_per_sec"]))),
			fmt.Sprintf("%.0f", r.Extra["sqs_messages_per_sec"]),
			fmt.Sprintf("%.3f", r.Extra["sec"]),
			fmt.Sprintf("%v", runtime.GOMAXPROCS(0)),
		})
	}

	w := tabwriter.NewWriter(os.Stdout, 1, 1, 1, ' ', tabwriter.Debug)
	fmt.Fprintln(w, strings.Join(headers, "\t"))
	for _, d := range data {
		fmt.Fprintln(w, strings.Join(d, "\t"))
	}
	require.NoError(t, w.Flush())
}

func benchmarkInputSQSV2(t *testing.T, log *logp.Logger, workerCount int) testing.BenchmarkResult {
	return testing.Benchmark(func(b *testing.B) {
		config := makeBenchmarkConfig(t)
		config.NumberOfWorkers = workerCount
		config.QueueURL = "https://sqs.us-east-1.amazonaws.com/1234/test"
		config.RegionName = "us-east-1"

		sqsAPI, err := newConstantSQS()
		require.NoError(t, err)
		s3API := newConstantS3(t)

		reg := monitoring.NewRegistry()
		metrics := newInputMetrics(reg, workerCount, logp.NewNopLogger())
		b.Cleanup(metrics.Close)

		processor := newObjectProcessorV2(s3API, metrics, config.getFileSelectors(), config.BackupConfig)

		disc := newSQSDiscoveryV2(sqsDiscoveryV2Config{
			SQS:               sqsAPI,
			S3Move:            s3API,
			QueueURL:          config.QueueURL,
			VisibilityTimeout: config.VisibilityTimeout,
			MaxReceiveCount:   config.SQSMaxReceiveCount,
			Processor:         processor,
			Metrics:           metrics,
			Log:               log.Named("sqs"),
			Status:            &statusReporterHelperMock{},
		})

		cc := newConcurrencyController(concurrencyControllerConfig{
			MaxWorkers:     workerCount,
			AdjustCooldown: 5 * time.Second,
			Log:            log.Named("flow"),
			Registry:       monitoring.NewRegistry(),
		})

		pipeline := newFakePipeline()

		ctx, cancel := context.WithCancel(context.Background())
		b.Cleanup(cancel)

		go func() {
			target := uint64(b.N) //nolint:gosec // b.N is non-negative
			for metrics.sqsMessagesReceivedTotal.Get() < target {
				time.Sleep(5 * time.Millisecond)
			}
			cancel()
		}()

		sem := make(chan struct{}, workerCount)
		var wg sync.WaitGroup

		b.ResetTimer()
		start := time.Now()

		disc.ReceiveLoop(ctx, workerCount, func(msgCtx context.Context, msg sqsTypes.Message) {
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			wg.Add(1)
			go func() {
				defer func() { <-sem; wg.Done() }()
				processSQSMessageV2Bench(msgCtx, disc, cc, msg, pipeline, metrics)
			}()
		})

		wg.Wait()
		b.StopTimer()
		elapsed := time.Since(start)

		b.ReportMetric(float64(workerCount), "number_of_workers")
		b.ReportMetric(elapsed.Seconds(), "sec")

		b.ReportMetric(float64(metrics.s3EventsCreatedTotal.Get()), "events")
		b.ReportMetric(float64(metrics.s3EventsCreatedTotal.Get())/elapsed.Seconds(), "events_per_sec")

		b.ReportMetric(float64(metrics.s3BytesProcessedTotal.Get()), "s3_bytes")
		b.ReportMetric(float64(metrics.s3BytesProcessedTotal.Get())/elapsed.Seconds(), "s3_bytes_per_sec")

		b.ReportMetric(float64(metrics.sqsMessagesDeletedTotal.Get()), "sqs_messages")
		b.ReportMetric(float64(metrics.sqsMessagesDeletedTotal.Get())/elapsed.Seconds(), "sqs_messages_per_sec")
	})
}

// processSQSMessageV2Bench mirrors inputV2.processSQSMessage but is a free
// function so the benchmark doesn't need a full inputV2 instance.
func processSQSMessageV2Bench(ctx context.Context, disc *sqsDiscoveryV2, cc *concurrencyController, msg sqsTypes.Message, pipeline beat.Pipeline, metrics *inputMetrics) {
	id := metrics.beginSQSWorker()
	defer metrics.endSQSWorker(id)

	acks := newAWSACKHandler()
	client, err := createPipelineClient(pipeline, acks)
	if err != nil {
		return
	}
	defer func() { acks.Close(); client.Close() }()

	publishCount := 0
	result := disc.ProcessMessage(ctx, &msg, func(e beat.Event) {
		publishWithBackpressure(cc, 50*time.Millisecond, func() {
			client.Publish(e)
		})
		metrics.s3EventsCreatedTotal.Inc()
		publishCount++
	})

	if publishCount == 0 {
		result.Done()
	} else {
		acks.Add(publishCount, result.Done)
	}
}
