// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws/ratelimit"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/backoff"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/go-concert/timed"
)

// pollingDiscoveryV2 implements the S3 polling discovery loop for the V2 input.
// It lists objects via ListObjectsV2, filters them, dispatches work to a pool
// of goroutines, and persists state after events are ACKed.
type pollingDiscoveryV2 struct {
	s3        s3API
	processor *objectProcessorV2
	registry  *stateRegistryV2
	metrics   *inputMetrics
	log       *logp.Logger
	status    status.StatusReporter

	bucketARN      string
	bucketName     string
	listPrefix     string
	listInterval   time.Duration
	numWorkers     int
	region         string
	provider       string
	strategy       pollingStrategy
	filterProvider *filterProvider
}

// pollingDiscoveryV2Config holds the parameters for creating V2 polling discovery.
type pollingDiscoveryV2Config struct {
	S3              s3API
	Processor       *objectProcessorV2
	Registry        *stateRegistryV2
	Metrics         *inputMetrics
	Log             *logp.Logger
	Status          status.StatusReporter
	BucketARN       string
	ListPrefix      string
	ListInterval    time.Duration
	NumWorkers      int
	Region          string
	Provider        string
	Lexicographical bool
	FilterProvider  *filterProvider
}

func newPollingDiscoveryV2(cfg pollingDiscoveryV2Config) *pollingDiscoveryV2 {
	return &pollingDiscoveryV2{
		s3:             cfg.S3,
		processor:      cfg.Processor,
		registry:       cfg.Registry,
		metrics:        cfg.Metrics,
		log:            cfg.Log,
		status:         cfg.Status,
		bucketARN:      cfg.BucketARN,
		bucketName:     getBucketNameFromARN(cfg.BucketARN),
		listPrefix:     cfg.ListPrefix,
		listInterval:   cfg.ListInterval,
		numWorkers:     cfg.NumWorkers,
		region:         cfg.Region,
		provider:       cfg.Provider,
		strategy:       newPollingStrategy(cfg.Lexicographical, cfg.Log),
		filterProvider: cfg.FilterProvider,
	}
}

// Run executes the poll loop until ctx is cancelled.
func (p *pollingDiscoveryV2) Run(ctx context.Context, pipeline beat.Pipeline) {
	p.status.UpdateStatus(status.Running, "Input is running")
	for ctx.Err() == nil {
		start := time.Now()
		p.poll(ctx, pipeline)
		elapsed := time.Since(start)
		p.metrics.s3PollingRunTime.Update(elapsed.Nanoseconds())
		p.metrics.s3PollingRunTimeTotal.Add(uint64(elapsed.Nanoseconds())) //nolint:gosec // elapsed is non-negative
		_ = timed.Wait(ctx, p.listInterval)
	}
}

func (p *pollingDiscoveryV2) poll(ctx context.Context, pipeline beat.Pipeline) {
	pollCtx, pollCancel := context.WithCancel(ctx)
	defer pollCancel()

	var wg sync.WaitGroup
	workChan := make(chan state)

	for range p.numWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.worker(pollCtx, pipeline, workChan, pollCancel)
		}()
	}

	ids, _, ok := p.listObjects(pollCtx, workChan)
	wg.Wait()

	if !ok {
		p.log.Warn("skipping state cleanup: listing ended with errors")
		return
	}
	if err := p.registry.CleanUp(ids); err != nil {
		p.log.Errorf("state cleanup failed: %v", err)
		p.status.UpdateStatus(status.Degraded, fmt.Sprintf("State cleanup failure: %s", err))
	}
}

