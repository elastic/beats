// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azure_blob

import (
	"context"
	"net/url"
	"sync"
	"time"

	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"go.uber.org/multierr"

	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/go-concert/timed"
)

type commitWriteState struct {
	time.Time
}

type blobObjectInfo struct {
	container_name string
	blob_name      string
	etag           string
	lastModified   time.Time
	listingID      string
}

type blobObjectPayload struct {
	blobObjectHandler blobObjectHandler
	blobObjectInfo    blobObjectInfo
}

type blobPoller struct {
	numberOfWorkers  int
	container        string
	listPrefix       string
	blobPollInterval time.Duration
	// workerSem            *awscommon.Sem
	log                  *logp.Logger
	metrics              *inputMetrics
	blob                 blobAPI
	blobObjectHandler    azureBlobProcessorFactory
	states               *states
	store                *statestore.Store
	workersListingMap    *sync.Map
	workersProcessingMap *sync.Map
	marker               azblob.Marker
}

func newBlobPoller(log *logp.Logger,
	metrics *inputMetrics,
	blob blobAPI,
	blobObjectHandlerFactory azureBlobProcessorFactory,
	states *states,
	store *statestore.Store,
	container string,
	listPrefix string,
	numberOfWorkers int,
	blobPollInterval time.Duration) *blobPoller {
	if metrics == nil {
		metrics = newInputMetrics(monitoring.NewRegistry(), "")
	}
	return &blobPoller{
		numberOfWorkers:  numberOfWorkers,
		container:        container,
		listPrefix:       listPrefix,
		blobPollInterval: blobPollInterval,
		// workerSem:          awscommon.NewSem(numberOfWorkers),
		log:                  log,
		metrics:              metrics,
		blob:                 blob,
		blobObjectHandler:    blobObjectHandlerFactory,
		states:               states,
		store:                store,
		workersListingMap:    new(sync.Map),
		workersProcessingMap: new(sync.Map),
		marker:               azblob.Marker{},
	}
}

func (p *blobPoller) handlePurgingLock(info blobObjectInfo, isStored bool) {
	id := info.container_name + info.blob_name
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

func (p *blobPoller) ProcessObject(s3ObjectPayloadChan <-chan *blobObjectPayload) error {
	var errs []error

	for s3ObjectPayload := range s3ObjectPayloadChan {
		// Process S3 object (download, parse, create events).
		err := s3ObjectPayload.blobObjectHandler.ProcessBlobObject()

		// Wait for all events to be ACKed before proceeding.
		s3ObjectPayload.blobObjectHandler.Wait()

		info := s3ObjectPayload.blobObjectInfo

		if err != nil {
			blob_name := s3ObjectPayload.blobObjectInfo.blob_name
			container_name := s3ObjectPayload.blobObjectInfo.container_name
			errs = append(errs, errors.Wrapf(err,
				"failed processing Azure Blob event for object name %q in container %q",
				blob_name, container_name))

			p.handlePurgingLock(info, false)
			continue
		}

		p.handlePurgingLock(info, true)

		// Metrics
		p.metrics.s3ObjectsAckedTotal.Inc()
	}

	return multierr.Combine(errs...)
}

func (p *blobPoller) GetS3Objects(ctx context.Context, blobObjectPayloadChan chan<- *blobObjectPayload) {
	defer close(blobObjectPayloadChan)

	for p.marker.NotDone() {
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

		page, err := p.blob.ListObjectsPaginator(ctx, p.listPrefix, p.marker)
		if err != nil {
			p.log.Errorf("Azure Blob GetObjects failed: %w", err)
			return
		}
		p.marker = page.NextMarker

		totProcessableObjects := 0
		totListedObjects := len(page.Segment.BlobItems)
		blobObjectPayloadChanByPage := make(chan *blobObjectPayload, totListedObjects)

		// Metrics
		p.metrics.s3ObjectsListedTotal.Add(uint64(totListedObjects))
		for _, object := range page.Segment.BlobItems {
			// Unescape s3 key name. For example, convert "%3D" back to "=".
			filename, err := url.QueryUnescape(object.Name)
			if err != nil {
				p.log.Errorw("Error when unescaping object key, skipping.", "error", err, "s3_object", object.Name)
				continue
			}

			state := newState(p.container, filename, string(object.Properties.Etag), object.Properties.LastModified)
			if p.states.MustSkip(state, p.store) {
				p.log.Debugw("skipping state.", "state", state)
				continue
			}

			p.states.Update(state, "")

			acker := NewEventACKTracker(ctx)

			blobProcessor := p.blobObjectHandler.Create(ctx, p.log, acker, p.container, object.Name)
			if blobProcessor == nil {
				continue
			}

			totProcessableObjects++

			blobObjectPayloadChanByPage <- &blobObjectPayload{
				blobObjectHandler: blobProcessor,
				blobObjectInfo: blobObjectInfo{
					container_name: p.container,
					blob_name:      filename,
					etag:           string(object.Properties.Etag),
					lastModified:   object.Properties.LastModified,
					listingID:      listingID.String(),
				},
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

		close(blobObjectPayloadChanByPage)
		for blobObjectPayload := range blobObjectPayloadChanByPage {
			blobObjectPayloadChan <- blobObjectPayload
		}
	}

	// if err := paginator.Err(); err != nil {
	// 	p.log.Warnw("Error when paginating listing.", "error", err)
	// }

	return
}

func (p *blobPoller) Purge() {
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

func (p *blobPoller) Poll(ctx context.Context) error {
	// This loop tries to keep the workers busy as much as possible while
	// honoring the number in config opposed to a simpler loop that does one
	//  listing, sequentially processes every object and then does another listing
	workerWg := new(sync.WaitGroup)
	for ctx.Err() == nil {
		// Determine how many S3 workers are available.
		// workers, err := p.workerSem.AcquireContext(p.numberOfWorkers, ctx)
		// if err != nil {
		// 	break
		// }

		// if workers == 0 {
		// 	continue
		// }

		s3ObjectPayloadChan := make(chan *blobObjectPayload)

		workerWg.Add(1)
		go func() {
			defer func() {
				workerWg.Done()
			}()

			p.GetS3Objects(ctx, s3ObjectPayloadChan)
			p.Purge()
		}()

		workerWg.Add(p.numberOfWorkers)
		for i := 0; i < p.numberOfWorkers; i++ {
			go func() {
				defer func() {
					workerWg.Done()
					// p.workerSem.Release(1)
				}()
				if err := p.ProcessObject(s3ObjectPayloadChan); err != nil {
					p.log.Warnw("Failed processing Blob listing.", "error", err)
				}
			}()
		}

		timed.Wait(ctx, p.blobPollInterval)

	}

	// Wait for all workers to finish.
	workerWg.Wait()

	if errors.Is(ctx.Err(), context.Canceled) {
		// A canceled context is a normal shutdown.
		return nil
	}
	return ctx.Err()
}
