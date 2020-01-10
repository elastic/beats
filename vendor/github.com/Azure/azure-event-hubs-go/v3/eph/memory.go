package eph

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
	"context"
	"errors"
	"sync"
	"time"

	"github.com/Azure/azure-amqp-common-go/v3/uuid"
	"github.com/devigned/tab"

	"github.com/Azure/azure-event-hubs-go/v3/persist"
)

type (
	memoryLeaserCheckpointer struct {
		store         *sharedStore
		processor     *EventProcessorHost
		leaseDuration time.Duration
		memMu         sync.Mutex
		leases        map[string]*memoryLease
	}

	memoryLease struct {
		Lease
		expirationTime time.Time
		Token          string
		Checkpoint     *persist.Checkpoint
		leaser         *memoryLeaserCheckpointer
	}

	sharedStore struct {
		leases  map[string]*storeLease
		storeMu sync.Mutex
	}

	storeLease struct {
		token      string
		expiration time.Time
		ml         *memoryLease
	}
)

func newMemoryLease(partitionID string) *memoryLease {
	lease := new(memoryLease)
	lease.PartitionID = partitionID
	return lease
}

func (s *sharedStore) exists() bool {
	s.storeMu.Lock()
	defer s.storeMu.Unlock()

	return s.leases != nil
}

func (s *sharedStore) ensure() bool {
	s.storeMu.Lock()
	defer s.storeMu.Unlock()

	if s.leases == nil {
		s.leases = make(map[string]*storeLease)
	}
	return true
}

func (s *sharedStore) getLease(partitionID string) memoryLease {
	s.storeMu.Lock()
	defer s.storeMu.Unlock()

	return *s.leases[partitionID].ml
}

func (s *sharedStore) deleteLease(partitionID string) {
	s.storeMu.Lock()
	defer s.storeMu.Unlock()

	delete(s.leases, partitionID)
}

func (s *sharedStore) createOrGetLease(partitionID string) memoryLease {
	s.storeMu.Lock()
	defer s.storeMu.Unlock()

	if _, ok := s.leases[partitionID]; !ok {
		s.leases[partitionID] = new(storeLease)
	}

	l := s.leases[partitionID]
	if l.ml != nil {
		return *l.ml
	}
	l.ml = newMemoryLease(partitionID)
	return *l.ml
}

func (s *sharedStore) changeLease(partitionID, newToken, oldToken string, duration time.Duration) bool {
	s.storeMu.Lock()
	defer s.storeMu.Unlock()

	if l, ok := s.leases[partitionID]; ok && l.token == oldToken {
		l.token = newToken
		l.expiration = time.Now().Add(duration)
		return true
	}
	return false
}

func (s *sharedStore) releaseLease(partitionID, token string) bool {
	s.storeMu.Lock()
	defer s.storeMu.Unlock()

	if l, ok := s.leases[partitionID]; ok && l.token == token {
		l.token = ""
		l.expiration = time.Now().Add(-1 * time.Second)
		return true
	}
	return false
}

func (s *sharedStore) renewLease(partitionID, token string, duration time.Duration) bool {
	s.storeMu.Lock()
	defer s.storeMu.Unlock()

	if l, ok := s.leases[partitionID]; ok && l.token == token {
		l.expiration = time.Now().Add(duration)
		return true
	}
	return false
}

func (s *sharedStore) acquireLease(partitionID, newToken string, duration time.Duration) bool {
	s.storeMu.Lock()
	defer s.storeMu.Unlock()

	if l, ok := s.leases[partitionID]; ok && (time.Now().After(l.expiration) || l.token == "") {
		l.token = newToken
		l.expiration = time.Now().Add(duration)
		return true
	}
	return false
}

func (s *sharedStore) storeLease(partitionID, token string, ml memoryLease) bool {
	s.storeMu.Lock()
	defer s.storeMu.Unlock()

	if l, ok := s.leases[partitionID]; ok && l.token == token {
		l.ml = &ml
		return true
	}
	return false
}

func (s *sharedStore) isLeased(partitionID string) bool {
	s.storeMu.Lock()
	defer s.storeMu.Unlock()

	if l, ok := s.leases[partitionID]; ok {
		if time.Now().After(l.expiration) || l.token == "" {
			return false
		}
		return true
	}
	return false
}

// IsNotOwnedOrExpired indicates that the lease has expired and does not owned by a processor
func (l *memoryLease) isNotOwnedOrExpired(ctx context.Context) bool {
	return l.IsExpired(ctx) || l.Owner == ""
}

