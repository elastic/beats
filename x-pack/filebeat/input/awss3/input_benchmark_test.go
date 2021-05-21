// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/dustin/go-humanize"
	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/monitoring"
	pubtest "github.com/elastic/beats/v7/libbeat/publisher/testing"
)

const cloudtrailTestFile = "testdata/aws-cloudtrail.json.gz"

type constantSQS struct {
	msgs []sqs.Message
}

var _ sqsAPI = (*constantSQS)(nil)

func newConstantSQS() *constantSQS {
	return &constantSQS{
		msgs: []sqs.Message{
			newSQSMessage(newS3Event(filepath.Base(cloudtrailTestFile))),
		},
	}
}

func (c *constantSQS) ReceiveMessage(ctx context.Context, maxMessages int) ([]sqs.Message, error) {
	return c.msgs, nil
}

func (_ *constantSQS) DeleteMessage(ctx context.Context, msg *sqs.Message) error {
	return nil
}

func (_ *constantSQS) ChangeMessageVisibility(ctx context.Context, msg *sqs.Message, timeout time.Duration) error {
	return nil
}

type constantS3 struct {
	filename    string
	data        []byte
	contentType string
}

var _ s3API = (*constantS3)(nil)

func newConstantS3(t testing.TB) *constantS3 {
	data, err := ioutil.ReadFile(cloudtrailTestFile)
	if err != nil {
		t.Fatal(err)
	}

	return &constantS3{
		filename:    filepath.Base(cloudtrailTestFile),
		data:        data,
		contentType: contentTypeJSON,
	}
}

func (c constantS3) GetObject(ctx context.Context, bucket, key string) (*s3.GetObjectResponse, error) {
	return newS3GetObjectResponse(c.filename, c.data, c.contentType), nil
}

func makeBenchmarkConfig(t testing.TB) config {
	cfg := common.MustNewConfigFrom(`---
queue_url: foo
file_selectors:
-
  regex: '.json.gz$'
  expand_event_list_from_field: Records
`)

	inputConfig := defaultConfig()
	if err := cfg.Unpack(&inputConfig); err != nil {
		t.Fatal(err)
	}
	return inputConfig
}

func benchmarkInput(t *testing.T, maxMessagesInflight int) testing.BenchmarkResult {
	return testing.Benchmark(func(b *testing.B) {
		log := logp.NewLogger(inputName)
		metricRegistry := monitoring.NewRegistry()
		metrics := newInputMetrics(metricRegistry, "test_id")
		sqsAPI := newConstantSQS()
		s3API := newConstantS3(t)
		client := pubtest.NewChanClient(100)
		defer close(client.Channel)
		conf := makeBenchmarkConfig(t)

		s3EventHandlerFactory := newS3ObjectProcessorFactory(log.Named("s3"), metrics, s3API, client, conf.FileSelectors)
		sqsMessageHandler := newSQSS3EventProcessor(log.Named("sqs_s3_event"), metrics, sqsAPI, time.Minute, 5, s3EventHandlerFactory)
		sqsReader := newSQSReader(log.Named("sqs"), metrics, sqsAPI, maxMessagesInflight, sqsMessageHandler)

		go func() {
			for event := range client.Channel {
				// Fake the ACK handling that's not implemented in pubtest.
				event.Private.(*eventACKTracker).ACK()
			}
		}()

		ctx, cancel := context.WithCancel(context.Background())
		b.Cleanup(cancel)

		go func() {
			for metrics.sqsMessagesReceivedTotal.Get() < uint64(b.N) {
				time.Sleep(5 * time.Millisecond)
			}
			cancel()
		}()

		b.ResetTimer()
		start := time.Now()
		if err := sqsReader.Receive(ctx); err != nil {
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatal(err)
			}
		}
		b.StopTimer()
		elapsed := time.Since(start)

		b.ReportMetric(float64(maxMessagesInflight), "max_messages_inflight")
		b.ReportMetric(elapsed.Seconds(), "sec")

		b.ReportMetric(float64(metrics.s3EventsCreatedTotal.Get()), "events")
		b.ReportMetric(float64(metrics.s3EventsCreatedTotal.Get())/elapsed.Seconds(), "events_per_sec")

		b.ReportMetric(float64(metrics.s3BytesProcessedTotal.Get()), "s3_bytes")
		b.ReportMetric(float64(metrics.s3BytesProcessedTotal.Get())/elapsed.Seconds(), "s3_bytes_per_sec")

		b.ReportMetric(float64(metrics.sqsMessagesDeletedTotal.Get()), "sqs_messages")
		b.ReportMetric(float64(metrics.sqsMessagesDeletedTotal.Get())/elapsed.Seconds(), "sqs_messages_per_sec")
	})
}

func TestBenchmarkInput(t *testing.T) {
	logp.TestingSetup(logp.WithLevel(logp.InfoLevel))

	results := []testing.BenchmarkResult{
		benchmarkInput(t, 1),
		benchmarkInput(t, 2),
		benchmarkInput(t, 4),
		benchmarkInput(t, 8),
		benchmarkInput(t, 16),
		benchmarkInput(t, 32),
		benchmarkInput(t, 64),
		benchmarkInput(t, 128),
		benchmarkInput(t, 256),
		benchmarkInput(t, 512),
		benchmarkInput(t, 1024),
	}

	headers := []string{
		"Max Msgs Inflight",
		"Events per sec",
		"S3 Bytes per sec",
		"Time (sec)",
		"CPUs",
	}
	var data [][]string
	for _, r := range results {
		data = append(data, []string{
			fmt.Sprintf("%v", r.Extra["max_messages_inflight"]),
			fmt.Sprintf("%v", r.Extra["events_per_sec"]),
			fmt.Sprintf("%v", humanize.Bytes(uint64(r.Extra["s3_bytes_per_sec"]))),
			fmt.Sprintf("%v", r.Extra["sec"]),
			fmt.Sprintf("%v", runtime.GOMAXPROCS(0)),
		})
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(headers)
	table.AppendBulk(data)
	table.Render()
}
