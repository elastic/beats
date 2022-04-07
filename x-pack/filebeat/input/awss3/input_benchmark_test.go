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

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/dustin/go-humanize"
	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/monitoring"
	pubtest "github.com/elastic/beats/v7/libbeat/publisher/testing"
	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/statestore/storetest"
	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
)

const (
	cloudtrailTestFile  = "testdata/aws-cloudtrail.json.gz"
	totalListingObjects = 10000
)

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

type s3PagerConstant struct {
	objects      []s3.Object
	currentIndex int
}

var _ s3Pager = (*s3PagerConstant)(nil)

func (c *s3PagerConstant) Next(ctx context.Context) bool {
	return c.currentIndex < len(c.objects)
}

func (c *s3PagerConstant) CurrentPage() *s3.ListObjectsOutput {
	ret := &s3.ListObjectsOutput{}
	pageSize := 1000
	if len(c.objects) < c.currentIndex+pageSize {
		pageSize = len(c.objects) - c.currentIndex
	}

	ret.Contents = c.objects[c.currentIndex : c.currentIndex+pageSize]
	c.currentIndex = c.currentIndex + pageSize

	return ret
}

func (c *s3PagerConstant) Err() error {
	if c.currentIndex >= len(c.objects) {
		c.currentIndex = 0
	}
	return nil
}

func newS3PagerConstant() *s3PagerConstant {
	lastModified := time.Now()
	ret := &s3PagerConstant{
		currentIndex: 0,
	}

	for i := 0; i < totalListingObjects; i++ {
		ret.objects = append(ret.objects, s3.Object{
			Key:          aws.String(fmt.Sprintf("key-%d.json.gz", i)),
			ETag:         aws.String(fmt.Sprintf("etag-%d", i)),
			LastModified: aws.Time(lastModified),
		})
	}

	return ret
}