// IsExpired indicates that the lease has expired and is no longer valid
func (l *memoryLease) IsExpired(_ context.Context) bool {
	return !l.leaser.store.isLeased(l.PartitionID)
}

func (l *memoryLease) expireAfter(d time.Duration) {
	l.expirationTime = time.Now().Add(d)
}

func newMemoryLeaserCheckpointer(leaseDuration time.Duration, store *sharedStore) *memoryLeaserCheckpointer {
	return &memoryLeaserCheckpointer{
		leaseDuration: leaseDuration,
		leases:        make(map[string]*memoryLease),
		store:         store,
	}
}

func (ml *memoryLeaserCheckpointer) SetEventHostProcessor(eph *EventProcessorHost) {
	ml.processor = eph
}

func (ml *memoryLeaserCheckpointer) StoreExists(ctx context.Context) (bool, error) {
	span, _ := startConsumerSpanFromContext(ctx, "eph.memoryLeaserCheckpointer.StoreExists")
	defer span.End()

	return ml.store.exists(), nil
}

func (ml *memoryLeaserCheckpointer) EnsureStore(ctx context.Context) error {
	span, _ := startConsumerSpanFromContext(ctx, "eph.memoryLeaserCheckpointer.EnsureStore")
	defer span.End()

	ml.store.ensure()
	return nil
}

func (ml *memoryLeaserCheckpointer) DeleteStore(ctx context.Context) error {
	span, _ := startConsumerSpanFromContext(ctx, "eph.memoryLeaserCheckpointer.DeleteStore")
	defer span.End()

	return ml.EnsureStore(ctx)
}

func (ml *memoryLeaserCheckpointer) GetLeases(ctx context.Context) ([]LeaseMarker, error) {
	ml.memMu.Lock()
	defer ml.memMu.Unlock()

	span, _ := startConsumerSpanFromContext(ctx, "eph.memoryLeaserCheckpointer.GetLeases")
	defer span.End()

	partitionIDs := ml.processor.GetPartitionIDs()
	leases := make([]LeaseMarker, len(partitionIDs))
	for idx, partitionID := range partitionIDs {
		lease := ml.store.getLease(partitionID)
		lease.leaser = ml
		leases[idx] = &lease
	}
	return leases, nil
}

func (ml *memoryLeaserCheckpointer) EnsureLease(ctx context.Context, partitionID string) (LeaseMarker, error) {
	ml.memMu.Lock()
	defer ml.memMu.Unlock()

	span, _ := startConsumerSpanFromContext(ctx, "eph.memoryLeaserCheckpointer.EnsureLease")
	defer span.End()

	l := ml.store.createOrGetLease(partitionID)
	l.leaser = ml
	return &l, nil
}

func (ml *memoryLeaserCheckpointer) DeleteLease(ctx context.Context, partitionID string) error {
	ml.memMu.Lock()
	defer ml.memMu.Unlock()

	span, _ := startConsumerSpanFromContext(ctx, "eph.memoryLeaserCheckpointer.DeleteLease")
	defer span.End()

	ml.store.deleteLease(partitionID)
	return nil
}

func (ml *memoryLeaserCheckpointer) AcquireLease(ctx context.Context, partitionID string) (LeaseMarker, bool, error) {
	ml.memMu.Lock()
	defer ml.memMu.Unlock()

	span, ctx := startConsumerSpanFromContext(ctx, "eph.memoryLeaserCheckpointer.AcquireLease")
	defer span.End()

	lease := ml.store.getLease(partitionID)
	lease.leaser = ml
	uuidToken, err := uuid.NewV4()
	if err != nil {
		tab.For(ctx).Error(err)
		return nil, false, err
	}

	newToken := uuidToken.String()
	if ml.store.isLeased(partitionID) {
		// is leased by someone else due to a race to acquire
		if !ml.store.changeLease(partitionID, newToken, lease.Token, ml.leaseDuration) {
			return nil, false, errors.New("failed to change lease")
		}
	} else {
		if !ml.store.acquireLease(partitionID, newToken, ml.leaseDuration) {
			return nil, false, errors.New("failed to acquire lease")
		}
	}

	lease.Token = newToken
	lease.Owner = ml.processor.GetName()
	lease.IncrementEpoch()
	if !ml.store.storeLease(partitionID, newToken, lease) {
		return nil, false, errors.New("failed to store lease after acquiring or changing")
	}
	ml.leases[partitionID] = &lease
	return &lease, true, nil
}