func (p *pollingDiscoveryV2) worker(ctx context.Context, pipeline beat.Pipeline, work <-chan state, cancel context.CancelFunc) {
	acks := newAWSACKHandler()
	client, err := createPipelineClient(pipeline, acks)
	if err != nil {
		p.log.Errorf("failed to create pipeline client: %v", err)
		p.status.UpdateStatus(status.Degraded, fmt.Sprintf("Pipeline client setup failed: %s", err))
		cancel()
		return
	}
	defer func() {
		acks.Close()
		client.Close()
	}()

	rateLimitWaiter := backoff.NewEqualJitterBackoff(ctx.Done(), 1, 120)
	for st := range work {
		if err := p.registry.MarkObjectInFlight(st.Key); err != nil {
			p.log.Errorf("failed to mark object in-flight: %v", err)
		}

		evt := p.stateToEvent(st)
		publishCount := 0
		n, procErr := p.processor.ProcessObject(ctx, p.log, evt, func(e beat.Event) {
			p.metrics.s3EventsCreatedTotal.Inc()
			client.Publish(e)
			publishCount++
		})
		_ = n
		p.metrics.s3EventsPerObject.Update(int64(publishCount))

		if errors.Is(procErr, errS3DownloadFailed) {
			p.status.UpdateStatus(status.Degraded, fmt.Sprintf("S3 download failure: %s/%s: %s", st.Bucket, st.Key, procErr))
			if err := p.registry.UnmarkObjectInFlight(st.Key); err != nil {
				p.log.Errorf("failed to unmark object in-flight: %v", err)
			}
			rateLimitWaiter.Wait()
			continue
		}
		rateLimitWaiter.Reset()

		if procErr != nil {
			p.log.Errorf("failed processing S3 object %q in bucket %q: %v", st.Key, st.Bucket, procErr)
			p.status.UpdateStatus(status.Degraded, fmt.Sprintf("S3 processing failure: %s/%s: %s", st.Bucket, st.Key, procErr))
			st.Failed = true
		} else {
			st.Stored = true
		}

		acks.Add(publishCount, func() {
			if err := p.registry.AddState(st); err != nil {
				p.log.Errorf("saving object state: %v", err)
				p.status.UpdateStatus(status.Degraded, fmt.Sprintf("State save failure: %s", err))
			} else {
				p.status.UpdateStatus(status.Running, "Input is running")
			}
			p.metrics.s3ObjectsAckedTotal.Inc()
			if st.Stored {
				if err := p.processor.Finalize(context.WithoutCancel(ctx), p.s3, evt); err != nil {
					p.log.Errorf("S3 finalization failed for %q: %v", st.Key, err)
					p.status.UpdateStatus(status.Degraded, fmt.Sprintf("S3 finalization failed: %s", err))
				}
			}
		})
	}
}

// listObjects paginates ListObjectsV2 and sends unprocessed objects to workChan.
// Returns tracked state IDs for cleanup and whether listing completed without
// hitting the circuit breaker.
func (p *pollingDiscoveryV2) listObjects(ctx context.Context, workChan chan<- state) (ids []string, numListed int, ok bool) {
	defer close(workChan)

	isStateValid := p.filterProvider.getApplierFunc()
	errorBackoff := backoff.NewEqualJitterBackoff(ctx.Done(), 1, 120)
	circuitBreaker := 0

	startAfterKey := p.registry.GetStartAfterKey()
	paginator := p.s3.ListObjectsPaginator(p.bucketName, p.listPrefix, startAfterKey)

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			p.log.Warnw("Error paginating S3 listing.", "error", err)
			p.status.UpdateStatus(status.Degraded, fmt.Sprintf("S3 pagination error: %s", err))
			if !errors.As(err, &ratelimit.QuotaExceededError{}) {
				circuitBreaker++
				if circuitBreaker >= readerLoopMaxCircuitBreaker {
					p.log.Warnf("%d consecutive pagination errors, breaking circuit.", circuitBreaker)
					return nil, numListed, false
				}
			}
			errorBackoff.Wait()
			continue
		}
		circuitBreaker = 0
		errorBackoff.Reset()

		numListed += len(page.Contents)
		p.metrics.s3ObjectsListedTotal.Add(uint64(len(page.Contents)))

		for _, obj := range page.Contents {
			st := newState(p.bucketName, *obj.Key, *obj.ETag, *obj.LastModified)

			if p.strategy.ShouldSkipObject(st, isStateValid) {
				continue
			}

			id := p.strategy.GetStateID(st)
			ids = append(ids, id)

			if p.registry.IsProcessed(id) {
				continue
			}

			select {
			case workChan <- st:
				p.metrics.s3ObjectsProcessedTotal.Inc()
			case <-ctx.Done():
				return ids, numListed, false
			}
		}
	}

	p.metrics.s3ObjectsListedPerRun.Update(int64(numListed))
	return ids, numListed, true
}

func (p *pollingDiscoveryV2) stateToEvent(st state) s3EventV2 {
	var evt s3EventV2
	evt.AWSRegion = p.region
	evt.Provider = p.provider
	evt.S3.Bucket.Name = st.Bucket
	evt.S3.Bucket.ARN = p.bucketARN
	evt.S3.Object.Key = st.Key
	evt.S3.Object.LastModified = st.LastModified
	return evt
}
