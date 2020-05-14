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
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/devigned/tab"
)

var (
	timeout = 60 * time.Second
)

const (
	// DefaultLeaseRenewalInterval defines the default amount of time between lease renewal attempts
	DefaultLeaseRenewalInterval = 10 * time.Second

	// DefaultLeaseDuration defines the default amount of time a lease is valid
	DefaultLeaseDuration = 60 * time.Second

	partitionIDTag = "eph.receiver.partitionID"
	epochTag       = "eph.receiver.epoch"

	greed = 15
)

type (
	scheduler struct {
		processor            *EventProcessorHost
		receivers            map[string]*leasedReceiver
		done                 func()
		leaseRenewalInterval time.Duration
		receiverMu           sync.Mutex
	}

	ownerCount struct {
		Owner  string
		Leases []LeaseMarker
	}
)

func newScheduler(eventHostProcessor *EventProcessorHost) *scheduler {
	return &scheduler{
		processor:            eventHostProcessor,
		receivers:            make(map[string]*leasedReceiver),
		leaseRenewalInterval: DefaultLeaseRenewalInterval,
	}
}

func (s *scheduler) Run(ctx context.Context) {
	ctx, done := context.WithCancel(ctx)
	s.done = done
	span, ctx := s.startConsumerSpanFromContext(ctx, "eph.scheduler.Run")
	defer span.End()

	for {
		select {
		case <-ctx.Done():
			s.dlog(ctx, "shutting down scan")
			return
		default:
			s.scan(ctx)
			skew := time.Duration(rand.Intn(1000)-500) * time.Millisecond
			time.Sleep(s.leaseRenewalInterval + skew)
		}
	}
}

func (s *scheduler) scan(ctx context.Context) {
	span, ctx := s.startConsumerSpanFromContext(ctx, "eph.scheduler.scan")
	defer span.End()

	s.dlog(ctx, "running scan")

	// fetch updated view of all leases
	leaseCtx, cancel := context.WithTimeout(ctx, timeout)
	allLeases, err := s.processor.leaser.GetLeases(leaseCtx)
	cancel()
	if err != nil {
		tab.For(ctx).Error(err)
		return
	}

	randomLeases := make([]LeaseMarker, len(allLeases))
	perm := rand.Perm(len(allLeases))
	for i, v := range perm {
		randomLeases[v] = allLeases[i]
	}
	allLeases = randomLeases

	// try to acquire any leases that have expired
	acquired, notAcquired, err := s.acquireExpiredLeases(ctx, allLeases)
	s.dlog(ctx, fmt.Sprintf("acquired: %v, not acquired: %v", acquired, notAcquired))
	if err != nil {
		tab.For(ctx).Error(err)
		return
	}

	// start receiving message from newly acquired partitions
	for _, lease := range acquired {
		if err := s.startReceiver(ctx, lease); err != nil {
			_, _ = s.processor.leaser.ReleaseLease(ctx, lease.GetPartitionID())
			tab.For(ctx).Error(err)
			return
		}
	}

	if len(acquired) >= greed {
		// don't be too greedy
		return
	}

	// calculate the number of leases we own including the newly acquired partitions
	byOwner := leasesByOwner(notAcquired)
	var countOwnedByMe int
	if val, ok := byOwner[s.processor.name]; ok {
		countOwnedByMe = len(val)
	}
	countOwnedByMe += len(acquired)

	// gather all of the leases owned by others
	var leasesOwnedByOthers []LeaseMarker
	for key, value := range byOwner {
		if key != s.processor.name {
			leasesOwnedByOthers = append(leasesOwnedByOthers, value...)
		}
	}

	// try to steal work away from others if work has become imbalanced
	if candidate, ok := s.leaseToSteal(ctx, leasesOwnedByOthers, countOwnedByMe); ok {
		s.dlog(ctx, fmt.Sprintf("attempting to steal: %v", candidate))
		acquireCtx, cancel := context.WithTimeout(ctx, timeout)
		stolen, ok, err := s.processor.leaser.AcquireLease(acquireCtx, candidate.GetPartitionID())
		cancel()
		switch {
		case err != nil:
			tab.For(ctx).Error(err)
		case !ok:
			s.dlog(ctx, fmt.Sprintf("failed to steal: %v", candidate))
		default:
			s.dlog(ctx, fmt.Sprintf("stole: %v", stolen))
			if err := s.startReceiver(ctx, stolen); err != nil {
				_, _ = s.processor.leaser.ReleaseLease(acquireCtx, candidate.GetPartitionID())
				tab.For(ctx).Error(err)
				return
			}
		}
	}
}

func (s *scheduler) Stop(ctx context.Context) error {
	s.receiverMu.Lock()
	defer s.receiverMu.Unlock()

	span, ctx := s.startConsumerSpanFromContext(ctx, "eph.scheduler.Stop")
	defer span.End()

	if s.done != nil {
		s.done()
	}

	// close all receivers even if errors occur reporting only the last error, but logging all
	var lastErr error
	for _, lr := range s.receivers {
		if err := lr.Close(ctx); err != nil {
			lastErr = err
		}
		_, _ = s.processor.leaser.ReleaseLease(ctx, lr.lease.GetPartitionID())
	}

	return lastErr
}

func (s *scheduler) getPartitionIDsBeingProcessed() []string {
	s.receiverMu.Lock()
	defer s.receiverMu.Unlock()

	ids := make([]string, len(s.receivers))
	count := 0
	for id := range s.receivers {
		ids[count] = id
		count++
	}

	return ids
}

