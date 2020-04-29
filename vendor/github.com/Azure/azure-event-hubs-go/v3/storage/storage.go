// Package storage provides implementations for Checkpointer and Leaser from package eph for persisting leases and
// checkpoints for the Event Processor Host using Azure Storage as a durable store.
package storage

//	MIT License
//
//	Copyright (c) Microsoft Corporation. All rights reserved.
//
//	Permission is hereby granted, free of charge, to any person obtaining a copy
//	of this software and associated documentation files (the "Software"), to deal
//	in the Software without restriction, including without limitation the rights
//	to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
//	copies of the Software, and to permit persons to whom the Software is
//	furnished to do so, subject to the following conditions:
//
//	The above copyright notice and this permission notice shall be included in all
//	copies or substantial portions of the Software.
//
//	THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
//	IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
//	FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
//	AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
//	LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
//	OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
//	SOFTWARE

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/url"
	"sync"
	"time"

	"github.com/Azure/azure-amqp-common-go/v3/uuid"
	"github.com/devigned/tab"

	"github.com/Azure/azure-event-hubs-go/v3"
	"github.com/Azure/azure-event-hubs-go/v3/eph"
	"github.com/Azure/azure-event-hubs-go/v3/persist"

	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/Azure/go-autorest/autorest/azure"
)

type (
	// LeaserCheckpointer implements the eph.LeaserCheckpointer interface for Azure Storage
	LeaserCheckpointer struct {
		// LeasePersistenceInterval is the default period of time which dirty leases will be persisted to Azure Storage
		LeasePersistenceInterval time.Duration
		leases                   map[string]*storageLease
		processor                *eph.EventProcessorHost
		leaseDuration            time.Duration
		credential               Credential
		containerURL             *azblob.ContainerURL
		serviceURL               *azblob.ServiceURL
		containerName            string
		accountName              string
		env                      azure.Environment
		dirtyPartitions          map[string]uuid.UUID
		leasesMu                 sync.Mutex
		done                     func()
	}

	storageLease struct {
		*eph.Lease
		leaser     *LeaserCheckpointer
		Checkpoint *persist.Checkpoint   `json:"checkpoint"`
		State      azblob.LeaseStateType `json:"state"`
		Token      string                `json:"token"`
	}

	// Credential is a wrapper for the Azure Storage azblob.Credential
	Credential interface {
		azblob.Credential
	}

	leaseGetResult struct {
		Lease *storageLease
		Err   error
	}

	dirtyResult struct {
		PartitionID string
		Err         error
	}
)

const (
	defaultLeasePersistenceInterval = 5 * time.Second
)

// NewStorageLeaserCheckpointer builds an Azure Storage Leaser Checkpointer which handles leasing and checkpointing for
// the EventProcessorHost
func NewStorageLeaserCheckpointer(credential Credential, accountName, containerName string, env azure.Environment) (*LeaserCheckpointer, error) {
	storageURL, err := url.Parse("https://" + accountName + ".blob." + env.StorageEndpointSuffix)
	if err != nil {
		return nil, err
	}

	svURL := azblob.NewServiceURL(*storageURL, azblob.NewPipeline(credential, azblob.PipelineOptions{}))
	containerURL := svURL.NewContainerURL(containerName)

	return &LeaserCheckpointer{
		credential:               credential,
		containerName:            containerName,
		accountName:              accountName,
		leaseDuration:            eph.DefaultLeaseDuration,
		env:                      env,
		serviceURL:               &svURL,
		containerURL:             &containerURL,
		leases:                   make(map[string]*storageLease),
		dirtyPartitions:          make(map[string]uuid.UUID),
		LeasePersistenceInterval: defaultLeasePersistenceInterval,
	}, nil
}

// SetEventHostProcessor sets the EventHostProcessor on the instance of the LeaserCheckpointer
func (sl *LeaserCheckpointer) SetEventHostProcessor(eph *eph.EventProcessorHost) {
	sl.processor = eph
	ctx, cancel := context.WithCancel(context.Background())
	go sl.persistLeases(ctx)
	sl.done = cancel
}

// StoreExists returns true if the storage container exists
func (sl *LeaserCheckpointer) StoreExists(ctx context.Context) (bool, error) {
	span, ctx := startConsumerSpanFromContext(ctx, "storage.LeaserCheckpointer.StoreExists")
	defer span.End()

	opts := azblob.ListContainersSegmentOptions{
		Prefix: sl.containerName,
	}
	res, err := sl.serviceURL.ListContainersSegment(ctx, azblob.Marker{}, opts)
	if err != nil {
		return false, err
	}

	for _, container := range res.ContainerItems {
		if container.Name == sl.containerName {
			return true, nil
		}
	}
	return false, nil
}

