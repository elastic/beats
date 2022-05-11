// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"context"
	"net/url"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"go.uber.org/multierr"

	"github.com/elastic/beats/v7/libbeat/monitoring"
	"github.com/elastic/beats/v7/libbeat/statestore"
	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/go-concert/timed"
)

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
	s3ObjectHandler      s3ObjectHandlerFactory
	states               *states
	store                *statestore.Store
	workersListingMap    *sync.Map
	workersProcessingMap *sync.Map
}

func newS3Poller(log *logp.Logger,
	metrics *inputMetrics,
	s3 s3API,
	s3ObjectHandler s3ObjectHandlerFactory,
	states *states,
	store *statestore.Store,
	bucket string,
	listPrefix string,
	awsRegion string,
	provider string,
	numberOfWorkers int,
	bucketPollInterval time.Duration) *s3Poller {
	if metrics == nil {
		metrics = newInputMetrics(monitoring.NewRegistry(), "")
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
		s3ObjectHandler:      s3ObjectHandler,
		states:               states,
		store:                store,
		workersListingMap:    new(sync.Map),
		workersProcessingMap: new(sync.Map),
	}
}

func (p *s3Poller) handlePurgingLock(info s3ObjectInfo, isStored bool) {
	id := info.name + info.key
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
			errs = append(errs, errors.Wrapf(err,
				"failed processing S3 event for object key %q in bucket %q",
				event.S3.Object.Key, event.S3.Bucket.Name))

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

	paginator := p.s3.ListObjectsPaginator(bucketName, p.listPrefix)
	for paginator.Next(ctx) {
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

		page := paginator.CurrentPage()

		totProcessableObjects := 0
		totListedObjects := len(page.Contents)
		s3ObjectPayloadChanByPage := make(chan *s3ObjectPayload, totListedObjects)

		// Metrics
		p.metrics.s3ObjectsListedTotal.Add(uint64(totListedObjects))
		for _, object := range page.Contents {
			// Unescape s3 key name. For example, convert "%3D" back to "=".
			filename, err := url.QueryUnescape(*object.Key)
			if err != nil {
				p.log.Errorw("Error when unescaping object key, skipping.", "error", err, "s3_object", *object.Key)
				continue
			}

			state := newState(bucketName, filename, *object.ETag, *object.LastModified)
			if p.states.MustSkip(state, p.store) {
				p.log.Debugw("skipping state.", "state", state)
				continue
			}

			p.states.Update(state, "")

			event := s3EventV2{}
			event.AWSRegion = p.region
			event.Provider = p.provider
			event.S3.Bucket.Name = bucketName
			event.S3.Bucket.ARN = p.bucket
			event.S3.Object.Key = filename

			acker := awscommon.NewEventACKTracker(ctx)

			s3Processor := p.s3ObjectHandler.Create(ctx, p.log, acker, event)
			if s3Processor == nil {
				continue
			}

			totProcessableObjects++

			s3ObjectPayloadChanByPage <- &s3ObjectPayload{
				s3ObjectHandler: s3Processor,
				s3ObjectInfo: s3ObjectInfo{
					name:         bucketName,
					key:          filename,
					etag:         *object.ETag,
					lastModified: *object.LastModified,
					listingID:    listingID.String(),
				},
				s3ObjectEvent: event,
			}
		}

		if totProcessableObjects == 0 {
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

	if err := paginator.Err(); err != nil {
		p.log.Warnw("Error when paginating listing.", "error", err)
	}

	return
}

func (p *s3Poller) Purge() {
	listingIDs := p.states.GetListingIDs()
	for _, listingID := range listingIDs {
		// we lock here in order to process the purge only after
		// full listing page is ACKed by all the workers
		lock, loaded := p.workersListingMap.Load(listingID)
		if !loaded {
			// purge calls can overlap, GetListingIDs can return
			// an outdated snapshot with listing already purged
			p.states.DeleteListing(listingID)
			continue
		}

		lock.(*sync.Mutex).Lock()

		keys := map[string]struct{}{}
		latestStoredTimeByBucket := make(map[string]time.Time, 0)

		for _, state := range p.states.GetStatesByListingID(listingID) {
			// it is not stored, keep
			if !state.Stored {
				continue
			}

			var latestStoredTime time.Time
			keys[state.ID] = struct{}{}
			latestStoredTime, ok := latestStoredTimeByBucket[state.Bucket]
			if !ok {
				var commitWriteState commitWriteState
				err := p.store.Get(awsS3WriteCommitPrefix+state.Bucket, &commitWriteState)
				if err == nil {
					// we have no entry in the map and we have no entry in the store
					// set zero time
					latestStoredTime = time.Time{}
				} else {
					latestStoredTime = commitWriteState.Time
				}
			}

			if state.LastModified.After(latestStoredTime) {
				latestStoredTimeByBucket[state.Bucket] = state.LastModified
			}

		}

		for key := range keys {
			p.states.Delete(key)
		}

		if err := p.states.writeStates(p.store); err != nil {
			p.log.Errorw("Failed to write states to the registry", "error", err)
		}

		for bucket, latestStoredTime := range latestStoredTimeByBucket {
			if err := p.store.Set(awsS3WriteCommitPrefix+bucket, commitWriteState{latestStoredTime}); err != nil {
				p.log.Errorw("Failed to write commit time to the registry", "error", err)
			}
		}

		// purge is done, we can unlock and clean
		lock.(*sync.Mutex).Unlock()
		p.workersListingMap.Delete(listingID)
		p.states.DeleteListing(listingID)
	}

	return
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
			p.Purge()
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

		timed.Wait(ctx, p.bucketPollInterval)

	}

	// Wait for all workers to finish.
	workerWg.Wait()

	if errors.Is(ctx.Err(), context.Canceled) {
		// A canceled context is a normal shutdown.
		return nil
	}
	return ctx.Err()
}
