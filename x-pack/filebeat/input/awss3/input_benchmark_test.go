// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/common/atomic"
	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/statestore/storetest"

	"github.com/elastic/beats/v7/libbeat/beat"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3Types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	sqsTypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/dustin/go-humanize"
	"github.com/olekukonko/tablewriter"

	pubtest "github.com/elastic/beats/v7/libbeat/publisher/testing"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

const (
	cloudtrailTestFileGz          = "testdata/aws-cloudtrail.json.gz"
	cloudtrailTestFile            = "testdata/aws-cloudtrail.json"
	totalListingObjects           = 10000
	totalListingObjectsForInputS3 = totalListingObjects / 5
)

type constantSQS struct {
	s3API        *constantS3
	receiveCallN *atomic.Uint64
	msgs         [][]sqsTypes.Message
}

var _ sqsAPI = (*constantSQS)(nil)

func newConstantSQS(t testing.TB, maxMessages int, totalSqsMessages uint64, s3API *constantS3) *constantSQS {
	customRand := rand.New(rand.NewSource(1))

	var s3ObjN int
	var generatedMessages uint64

	msgs := make([][]sqsTypes.Message, 0)

	c := &constantSQS{s3API: s3API, receiveCallN: atomic.NewUint64(0)}
	for {
		if generatedMessages == totalSqsMessages {
			break
		}

		currentMessages := uint64(customRand.Intn(maxMessages)) + 1
		if totalSqsMessages < generatedMessages+currentMessages {
			currentMessages = totalSqsMessages - generatedMessages
		}

		generatedMessages += currentMessages

		currentMsgs := make([]sqsTypes.Message, 0, currentMessages)
		for ; currentMessages > 0; currentMessages-- {
			totS3Events := customRand.Intn(9) + 1
			s3Events := make([]s3EventV2, 0, totS3Events)
			for ; totS3Events > 0; totS3Events-- {
				totRecordsInS3Events := customRand.Intn(160) + 1
				recordsInS3Events := make([]map[string]any, 0, totRecordsInS3Events)
				for ; totRecordsInS3Events > 0; totRecordsInS3Events-- {
					recordN := customRand.Intn(len(c.s3API.records)-1) + 1
					recordsInS3Events = append(recordsInS3Events, c.s3API.records[recordN])
				}

				s3ObjKey := fmt.Sprintf("%d", s3ObjN)
				s3Events = append(s3Events, newS3Event(s3ObjKey))

				data, err := json.Marshal(struct{ Records []map[string]any }{Records: recordsInS3Events})
				if err != nil {
					t.Fatal(err)
				}

				c.s3API.objects = append(c.s3API.objects, data)
				s3ObjN++
			}

			currentMsgs = append(currentMsgs, newSQSMessage(s3Events...))
		}

		msgs = append(msgs, currentMsgs)
	}

	c.msgs = msgs

	return c
}

func (c *constantSQS) ReceiveMessage(ctx context.Context, maxMessages int) ([]sqsTypes.Message, error) {
	receiveCallN := c.receiveCallN.Add(1)
	var msgs []sqsTypes.Message
	if receiveCallN <= uint64(len(c.msgs)) {
		msgs = c.msgs[receiveCallN-1]
	}

	return msgs, nil
}

func (*constantSQS) DeleteMessage(ctx context.Context, msg *sqsTypes.Message) error {
	return nil
}

func (*constantSQS) ChangeMessageVisibility(ctx context.Context, msg *sqsTypes.Message, timeout time.Duration) error {
	return nil
}

func (c *constantSQS) GetQueueAttributes(ctx context.Context, attr []sqsTypes.QueueAttributeName) (map[string]string, error) {
	return map[string]string{}, nil
}

type s3PagerConstant struct {
	mutex        *sync.Mutex
	objects      []s3Types.Object
	currentIndex int
}

var _ s3Pager = (*s3PagerConstant)(nil)

func (c *s3PagerConstant) HasMorePages() bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.currentIndex < len(c.objects)
}

func (c *s3PagerConstant) NextPage(ctx context.Context, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	if !c.HasMorePages() {
		return nil, errors.New("no more pages")
	}
	c.mutex.Lock()
	defer c.mutex.Unlock()

	ret := &s3.ListObjectsV2Output{}
	pageSize := 1000
	if len(c.objects) < c.currentIndex+pageSize {
		pageSize = len(c.objects) - c.currentIndex
	}

	ret.Contents = c.objects[c.currentIndex : c.currentIndex+pageSize]
	c.currentIndex = c.currentIndex + pageSize

	return ret, nil
}