// EnsureStore creates the container if it does not exist
func (sl *LeaserCheckpointer) EnsureStore(ctx context.Context) error {
	sl.leasesMu.Lock()
	defer sl.leasesMu.Unlock()
	span, ctx := startConsumerSpanFromContext(ctx, "storage.LeaserCheckpointer.EnsureStore")
	defer span.End()

	ok, err := sl.StoreExists(ctx)
	if err != nil {
		return err
	}

	if !ok {
		containerURL := sl.serviceURL.NewContainerURL(sl.containerName)
		_, err := containerURL.Create(ctx, azblob.Metadata{}, azblob.PublicAccessNone)
		if err != nil {
			return err
		}
		sl.containerURL = &containerURL
	}
	return nil
}

// DeleteStore deletes the Azure Storage container
func (sl *LeaserCheckpointer) DeleteStore(ctx context.Context) error {
	sl.leasesMu.Lock()
	defer sl.leasesMu.Unlock()

	span, ctx := startConsumerSpanFromContext(ctx, "storage.LeaserCheckpointer.DeleteStore")
	defer span.End()

	_, err := sl.containerURL.Delete(ctx, azblob.ContainerAccessConditions{})
	return err
}

// GetLeases gets all of the partition leases
func (sl *LeaserCheckpointer) GetLeases(ctx context.Context) ([]eph.LeaseMarker, error) {
	sl.leasesMu.Lock()
	defer sl.leasesMu.Unlock()

	span, ctx := startConsumerSpanFromContext(ctx, "storage.LeaserCheckpointer.GetLeases")
	defer span.End()

	partitionIDs := sl.processor.GetPartitionIDs()
	leaseCh := make(chan leaseGetResult)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for _, partitionID := range partitionIDs {
		go func(pID string) {
			lease, err := sl.getLease(ctx, pID)
			select {
			case <-ctx.Done():
				return
			case leaseCh <- leaseGetResult{Lease: lease, Err: err}:
			}
		}(partitionID)
	}

	leases := make([]eph.LeaseMarker, len(partitionIDs))
	for i := 0; i < len(partitionIDs); i++ {
		select {
		case <-ctx.Done():
			return leases, ctx.Err()
		case result := <-leaseCh:
			if result.Err != nil {
				return nil, result.Err
			}
			leases[i] = result.Lease
		}
	}
	return leases, nil
}

// EnsureLease creates a lease in the container if it doesn't exist
func (sl *LeaserCheckpointer) EnsureLease(ctx context.Context, partitionID string) (eph.LeaseMarker, error) {
	sl.leasesMu.Lock()
	defer sl.leasesMu.Unlock()
	span, ctx := startConsumerSpanFromContext(ctx, "storage.LeaserCheckpointer.EnsureLease")
	defer span.End()

	return sl.createOrGetLease(ctx, partitionID)
}

// DeleteLease deletes a lease in the storage container
func (sl *LeaserCheckpointer) DeleteLease(ctx context.Context, partitionID string) error {
	sl.leasesMu.Lock()
	defer sl.leasesMu.Unlock()

	span, ctx := startConsumerSpanFromContext(ctx, "storage.LeaserCheckpointer.DeleteLease")
	defer span.End()

	_, err := sl.containerURL.NewBlobURL(partitionID).Delete(ctx, azblob.DeleteSnapshotsOptionInclude, azblob.BlobAccessConditions{})
	delete(sl.leases, partitionID)
	return err
}

