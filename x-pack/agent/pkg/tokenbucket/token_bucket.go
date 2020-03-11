// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package tokenbucket

import (
	"fmt"
	"time"

	"github.com/elastic/beats/v7/x-pack/agent/pkg/scheduler"
)

// Bucket is a Token Bucket for rate limiting
type Bucket struct {
	size       int
	dropAmount int
	rateChan   chan struct{}
	closeChan  chan struct{}
	scheduler  scheduler.Scheduler
}

// NewTokenBucket creates a bucket and starts it.
// size: total size of the bucket
// dropAmount: amount which is dropped per every specified interval
// dropRate: specified interval when drop will happen
func NewTokenBucket(size, dropAmount int, dropRate time.Duration) (*Bucket, error) {
	s := scheduler.NewPeriodic(dropRate)
	return newTokenBucketWithScheduler(size, dropAmount, s)
}

func newTokenBucketWithScheduler(
	size, dropAmount int,
	s scheduler.Scheduler,
) (*Bucket, error) {
	if dropAmount > size {
		return nil, fmt.Errorf(
			"TokenBucket: invalid configuration, size '%d' is lower than drop amount '%d'",
			size,
			dropAmount,
		)
	}

	b := &Bucket{
		dropAmount: dropAmount,
		rateChan:   make(chan struct{}, size),
		closeChan:  make(chan struct{}),
		scheduler:  s,
	}
	go b.run()

	return b, nil
}

// Add adds item into a bucket. Add blocks until it is able to add item into a bucket.
func (b *Bucket) Add() {
	b.rateChan <- struct{}{}
}

// Close stops the rate limiting and does not let pass anything anymore.
func (b *Bucket) Close() {
	close(b.closeChan)
	close(b.rateChan)
	b.scheduler.Stop()
}

// run runs basic loop and consumes configured tokens per every configured period.
func (b *Bucket) run() {
	for {
		select {
		case <-b.scheduler.WaitTick():
			for i := 0; i < b.dropAmount; i++ {
				select {
				case <-b.rateChan:
				default: // do not cumulate drops
				}
			}
		case <-b.closeChan:
			return
		}
	}
}