func (s *scheduler) startReceiver(ctx context.Context, lease LeaseMarker) error {
	s.receiverMu.Lock()
	defer s.receiverMu.Unlock()

	span, ctx := s.startConsumerSpanFromContext(ctx, "eph.scheduler.startReceiver")
	defer span.End()

	if receiver, ok := s.receivers[lease.GetPartitionID()]; ok {
		// receiver thinks it's already running... this is probably a bug if it happens
		if err := receiver.Close(ctx); err != nil {
			tab.For(ctx).Error(err)
		}
		delete(s.receivers, lease.GetPartitionID())
	}

	span.AddAttributes(
		tab.StringAttribute(partitionIDTag, lease.GetPartitionID()),
		tab.Int64Attribute(epochTag, lease.GetEpoch()),
	)
	lr := newLeasedReceiver(s.processor, lease)
	if err := lr.Run(ctx); err != nil {
		tab.For(ctx).Error(err)
		return err
	}
	s.receivers[lease.GetPartitionID()] = lr
	return nil
}

func (s *scheduler) stopReceiver(ctx context.Context, lease LeaseMarker) error {
	s.receiverMu.Lock()
	defer s.receiverMu.Unlock()

	span, ctx := s.startConsumerSpanFromContext(ctx, "eph.scheduler.stopReceiver")
	defer span.End()

	span.AddAttributes(
		tab.StringAttribute(partitionIDTag, lease.GetPartitionID()),
		tab.Int64Attribute(epochTag, lease.GetEpoch()),
	)
	s.dlog(ctx, fmt.Sprintf("stopping receiver for partitionID %q", lease.GetPartitionID()))
	if receiver, ok := s.receivers[lease.GetPartitionID()]; ok {
		// try to release the lease if possible
		_, _ = s.processor.leaser.ReleaseLease(ctx, lease.GetPartitionID())
		err := receiver.Close(ctx)
		delete(s.receivers, lease.GetPartitionID())
		if err != nil {
			tab.For(ctx).Error(err)
			return err
		}
	}
	return nil
}

func (s *scheduler) acquireExpiredLeases(ctx context.Context, leases []LeaseMarker) (acquired []LeaseMarker, notAcquired []LeaseMarker, err error) {
	span, ctx := s.startConsumerSpanFromContext(ctx, "eph.scheduler.acquireExpiredLeases")
	defer span.End()

	for _, lease := range leases {
		if lease.IsExpired(ctx) && len(acquired) < greed {
			// if lease has no owner or is expired and we haven't been too greedy
			acquireCtx, cancel := context.WithTimeout(ctx, timeout)
			if acquiredLease, ok, err := s.processor.leaser.AcquireLease(acquireCtx, lease.GetPartitionID()); ok {
				cancel()
				acquired = append(acquired, acquiredLease)
			} else {
				cancel()
				if err != nil {
					return nil, nil, err
				}
				notAcquired = append(notAcquired, lease)
			}
		} else {
			notAcquired = append(notAcquired, lease)
		}

	}
	return acquired, notAcquired, nil
}

func (s *scheduler) dlog(ctx context.Context, msg string) {
	name := s.processor.name
	tab.For(ctx).Debug(fmt.Sprintf("eph %q: "+msg, name))
}

func (s *scheduler) leaseToSteal(ctx context.Context, candidates []LeaseMarker, myLeaseCount int) (LeaseMarker, bool) {
	span, ctx := s.startConsumerSpanFromContext(ctx, "eph.scheduler.leaseToSteal")
	defer span.End()

	biggestOwner := ownerWithMostLeases(candidates)
	if biggestOwner != nil && s.processor.GetName() != biggestOwner.Owner {
		leasesByOwner := leasesByOwner(candidates)
		tab.For(ctx).Debug(fmt.Sprintf("i am %v, the biggest owner is %v and leases by owner: %v", s.processor.GetName(), biggestOwner.Owner, leasesByOwner))
		if leasesByOwner[biggestOwner.Owner] != nil &&
			(len(biggestOwner.Leases)-myLeaseCount) >= 2 && len(leasesByOwner[biggestOwner.Owner]) >= 1 {
			selection := rand.Intn(len(leasesByOwner[biggestOwner.Owner]))
			return leasesByOwner[biggestOwner.Owner][selection], true
		}
	}
	return nil, false
}

func ownerWithMostLeases(candidates []LeaseMarker) *ownerCount {
	var largest *ownerCount
	for key, value := range leasesByOwner(candidates) {
		if largest == nil || len(largest.Leases) < len(value) {
			largest = &ownerCount{
				Owner:  key,
				Leases: value,
			}
		}
	}
	return largest
}

func leasesByOwner(candidates []LeaseMarker) map[string][]LeaseMarker {
	byOwner := make(map[string][]LeaseMarker)
	for _, candidate := range candidates {
		if val, ok := byOwner[candidate.GetOwner()]; ok {
			byOwner[candidate.GetOwner()] = append(val, candidate)
		} else {
			byOwner[candidate.GetOwner()] = []LeaseMarker{candidate}
		}
	}
	return byOwner
}

func (s *scheduler) startConsumerSpanFromContext(ctx context.Context, operationName string) (tab.Spanner, context.Context) {
	span, ctx := startConsumerSpanFromContext(ctx, operationName)
	span.AddAttributes(tab.StringAttribute("eph.id", s.processor.name))
	return span, ctx
}