// AcquireLease acquires the lease to the Azure blob in the container
func (sl *LeaserCheckpointer) AcquireLease(ctx context.Context, partitionID string) (eph.LeaseMarker, bool, error) {
	sl.leasesMu.Lock()
	defer sl.leasesMu.Unlock()

	span, ctx := startConsumerSpanFromContext(ctx, "storage.LeaserCheckpointer.AcquireLease")
	defer span.End()

	blobURL := sl.containerURL.NewBlobURL(partitionID)
	lease, err := sl.getLease(ctx, partitionID)
	if err != nil {
		tab.For(ctx).Error(err)
		return nil, false, nil
	}

	res, err := blobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
	if err != nil {
		tab.For(ctx).Error(err)
		return nil, false, err
	}

	uuidToken, err := uuid.NewV4()
	if err != nil {
		tab.For(ctx).Error(err)
		return nil, false, err
	}

	newToken := uuidToken.String()
	if res.LeaseState() == azblob.LeaseStateLeased {
		// is leased by someone else due to a race to acquire
		_, err := blobURL.ChangeLease(ctx, lease.Token, newToken, azblob.ModifiedAccessConditions{})
		if err != nil {
			tab.For(ctx).Error(err)
			return nil, false, err
		}
	} else {
		_, err = blobURL.AcquireLease(ctx, newToken, int32(sl.leaseDuration.Round(time.Second).Seconds()), azblob.ModifiedAccessConditions{})
		if err != nil {
			tab.For(ctx).Error(err)
			return nil, false, err
		}
	}

	lease.Token = newToken
	lease.Owner = sl.processor.GetName()
	lease.IncrementEpoch()
	err = sl.uploadLease(ctx, lease)
	if err != nil {
		return nil, false, err
	}
	sl.leases[partitionID] = lease
	return lease, true, nil
}

// RenewLease renews the lease to the Azure blob
func (sl *LeaserCheckpointer) RenewLease(ctx context.Context, partitionID string) (eph.LeaseMarker, bool, error) {
	sl.leasesMu.Lock()
	defer sl.leasesMu.Unlock()

	span, ctx := startConsumerSpanFromContext(ctx, "storage.LeaserCheckpointer.RenewLease")
	defer span.End()

	blobURL := sl.containerURL.NewBlobURL(partitionID)
	lease, ok := sl.leases[partitionID]
	if !ok {
		return nil, false, errors.New("lease was not found")
	}

	_, err := blobURL.RenewLease(ctx, lease.Token, azblob.ModifiedAccessConditions{})
	if err != nil {
		tab.For(ctx).Error(err)
		return nil, false, err
	}
	return lease, true, nil
}

// ReleaseLease releases the lease to the blob in Azure storage
func (sl *LeaserCheckpointer) ReleaseLease(ctx context.Context, partitionID string) (bool, error) {
	sl.leasesMu.Lock()
	defer sl.leasesMu.Unlock()

	span, ctx := startConsumerSpanFromContext(ctx, "storage.LeaserCheckpointer.ReleaseLease")
	defer span.End()

	blobURL := sl.containerURL.NewBlobURL(partitionID)
	lease, ok := sl.leases[partitionID]
	if !ok {
		return false, errors.New("lease was not found")
	}

	_, err := blobURL.ReleaseLease(ctx, lease.Token, azblob.ModifiedAccessConditions{})
	if err != nil {
		tab.For(ctx).Error(err)
		return false, err
	}
	delete(sl.leases, partitionID)
	delete(sl.dirtyPartitions, partitionID)
	return true, nil
}

// UpdateLease renews and uploads the latest lease to the blob store
func (sl *LeaserCheckpointer) UpdateLease(ctx context.Context, partitionID string) (eph.LeaseMarker, bool, error) {
	sl.leasesMu.Lock()
	defer sl.leasesMu.Unlock()

	span, ctx := startConsumerSpanFromContext(ctx, "storage.LeaserCheckpointer.UpdateLease")
	defer span.End()

	return sl.updateLease(ctx, partitionID)
}

func (sl *LeaserCheckpointer) updateLease(ctx context.Context, partitionID string) (eph.LeaseMarker, bool, error) {
	span, ctx := startConsumerSpanFromContext(ctx, "storage.LeaserCheckpointer.updateLease")
	defer span.End()

	blobURL := sl.containerURL.NewBlobURL(partitionID)
	lease, ok := sl.leases[partitionID]
	if !ok {
		return nil, false, errors.New("lease was not found")
	}

	_, err := blobURL.RenewLease(ctx, lease.Token, azblob.ModifiedAccessConditions{})
	if err != nil {
		tab.For(ctx).Error(err)
		return nil, false, err
	}

	if !ok {
		return nil, false, errors.New("could not renew lease when updating lease")
	}

	err = sl.uploadLease(ctx, lease)
	if err != nil {
		tab.For(ctx).Error(err)
		return nil, false, err
	}

	return lease, true, nil
}

// GetCheckpoint returns the latest checkpoint for the partitionID.
func (sl *LeaserCheckpointer) GetCheckpoint(ctx context.Context, partitionID string) (persist.Checkpoint, bool) {
	sl.leasesMu.Lock()
	defer sl.leasesMu.Unlock()

	span, _ := startConsumerSpanFromContext(ctx, "storage.LeaserCheckpointer.GetCheckpoint")
	defer span.End()

	lease, ok := sl.leases[partitionID]
	if ok {
		return *lease.Checkpoint, ok
	}
	return persist.NewCheckpointFromStartOfStream(), ok
}