func newS3PagerConstant(listPrefix string) *s3PagerConstant {
	lastModified := time.Now()
	ret := &s3PagerConstant{
		mutex:        new(sync.Mutex),
		currentIndex: 0,
	}

	for i := 0; i < totalListingObjectsForInputS3; i++ {
		ret.objects = append(ret.objects, s3Types.Object{
			Key:          aws.String(fmt.Sprintf("%s-%d.json.gz", listPrefix, i)),
			ETag:         aws.String(fmt.Sprintf("etag-%s-%d", listPrefix, i)),
			LastModified: aws.Time(lastModified),
		})
	}

	return ret
}

type constantS3 struct {
	filename      string
	data          []byte
	records       []map[string]any
	objects       [][]byte
	contentType   string
	pagerConstant s3Pager
}

var _ s3API = (*constantS3)(nil)

func newConstantS3(t testing.TB) *constantS3 {
	dataGz, err := os.ReadFile(cloudtrailTestFileGz)
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(cloudtrailTestFile)
	if err != nil {
		t.Fatal(err)
	}

	var records struct{ Records []map[string]any }
	err = json.Unmarshal(data, &records)
	if err != nil {
		t.Fatal(err)
	}

	return &constantS3{
		filename:    filepath.Base(cloudtrailTestFileGz),
		data:        dataGz,
		records:     records.Records,
		contentType: contentTypeJSON,
		objects:     make([][]byte, 0),
	}
}

func (c *constantS3) GetObject(ctx context.Context, bucket, key string) (*s3.GetObjectOutput, error) {
	// direct listing uses gz content
	if strings.HasSuffix(key, ".json.gz") {
		return newS3GetObjectResponse(c.filename, c.data, c.contentType), nil
	}

	// this is s3 sqs notification
	keyN, err := strconv.Atoi(key)
	if err != nil {
		return nil, err
	}

	return newS3GetObjectResponse(key, c.objects[keyN], c.contentType), nil
}

func (c *constantS3) CopyObject(ctx context.Context, from_bucket, to_bucket, from_key, to_key string) (*s3.CopyObjectOutput, error) {
	return nil, nil
}

func (c *constantS3) DeleteObject(ctx context.Context, bucket, key string) (*s3.DeleteObjectOutput, error) {
	return nil, nil
}

func (c *constantS3) ListObjectsPaginator(bucket, prefix string) s3Pager {
	return c.pagerConstant
}

var _ beat.Pipeline = (*fakePipeline)(nil)

func newFakePipeline() *fakePipeline {
	fp := &fakePipeline{
		mutex:         new(sync.Mutex),
		flush:         time.NewTicker(10 * time.Second),
		pendingEvents: atomic.NewUint64(0),
		clients:       make([]*ackClient, 0),
	}

	go func() {
		for {
			<-fp.flush.C
			fp.mutex.Lock()
			fp.ackEvents()
			fp.mutex.Unlock()
		}
	}()

	return fp
}

// fakePipeline returns new ackClients.
type fakePipeline struct {
	flush         *time.Ticker
	mutex         *sync.Mutex
	pendingEvents *atomic.Uint64
	clients       []*ackClient
}

func (fp *fakePipeline) ackEvents() {
	for _, client := range fp.clients {
		for _, acker := range client.ackers {
			if acker.FullyAcked() {
				continue
			}

			addedEvents := 0
			for acker.EventsToBeAcked.Load() > 0 && uint64(addedEvents) < acker.EventsToBeAcked.Load() {
				addedEvents++
				fp.pendingEvents.Dec()
				client.eventListener.AddEvent(beat.Event{Private: acker}, true)
			}

			if addedEvents > 0 {
				client.eventListener.ACKEvents(addedEvents)
			}
		}
	}
}

func (fp *fakePipeline) ConnectWith(clientConfig beat.ClientConfig) (beat.Client, error) {
	fp.mutex.Lock()
	client := &ackClient{fp: fp, ackers: make(map[uint64]*EventACKTracker), eventListener: NewEventACKHandler()}
	fp.clients = append(fp.clients, client)
	fp.mutex.Unlock()
	return client, nil
}

func (fp *fakePipeline) Connect() (beat.Client, error) {
	panic("Connect() is not implemented.")
}

var _ beat.Client = (*ackClient)(nil)

// ackClient is a fake beat.Client that ACKs the published messages.
type ackClient struct {
	fp            *fakePipeline
	ackers        map[uint64]*EventACKTracker
	eventListener beat.EventListener
}

