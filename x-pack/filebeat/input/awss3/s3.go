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

	"github.com/gofrs/uuid"
	"go.uber.org/multierr"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/statestore"
	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/go-concert/timed"
)

const maxCircuitBreaker = 5

type commitWriteState struct {
	time.Time
}

type s3ObjectInfo struct {
	name         string
	key          string
	etag         string
	lastModified time.Time
	listingID    string
}

type s3ObjectPayload struct {
	s3ObjectHandler s3ObjectHandler
	s3ObjectInfo    s3ObjectInfo
	s3ObjectEvent   s3EventV2
}

type s3Poller struct {
	numberOfWorkers      int
	bucket               string
	listPrefix           string
	region               string
	provider             string
	bucketPollInterval   time.Duration
	workerSem            *awscommon.Sem
	s3                   s3API
	log                  *logp.Logger
	metrics              *inputMetrics
	client               beat.Client
	s3ObjectHandler      s3ObjectHandlerFactory
	states               *states
	store                *statestore.Store
	workersListingMap    *sync.Map
	workersProcessingMap *sync.Map
}

func newS3Poller(log *logp.Logger,
	metrics *inputMetrics,
	s3 s3API,
	client beat.Client,
	s3ObjectHandler s3ObjectHandlerFactory,
	states *states,
	store *statestore.Store,
	bucket string,
	listPrefix string,
	awsRegion string,
	provider string,
	numberOfWorkers int,
	bucketPollInterval time.Duration,
) *s3Poller {
	if metrics == nil {
		// Metrics are optional. Initialize a stub.
		metrics = newInputMetrics("", nil, 0)
	}
	return &s3Poller{
		numberOfWorkers:      numberOfWorkers,
		bucket:               bucket,
		listPrefix:           listPrefix,
		region:               awsRegion,
		provider:             provider,
		bucketPollInterval:   bucketPollInterval,
		workerSem:            awscommon.NewSem(numberOfWorkers),
		s3:                   s3,
		log:                  log,
		metrics:              metrics,
		client:               client,
		s3ObjectHandler:      s3ObjectHandler,
		states:               states,
		store:                store,
		workersListingMap:    new(sync.Map),
		workersProcessingMap: new(sync.Map),
	}
}

func (p *s3Poller) handlePurgingLock(info s3ObjectInfo, isStored bool) {
	id := stateID(info.name, info.key, info.etag, info.lastModified)
	previousState := p.states.FindPreviousByID(id)
	if !previousState.IsEmpty() {
		if isStored {
			previousState.MarkAsStored()
		} else {
			previousState.MarkAsError()
		}

		p.states.Update(previousState, info.listingID)
	}

	// Manage locks for purging.
	if p.states.IsListingFullyStored(info.listingID) {
		// locked on processing we unlock when all the object were ACKed
		lock, _ := p.workersListingMap.Load(info.listingID)
		lock.(*sync.Mutex).Unlock()
	}
}

func (p *s3Poller) createS3ObjectProcessor(ctx context.Context, state state) (s3ObjectHandler, s3EventV2) {
	event := s3EventV2{}
	event.AWSRegion = p.region
	event.Provider = p.provider
	event.S3.Bucket.Name = state.Bucket
	event.S3.Bucket.ARN = p.bucket
	event.S3.Object.Key = state.Key

	acker := awscommon.NewEventACKTracker(ctx)

	return p.s3ObjectHandler.Create(ctx, p.log, p.client, acker, event), event
}

func (p *s3Poller) ProcessObject(s3ObjectPayloadChan <-chan *s3ObjectPayload) error {
	var errs []error

	for s3ObjectPayload := range s3ObjectPayloadChan {
		// Process S3 object (download, parse, create events).
		err := s3ObjectPayload.s3ObjectHandler.ProcessS3Object()

		// Wait for all events to be ACKed before proceeding.
		s3ObjectPayload.s3ObjectHandler.Wait()

		info := s3ObjectPayload.s3ObjectInfo

		if err != nil {
			event := s3ObjectPayload.s3ObjectEvent
			errs = append(errs,
				fmt.Errorf(
					fmt.Sprintf("failed processing S3 event for object key %q in bucket %q: %%w",
						event.S3.Object.Key, event.S3.Bucket.Name),
					err))

			p.handlePurgingLock(info, false)
			continue
		}

		p.handlePurgingLock(info, true)

		// Metrics
		p.metrics.s3ObjectsAckedTotal.Inc()
	}

	return multierr.Combine(errs...)
}