type constantS3 struct {
	filename      string
	data          []byte
	contentType   string
	pagerConstant s3Pager
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

func (c constantS3) ListObjectsPaginator(bucket, prefix string) s3Pager {
	return c.pagerConstant
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

func benchmarkInputSQS(t *testing.T, maxMessagesInflight int) testing.BenchmarkResult {
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
		sqsMessageHandler := newSQSS3EventProcessor(log.Named("sqs_s3_event"), metrics, sqsAPI, nil, time.Minute, 5, s3EventHandlerFactory)
		sqsReader := newSQSReader(log.Named("sqs"), metrics, sqsAPI, maxMessagesInflight, sqsMessageHandler)

		go func() {
			for event := range client.Channel {
				// Fake the ACK handling that's not implemented in pubtest.
				event.Private.(*awscommon.EventACKTracker).ACK()
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

func TestBenchmarkInputSQS(t *testing.T) {
	logp.TestingSetup(logp.WithLevel(logp.InfoLevel))

	results := []testing.BenchmarkResult{
		benchmarkInputSQS(t, 1),
		benchmarkInputSQS(t, 2),
		benchmarkInputSQS(t, 4),
		benchmarkInputSQS(t, 8),
		benchmarkInputSQS(t, 16),
		benchmarkInputSQS(t, 32),
		benchmarkInputSQS(t, 64),
		benchmarkInputSQS(t, 128),
		benchmarkInputSQS(t, 256),
		benchmarkInputSQS(t, 512),
		benchmarkInputSQS(t, 1024),
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

func benchmarkInputS3(t *testing.T, numberOfWorkers int) testing.BenchmarkResult {
	return testing.Benchmark(func(b *testing.B) {
		log := logp.NewLogger(inputName)
		metricRegistry := monitoring.NewRegistry()
		metrics := newInputMetrics(metricRegistry, "test_id")
		s3API := newConstantS3(t)
		s3API.pagerConstant = newS3PagerConstant()
		client := pubtest.NewChanClientWithCallback(100, func(event beat.Event) {
			event.Private.(*awscommon.EventACKTracker).ACK()
		})

		defer close(client.Channel)
		conf := makeBenchmarkConfig(t)

		storeReg := statestore.NewRegistry(storetest.NewMemoryStoreBackend())
		store, err := storeReg.Get("test")
		if err != nil {
			t.Fatalf("Failed to access store: %v", err)
		}

		err = store.Set(awsS3WriteCommitPrefix+"bucket", &commitWriteState{time.Time{}})
		if err != nil {
			t.Fatalf("Failed to reset store: %v", err)
		}

		s3EventHandlerFactory := newS3ObjectProcessorFactory(log.Named("s3"), metrics, s3API, client, conf.FileSelectors)
		s3Poller := newS3Poller(logp.NewLogger(inputName), metrics, s3API, s3EventHandlerFactory, newStates(inputCtx), store, "bucket", "key-", "region", "provider", numberOfWorkers, time.Second)

		ctx, cancel := context.WithCancel(context.Background())
		b.Cleanup(cancel)

		go func() {
			for metrics.s3ObjectsAckedTotal.Get() < totalListingObjects {
				time.Sleep(5 * time.Millisecond)
			}
			cancel()
		}()

		b.ResetTimer()
		start := time.Now()
		if err := s3Poller.Poll(ctx); err != nil {
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatal(err)
			}
		}
		b.StopTimer()
		elapsed := time.Since(start)

		b.ReportMetric(float64(numberOfWorkers), "number_of_workers")
		b.ReportMetric(elapsed.Seconds(), "sec")

		b.ReportMetric(float64(metrics.s3EventsCreatedTotal.Get()), "events")
		b.ReportMetric(float64(metrics.s3EventsCreatedTotal.Get())/elapsed.Seconds(), "events_per_sec")

		b.ReportMetric(float64(metrics.s3BytesProcessedTotal.Get()), "s3_bytes")
		b.ReportMetric(float64(metrics.s3BytesProcessedTotal.Get())/elapsed.Seconds(), "s3_bytes_per_sec")

		b.ReportMetric(float64(metrics.s3ObjectsListedTotal.Get()), "objects_listed")
		b.ReportMetric(float64(metrics.s3ObjectsListedTotal.Get())/elapsed.Seconds(), "objects_listed_per_sec")

		b.ReportMetric(float64(metrics.s3ObjectsProcessedTotal.Get()), "objects_processed")
		b.ReportMetric(float64(metrics.s3ObjectsProcessedTotal.Get())/elapsed.Seconds(), "objects_processed_per_sec")

		b.ReportMetric(float64(metrics.s3ObjectsAckedTotal.Get()), "objects_acked")
		b.ReportMetric(float64(metrics.s3ObjectsAckedTotal.Get())/elapsed.Seconds(), "objects_acked_per_sec")
	})
}

func TestBenchmarkInputS3(t *testing.T) {
	logp.TestingSetup(logp.WithLevel(logp.InfoLevel))

	results := []testing.BenchmarkResult{
		benchmarkInputS3(t, 1),
		benchmarkInputS3(t, 2),
		benchmarkInputS3(t, 4),
		benchmarkInputS3(t, 8),
		benchmarkInputS3(t, 16),
		benchmarkInputS3(t, 32),
		benchmarkInputS3(t, 64),
		benchmarkInputS3(t, 128),
		benchmarkInputS3(t, 256),
		benchmarkInputS3(t, 512),
		benchmarkInputS3(t, 1024),
	}

	headers := []string{
		"Number of workers",
		"Objects listed per sec",
		"Objects processed per sec",
		"Objects acked per sec",
		"Events per sec",
		"S3 Bytes per sec",
		"Time (sec)",
		"CPUs",
	}
	var data [][]string
	for _, r := range results {
		data = append(data, []string{
			fmt.Sprintf("%v", r.Extra["number_of_workers"]),
			fmt.Sprintf("%v", r.Extra["objects_listed_per_sec"]),
			fmt.Sprintf("%v", r.Extra["objects_processed_per_sec"]),
			fmt.Sprintf("%v", r.Extra["objects_acked_per_sec"]),
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