func (ml *memoryLeaserCheckpointer) RenewLease(ctx context.Context, partitionID string) (LeaseMarker, bool, error) {
	ml.memMu.Lock()
	defer ml.memMu.Unlock()

	span, _ := startConsumerSpanFromContext(ctx, "eph.memoryLeaserCheckpointer.RenewLease")
	defer span.End()

	lease, ok := ml.leases[partitionID]
	if !ok {
		return nil, false, errors.New("lease was not found")
	}

	if !ml.store.renewLease(partitionID, lease.Token, ml.leaseDuration) {
		return nil, false, errors.New("unable to renew lease")
	}
	return lease, true, nil
}

func (ml *memoryLeaserCheckpointer) ReleaseLease(ctx context.Context, partitionID string) (bool, error) {
	ml.memMu.Lock()
	defer ml.memMu.Unlock()

	span, _ := startConsumerSpanFromContext(ctx, "eph.memoryLeaserCheckpointer.ReleaseLease")
	defer span.End()

	lease, ok := ml.leases[partitionID]
	if !ok {
		return false, errors.New("lease was not found")
	}

	if !ml.store.releaseLease(partitionID, lease.Token) {
		return false, errors.New("could not release the lease")
	}
	delete(ml.leases, partitionID)
	return true, nil
}

func (ml *memoryLeaserCheckpointer) UpdateLease(ctx context.Context, partitionID string) (LeaseMarker, bool, error) {
	span, _ := startConsumerSpanFromContext(ctx, "eph.memoryLeaserCheckpointer.UpdateLease")
	defer span.End()

	lease, ok := ml.leases[partitionID]
	if !ok {
		return nil, false, errors.New("lease was not found")
	}

	if !ml.store.renewLease(partitionID, lease.Token, ml.leaseDuration) {
		return nil, false, errors.New("unable to renew lease")
	}

	if !ml.store.storeLease(partitionID, lease.Token, *lease) {
		return nil, false, errors.New("unable to store lease after renewal")
	}

	return lease, true, nil
}

func (ml *memoryLeaserCheckpointer) GetCheckpoint(ctx context.Context, partitionID string) (persist.Checkpoint, bool) {
	ml.memMu.Lock()
	defer ml.memMu.Unlock()

	span, _ := startConsumerSpanFromContext(ctx, "eph.memoryCheckpointer.GetCheckpoint")
	defer span.End()

	lease, ok := ml.leases[partitionID]
	if ok {
		return *lease.Checkpoint, ok
	}
	return persist.NewCheckpointFromStartOfStream(), ok
}

func (ml *memoryLeaserCheckpointer) EnsureCheckpoint(ctx context.Context, partitionID string) (persist.Checkpoint, error) {
	ml.memMu.Lock()
	defer ml.memMu.Unlock()

	span, _ := startConsumerSpanFromContext(ctx, "eph.memoryCheckpointer.EnsureCheckpoint")
	defer span.End()

	lease, ok := ml.leases[partitionID]
	if ok {
		if lease.Checkpoint == nil {
			checkpoint := persist.NewCheckpointFromStartOfStream()
			lease.Checkpoint = &checkpoint
		}
		return *lease.Checkpoint, nil
	}
	return persist.NewCheckpointFromStartOfStream(), nil
}

func (ml *memoryLeaserCheckpointer) UpdateCheckpoint(ctx context.Context, partitionID string, checkpoint persist.Checkpoint) error {
	ml.memMu.Lock()
	defer ml.memMu.Unlock()

	span, _ := startConsumerSpanFromContext(ctx, "eph.memoryCheckpointer.UpdateCheckpoint")
	defer span.End()

	lease, ok := ml.leases[partitionID]
	if !ok {
		return errors.New("lease for partition isn't owned by this EventProcessorHost")
	}

	lease.Checkpoint = &checkpoint
	if !ml.store.storeLease(partitionID, lease.Token, *lease) {
		return errors.New("could not store lease on update of checkpoint")
	}
	return nil
}

func (ml *memoryLeaserCheckpointer) DeleteCheckpoint(ctx context.Context, partitionID string) error {
	ml.memMu.Lock()
	defer ml.memMu.Unlock()

	span, _ := startConsumerSpanFromContext(ctx, "eph.memoryCheckpointer.DeleteCheckpoint")
	defer span.End()

	lease, ok := ml.leases[partitionID]
	if !ok {
		return errors.New("lease for partition isn't owned by this EventProcessorHost")
	}

	checkpoint := persist.NewCheckpointFromStartOfStream()
	lease.Checkpoint = &checkpoint
	if !ml.store.storeLease(partitionID, lease.Token, *lease) {
		return errors.New("failed to store deleted checkpoint")
	}
	ml.leases[partitionID] = lease
	return nil
}

func (ml *memoryLeaserCheckpointer) Close() error {
	return nil
}
