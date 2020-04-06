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
	"encoding/json"
	"io"
	"sync/atomic"
)

type (
	// StoreProvisioner provides CRUD functionality for Lease and Checkpoint storage
	StoreProvisioner interface {
		StoreExists(ctx context.Context) (bool, error)
		EnsureStore(ctx context.Context) error
		DeleteStore(ctx context.Context) error
	}

	// EventProcessHostSetter provides the ability to set an EventHostProcessor on the implementor
	EventProcessHostSetter interface {
		SetEventHostProcessor(eph *EventProcessorHost)
	}

	// Leaser provides the functionality needed to persist and coordinate leases for partitions
	Leaser interface {
		io.Closer
		StoreProvisioner
		EventProcessHostSetter
		GetLeases(ctx context.Context) ([]LeaseMarker, error)
		EnsureLease(ctx context.Context, partitionID string) (LeaseMarker, error)
		DeleteLease(ctx context.Context, partitionID string) error
		AcquireLease(ctx context.Context, partitionID string) (LeaseMarker, bool, error)
		RenewLease(ctx context.Context, partitionID string) (LeaseMarker, bool, error)
		ReleaseLease(ctx context.Context, partitionID string) (bool, error)
		UpdateLease(ctx context.Context, partitionID string) (LeaseMarker, bool, error)
	}

	// Lease represents the information needed to coordinate partitions
	Lease struct {
		PartitionID string `json:"partitionID"`
		Epoch       int64  `json:"epoch"`
		Owner       string `json:"owner"`
	}

	// LeaseMarker provides the functionality expected of a partition lease with an owner
	LeaseMarker interface {
		GetPartitionID() string
		IsExpired(context.Context) bool
		GetOwner() string
		IncrementEpoch() int64
		GetEpoch() int64
		String() string
	}
)

// GetPartitionID returns the partition which belongs to this lease
func (l *Lease) GetPartitionID() string {
	return l.PartitionID
}

// GetOwner returns the owner of the lease
func (l *Lease) GetOwner() string {
	return l.Owner
}

// IncrementEpoch increase the time on the lease by one
func (l *Lease) IncrementEpoch() int64 {
	return atomic.AddInt64(&l.Epoch, 1)
}

// GetEpoch returns the value of the epoch
func (l *Lease) GetEpoch() int64 {
	return l.Epoch
}

func (l *Lease) String() string {
	bytes, _ := json.Marshal(l)
	return string(bytes)
}