// EnsureCheckpoint ensures a checkpoint exists for the lease
func (sl *LeaserCheckpointer) EnsureCheckpoint(ctx context.Context, partitionID string) (persist.Checkpoint, error) {
	sl.leasesMu.Lock()
	defer sl.leasesMu.Unlock()

	span, _ := startConsumerSpanFromContext(ctx, "storage.LeaserCheckpointer.EnsureCheckpoint")
	defer span.End()

	lease, ok := sl.leases[partitionID]
	if ok {
		if lease.Checkpoint == nil {
			checkpoint := persist.NewCheckpointFromStartOfStream()
			lease.Checkpoint = &checkpoint
		}
		return *lease.Checkpoint, nil
	}
	return persist.NewCheckpointFromStartOfStream(), nil
}

// UpdateCheckpoint will attempt to write the checkpoint to Azure Storage
func (sl *LeaserCheckpointer) UpdateCheckpoint(ctx context.Context, partitionID string, checkpoint persist.Checkpoint) error {
	sl.leasesMu.Lock()
	defer sl.leasesMu.Unlock()

	span, _ := startConsumerSpanFromContext(ctx, "storage.LeaserCheckpointer.UpdateCheckpoint")
	defer span.End()

	lease, ok := sl.leases[partitionID]
	if !ok {
		return errors.New("lease for partition isn't owned by this EventProcessorHost")
	}

	lease.Checkpoint = &checkpoint
	dirtyPartitionID, err := uuid.NewV4()
	if err != nil {
		return err
	}
	sl.dirtyPartitions[partitionID] = dirtyPartitionID
	return nil
}

// DeleteCheckpoint will attempt to delete the checkpoint from Azure Storage
func (sl *LeaserCheckpointer) DeleteCheckpoint(ctx context.Context, partitionID string) error {
	sl.leasesMu.Lock()
	defer sl.leasesMu.Unlock()

	span, ctx := startConsumerSpanFromContext(ctx, "storage.LeaserCheckpointer.DeleteCheckpoint")
	defer span.End()

	lease, ok := sl.leases[partitionID]
	if !ok {
		return errors.New("lease for partition isn't owned by this EventProcessorHost")
	}

	checkpoint := persist.NewCheckpointFromStartOfStream()
	lease.Checkpoint = &checkpoint
	updatedLease, ok, err := sl.updateLease(ctx, lease.PartitionID)
	if err != nil {
		return err
	}

	if !ok {
		return errors.New("checkpoint update was not successful")
	}
	sl.leases[partitionID] = updatedLease.(*storageLease)
	return nil

}

// Close will stop the leaser / checkpointer from persisting dirty leases & checkpoints to storage
func (sl *LeaserCheckpointer) Close() error {
	if sl.done != nil {
		sl.done()
	}
	return nil
}

func (sl *LeaserCheckpointer) persistLeases(ctx context.Context) {
	span, ctx := startConsumerSpanFromContext(ctx, "storage.LeaserCheckpointer.persistLeases")
	defer span.End()
	<-time.After(5 * time.Second) // initial delay

	for {
		select {
		case <-ctx.Done():
			return
		default:
			err := sl.persistDirtyPartitions(ctx)
			if err != nil {
				tab.For(ctx).Error(err)
			}
			<-time.After(sl.LeasePersistenceInterval)
		}
	}
}

func (sl *LeaserCheckpointer) persistDirtyPartitions(ctx context.Context) error {
	sl.leasesMu.Lock()
	defer sl.leasesMu.Unlock()

	span, ctx := startConsumerSpanFromContext(ctx, "storage.LeaserCheckpointer.persistDirtyPartitions")
	defer span.End()

	resCh := make(chan dirtyResult)

	// gather all of the dirty partition ids
	pids := make([]string, len(sl.dirtyPartitions))
	count := 0
	for pid := range sl.dirtyPartitions {
		pids[count] = pid
		count++
	}

	// send each partition data to storage concurrently capturing errors
	for _, pid := range pids {
		go func(id string) {
			err := sl.persistLease(ctx, id)
			select {
			case <-ctx.Done():
				return
			case resCh <- dirtyResult{PartitionID: id, Err: err}:
			}
		}(pid)
	}

	// collect all of the results as they complete
	// don't return until each partition has had a chance to persist or the context has expired
	var lastErr error
	for i := 0; i < len(pids); i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case res := <-resCh:
			if res.Err != nil {
				lastErr = res.Err
			}
			delete(sl.dirtyPartitions, res.PartitionID)
		}
	}

	return lastErr
}