func (p *s3Poller) GetS3Objects(ctx context.Context, s3ObjectPayloadChan chan<- *s3ObjectPayload) {
	defer close(s3ObjectPayloadChan)

	bucketName := getBucketNameFromARN(p.bucket)

	circuitBreaker := 0
	paginator := p.s3.ListObjectsPaginator(bucketName, p.listPrefix)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			if !paginator.HasMorePages() {
				break
			}

			p.log.Warnw("Error when paginating listing.", "error", err)
			circuitBreaker++
			if circuitBreaker >= maxCircuitBreaker {
				p.log.Warnw(fmt.Sprintf("%d consecutive error when paginating listing, breaking the circuit.", circuitBreaker), "error", err)
				break
			}
			continue
		}

		listingID, err := uuid.NewV4()
		if err != nil {
			p.log.Warnw("Error generating UUID for listing page.", "error", err)
			continue
		}

		// lock for the listing page and state in workersListingMap
		// this map is shared with the storedOp and will be unlocked there
		lock := new(sync.Mutex)
		lock.Lock()
		p.workersListingMap.Store(listingID.String(), lock)

		totProcessableObjects := 0
		totListedObjects := len(page.Contents)
		s3ObjectPayloadChanByPage := make(chan *s3ObjectPayload, totListedObjects)

		// Metrics
		p.metrics.s3ObjectsListedTotal.Add(uint64(totListedObjects))
		for _, object := range page.Contents {
			state := newState(bucketName, *object.Key, *object.ETag, p.listPrefix, *object.LastModified)
			if p.states.MustSkip(state, p.store) {
				p.log.Debugw("skipping state.", "state", state)
				continue
			}

			// we have no previous state or the previous state
			// is not stored: refresh the state
			previousState := p.states.FindPrevious(state)
			if previousState.IsEmpty() || !previousState.IsProcessed() {
				p.states.Update(state, "")
			}

			s3Processor, event := p.createS3ObjectProcessor(ctx, state)
			if s3Processor == nil {
				p.log.Debugw("empty s3 processor.", "state", state)
				continue
			}

			totProcessableObjects++

			s3ObjectPayloadChanByPage <- &s3ObjectPayload{
				s3ObjectHandler: s3Processor,
				s3ObjectInfo: s3ObjectInfo{
					name:         bucketName,
					key:          *object.Key,
					etag:         *object.ETag,
					lastModified: *object.LastModified,
					listingID:    listingID.String(),
				},
				s3ObjectEvent: event,
			}
		}

		if totProcessableObjects == 0 {
			p.log.Debugw("0 processable objects on bucket pagination.", "bucket", p.bucket, "listPrefix", p.listPrefix, "listingID", listingID)
			// nothing to be ACKed, unlock here
			p.states.DeleteListing(listingID.String())
			lock.Unlock()
		} else {
			listingInfo := &listingInfo{totObjects: totProcessableObjects}
			p.states.AddListing(listingID.String(), listingInfo)

			// Metrics
			p.metrics.s3ObjectsProcessedTotal.Add(uint64(totProcessableObjects))
		}

		close(s3ObjectPayloadChanByPage)
		for s3ObjectPayload := range s3ObjectPayloadChanByPage {
			s3ObjectPayloadChan <- s3ObjectPayload
		}
	}
}