func (c *ackClient) Close() error { return nil }

func (c *ackClient) Publish(event beat.Event) {
	c.fp.mutex.Lock()
	c.fp.pendingEvents.Inc()
	acker := event.Private.(*EventACKTracker)
	c.ackers[acker.ID] = acker
	if c.fp.pendingEvents.Load() > 3200 {
		c.fp.ackEvents()
	}

	c.fp.mutex.Unlock()
}

func (c *ackClient) PublishAll(event []beat.Event) {
	for _, e := range event {
		c.Publish(e)
	}
}

func makeBenchmarkConfig(t testing.TB) config {
	cfg := conf.MustNewConfigFrom(`---
queue_url: foo
file_selectors:
-
  regex: '.json.gz$'
  expand_event_list_from_field: Records
-
  regex: '^[\d]+$'
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
		metrics := newInputMetrics("test_id", metricRegistry, maxMessagesInflight)
		totalSqsMessages := uint64(math.Ceil(float64(maxMessagesInflight) * 1.1))
		s3API := newConstantS3(t)
		sqsAPI := newConstantSQS(t, maxMessagesInflight, totalSqsMessages, s3API)

		logSqs := log.Named("sqs")
		pipeline := newFakePipeline()

		conf := makeBenchmarkConfig(t)

		s3EventHandlerFactory := newS3ObjectProcessorFactory(log.Named("s3"), metrics, s3API, conf.FileSelectors, backupConfig{}, maxMessagesInflight)
		sqsMessageHandler := newSQSS3EventProcessor(log.Named("sqs_s3_event"), metrics, sqsAPI, nil, time.Minute, 5, s3EventHandlerFactory)
		sqsReader := newSQSReader(logSqs, metrics, sqsAPI, maxMessagesInflight, sqsMessageHandler, pipeline)

		ctx, cancel := context.WithCancel(context.Background())
		b.Cleanup(cancel)

		cancelChan := make(chan time.Duration)
		start := time.Now()
		go func() {
			for {
				if metrics.sqsMessagesProcessedTotal.Get() == totalSqsMessages {
					break
				}
				time.Sleep(5 * time.Millisecond)
			}

			cancel()
			cancelChan <- time.Since(start)
		}()

		b.ResetTimer()
		if err := sqsReader.Receive(ctx); err != nil {
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatal(err)
			}
		}
		b.StopTimer()
		elapsed := time.Since(start)
		cancelElapsed := <-cancelChan
		deltaElapsedCanceled := elapsed.Seconds() - cancelElapsed.Seconds()

		b.ReportMetric(float64(maxMessagesInflight), "max_messages_inflight")
		b.ReportMetric(elapsed.Seconds(), "sec")
		b.ReportMetric(cancelElapsed.Seconds(), "cancel_sec")
		b.ReportMetric(deltaElapsedCanceled, "delta_sec_from_cancel")
		b.ReportMetric(100.*(deltaElapsedCanceled/elapsed.Seconds()), "flushing_time_percentage")

		b.ReportMetric(float64(metrics.s3EventsCreatedTotal.Get()), "events")
		b.ReportMetric(float64(metrics.s3EventsCreatedTotal.Get())/cancelElapsed.Seconds(), "events_per_sec")

		b.ReportMetric(float64(metrics.s3BytesProcessedTotal.Get()), "s3_bytes")
		b.ReportMetric(float64(metrics.s3BytesProcessedTotal.Get())/cancelElapsed.Seconds(), "s3_bytes_per_sec")

		b.ReportMetric(float64(metrics.s3ObjectsRequestedTotal.Get()), "s3_objects")
		b.ReportMetric(float64(metrics.s3ObjectsRequestedTotal.Get())/cancelElapsed.Seconds(), "s3_objects_per_sec")

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
		"Events total",
		"Events per sec",
		"S3 Bytes total",
		"S3 Bytes per sec",
		"S3 Objects total",
		"S3 Objects per sec",
		"SQS Messages total",
		"SQS Messages per sec",
		"Full Time (sec)",
		"Processing Time (sec)",
		"Flushing Time (sec)",
		"Flushing time (%)",
		"CPUs",
	}
	data := make([][]string, 0)
	for _, r := range results {
		data = append(data, []string{
			fmt.Sprintf("%v", r.Extra["max_messages_inflight"]),
			fmt.Sprintf("%v", r.Extra["events"]),
			fmt.Sprintf("%v", r.Extra["events_per_sec"]),
			fmt.Sprintf("%v", humanize.Bytes(uint64(r.Extra["s3_bytes"]))),
			fmt.Sprintf("%v", humanize.Bytes(uint64(r.Extra["s3_bytes_per_sec"]))),
			fmt.Sprintf("%v", r.Extra["s3_objects"]),
			fmt.Sprintf("%v", r.Extra["s3_objects_per_sec"]),
			fmt.Sprintf("%v", r.Extra["sqs_messages"]),
			fmt.Sprintf("%v", r.Extra["sqs_messages_per_sec"]),
			fmt.Sprintf("%v", r.Extra["sec"]),
			fmt.Sprintf("%v", r.Extra["cancel_sec"]),
			fmt.Sprintf("%v", r.Extra["delta_sec_from_cancel"]),
			fmt.Sprintf("%v", humanize.FormatFloat("#,##", r.Extra["flushing_time_percentage"])),
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
		log.Infof("benchmark with %d number of workers", numberOfWorkers)

		metricRegistry := monitoring.NewRegistry()
		metrics := newInputMetrics("test_id", metricRegistry, numberOfWorkers)

		client := pubtest.NewChanClientWithCallback(100, func(event beat.Event) {
			go func(acker *EventACKTracker) {
				// 63 is the total number of events in a single S3 object
				acker.MarkS3FromListingProcessedWithData(63)
			}(event.Private.(*EventACKTracker))

			event.Private.(*EventACKTracker).ACK()
		})

		defer func() {
			_ = client.Close()
		}()

		conf := makeBenchmarkConfig(t)

		b.ResetTimer()
		start := time.Now()
		ctx, cancel := context.WithCancel(context.Background())
		b.Cleanup(cancel)

		go func() {
			for metrics.s3ObjectsAckedTotal.Get() < totalListingObjects {
				time.Sleep(5 * time.Millisecond)
			}
			cancel()
		}()

		errChan := make(chan error)
		wg := new(sync.WaitGroup)
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(i int, wg *sync.WaitGroup) {
				defer wg.Done()
				listPrefix := fmt.Sprintf("list_prefix_%d", i)
				s3API := newConstantS3(t)
				s3API.pagerConstant = newS3PagerConstant(listPrefix)
				storeReg := statestore.NewRegistry(storetest.NewMemoryStoreBackend())
				store, err := storeReg.Get("test")
				if err != nil {
					errChan <- fmt.Errorf("failed to access store: %w", err)
					return
				}

				err = store.Set(awsS3WriteCommitPrefix+"bucket"+listPrefix, &commitWriteState{time.Time{}})
				if err != nil {
					errChan <- err
					return
				}

				s3EventHandlerFactory := newS3ObjectProcessorFactory(log.Named("s3"), metrics, s3API, conf.FileSelectors, backupConfig{}, numberOfWorkers)
				s3Poller := newS3Poller(logp.NewLogger(inputName), metrics, s3API, client, s3EventHandlerFactory, newStates(inputCtx), store, "bucket", listPrefix, "region", "provider", numberOfWorkers, time.Second)

				if err := s3Poller.Poll(ctx); err != nil {
					if !errors.Is(err, context.DeadlineExceeded) {
						errChan <- err
					}
				}
			}(i, wg)
		}

		wg.Wait()
		select {
		case err := <-errChan:
			if err != nil {
				t.Fatal(err)
			}
		default:

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
		"Objects listed total",
		"Objects listed per sec",
		"Objects processed total",
		"Objects processed per sec",
		"Objects acked total",
		"Objects acked per sec",
		"Events total",
		"Events per sec",
		"S3 Bytes total",
		"S3 Bytes per sec",
		"Time (sec)",
		"CPUs",
	}
	data := make([][]string, 0)
	for _, r := range results {
		data = append(data, []string{
			fmt.Sprintf("%v", r.Extra["number_of_workers"]),
			fmt.Sprintf("%v", r.Extra["objects_listed"]),
			fmt.Sprintf("%v", r.Extra["objects_listed_per_sec"]),
			fmt.Sprintf("%v", r.Extra["objects_processed"]),
			fmt.Sprintf("%v", r.Extra["objects_processed_per_sec"]),
			fmt.Sprintf("%v", r.Extra["objects_acked"]),
			fmt.Sprintf("%v", r.Extra["objects_acked_per_sec"]),
			fmt.Sprintf("%v", r.Extra["events"]),
			fmt.Sprintf("%v", r.Extra["events_per_sec"]),
			fmt.Sprintf("%v", humanize.Bytes(uint64(r.Extra["s3_bytes"]))),
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