func (sl *LeaserCheckpointer) persistLease(ctx context.Context, partitionID string) error {
	span, _ := startConsumerSpanFromContext(ctx, "storage.LeaserCheckpointer.persistLease")
	defer span.End()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	_, ok, err := sl.updateLease(ctx, partitionID)

	if err != nil {
		return err
	}

	if !ok {
		return errors.New("unable to update dirty lease -- this may mean there will be reprocessing")
	}
	return nil
}

func (sl *LeaserCheckpointer) uploadLease(ctx context.Context, lease *storageLease) error {
	span, ctx := startConsumerSpanFromContext(ctx, "storage.LeaserCheckpointer.uploadLease")
	defer span.End()

	blobURL := sl.containerURL.NewBlobURL(lease.PartitionID)
	jsonLease, err := json.Marshal(lease)
	if err != nil {
		return err
	}
	reader := bytes.NewReader(jsonLease)
	_, err = blobURL.ToBlockBlobURL().Upload(ctx, reader, azblob.BlobHTTPHeaders{}, azblob.Metadata{}, azblob.BlobAccessConditions{
		LeaseAccessConditions: azblob.LeaseAccessConditions{
			LeaseID: lease.Token,
		},
	})

	return err
}

func (sl *LeaserCheckpointer) createOrGetLease(ctx context.Context, partitionID string) (*storageLease, error) {
	span, ctx := startConsumerSpanFromContext(ctx, "storage.LeaserCheckpointer.createOrGetLease")
	defer span.End()

	lease := &storageLease{
		Lease: &eph.Lease{
			PartitionID: partitionID,
		},
	}
	blobURL := sl.containerURL.NewBlobURL(partitionID)
	jsonLease, err := json.Marshal(lease)
	if err != nil {
		return nil, err
	}
	reader := bytes.NewReader(jsonLease)
	res, err := blobURL.ToBlockBlobURL().Upload(ctx, reader, azblob.BlobHTTPHeaders{}, azblob.Metadata{}, azblob.BlobAccessConditions{
		ModifiedAccessConditions: azblob.ModifiedAccessConditions{
			IfNoneMatch: "*",
		},
	})

	if err != nil {
		return nil, err
	}

	if res.StatusCode() == 404 {
		return sl.getLease(ctx, partitionID)
	}
	return lease, err
}

func (sl *LeaserCheckpointer) getLease(ctx context.Context, partitionID string) (*storageLease, error) {
	span, ctx := startConsumerSpanFromContext(ctx, "storage.LeaserCheckpointer.getLease")
	defer span.End()

	blobURL := sl.containerURL.NewBlobURL(partitionID)
	res, err := blobURL.Download(ctx, 0, azblob.CountToEnd, azblob.BlobAccessConditions{}, false)
	if err != nil {
		return nil, err
	}
	return sl.leaseFromResponse(res)
}

func (sl *LeaserCheckpointer) leaseFromResponse(res *azblob.DownloadResponse) (*storageLease, error) {
	b, err := ioutil.ReadAll(res.Response().Body)
	if err != nil {
		return nil, err
	}

	var lease storageLease
	if err := json.Unmarshal(b, &lease); err != nil {
		return nil, err
	}
	lease.leaser = sl
	lease.State = res.LeaseState()
	return &lease, nil
}

// IsExpired checks to see if the blob is not still leased
func (s *storageLease) IsExpired(ctx context.Context) bool {
	span, ctx := startConsumerSpanFromContext(ctx, "storage.storageLease.IsExpired")
	defer span.End()

	lease, err := s.leaser.getLease(ctx, s.PartitionID)
	if err != nil {
		return false
	}
	return lease.State != azblob.LeaseStateLeased
}

func (s *storageLease) String() string {
	bits, err := json.Marshal(s)
	if err != nil {
		return ""
	}
	return string(bits)
}

func startConsumerSpanFromContext(ctx context.Context, operationName string) (tab.Spanner, context.Context) {
	ctx, span := tab.StartSpan(ctx, operationName)
	eventhub.ApplyComponentInfo(span)
	span.AddAttributes(
		tab.StringAttribute("span.kind", "client"),
		tab.StringAttribute("eh.eventprocessorhost.kind", "azure.storage"),
	)
	return span, ctx
}