func (p *s3Poller) Purge(ctx context.Context) {
	listingIDs := p.states.GetListingIDs()
	p.log.Debugw("purging listing.", "listingIDs", listingIDs)
	for _, listingID := range listingIDs {
		// we lock here in order to process the purge only after
		// full listing page is ACKed by all the workers
		lock, loaded := p.workersListingMap.Load(listingID)
		if !loaded {
			// purge calls can overlap, GetListingIDs can return
			// an outdated snapshot with listing already purged
			p.states.DeleteListing(listingID)
			p.log.Debugw("deleting already purged listing from states.", "listingID", listingID)
			continue
		}

		lock.(*sync.Mutex).Lock()

		states := map[string]*state{}
		latestStoredTimeByBucketAndListPrefix := make(map[string]time.Time, 0)

		listingStates := p.states.GetStatesByListingID(listingID)
		for i, state := range listingStates {
			// it is not stored, keep
			if !state.IsProcessed() {
				p.log.Debugw("state not stored or with error, skip purge", "state", state)
				continue
			}

			var latestStoredTime time.Time
			states[state.ID] = &listingStates[i]
			latestStoredTime, ok := latestStoredTimeByBucketAndListPrefix[state.Bucket+state.ListPrefix]
			if !ok {
				var commitWriteState commitWriteState
				err := p.store.Get(awsS3WriteCommitPrefix+state.Bucket+state.ListPrefix, &commitWriteState)
				if err == nil {
					// we have no entry in the map, and we have no entry in the store
					// set zero time
					latestStoredTime = time.Time{}
					p.log.Debugw("last stored time is zero time", "bucket", state.Bucket, "listPrefix", state.ListPrefix)
				} else {
					latestStoredTime = commitWriteState.Time
					p.log.Debugw("last stored time is commitWriteState", "commitWriteState", commitWriteState, "bucket", state.Bucket, "listPrefix", state.ListPrefix)
				}
			} else {
				p.log.Debugw("last stored time from memory", "latestStoredTime", latestStoredTime, "bucket", state.Bucket, "listPrefix", state.ListPrefix)
			}

			if state.LastModified.After(latestStoredTime) {
				p.log.Debugw("last stored time updated", "state.LastModified", state.LastModified, "bucket", state.Bucket, "listPrefix", state.ListPrefix)
				latestStoredTimeByBucketAndListPrefix[state.Bucket+state.ListPrefix] = state.LastModified
			}
		}

		for key := range states {
			p.states.Delete(key)
		}

		if err := p.states.writeStates(p.store); err != nil {
			p.log.Errorw("Failed to write states to the registry", "error", err)
		}

		for bucketAndListPrefix, latestStoredTime := range latestStoredTimeByBucketAndListPrefix {
			if err := p.store.Set(awsS3WriteCommitPrefix+bucketAndListPrefix, commitWriteState{latestStoredTime}); err != nil {
				p.log.Errorw("Failed to write commit time to the registry", "error", err)
			}
		}

		// purge is done, we can unlock and clean
		lock.(*sync.Mutex).Unlock()
		p.workersListingMap.Delete(listingID)
		p.states.DeleteListing(listingID)

		// Listing is removed from all states, we can finalize now
		for _, state := range states {
			processor, _ := p.createS3ObjectProcessor(ctx, *state)
			if err := processor.FinalizeS3Object(); err != nil {
				p.log.Errorw("Failed to finalize S3 object", "key", state.Key, "error", err)
			}
		}
	}
}

func (p *s3Poller) Poll(ctx context.Context) error {
	// This loop tries to keep the workers busy as much as possible while
	// honoring the number in config opposed to a simpler loop that does one
	//  listing, sequentially processes every object and then does another listing
	workerWg := new(sync.WaitGroup)
	for ctx.Err() == nil {
		// Determine how many S3 workers are available.
		workers, err := p.workerSem.AcquireContext(p.numberOfWorkers, ctx)
		if err != nil {
			break
		}

		if workers == 0 {
			continue
		}

		s3ObjectPayloadChan := make(chan *s3ObjectPayload)

		workerWg.Add(1)
		go func() {
			defer func() {
				workerWg.Done()
			}()

			p.GetS3Objects(ctx, s3ObjectPayloadChan)
			p.Purge(ctx)
		}()

		workerWg.Add(workers)
		for i := 0; i < workers; i++ {
			go func() {
				defer func() {
					workerWg.Done()
					p.workerSem.Release(1)
				}()
				if err := p.ProcessObject(s3ObjectPayloadChan); err != nil {
					p.log.Warnw("Failed processing S3 listing.", "error", err)
				}
			}()
		}

		err = timed.Wait(ctx, p.bucketPollInterval)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				// A canceled context is a normal shutdown.
				return nil
			}

			return err
		}
	}

	// Wait for all workers to finish.
	workerWg.Wait()

	if errors.Is(ctx.Err(), context.Canceled) {
		// A canceled context is a normal shutdown.
		return nil
	}
	return ctx.Err()
}
