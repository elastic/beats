// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"sync"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/acker"
	"github.com/elastic/beats/v7/libbeat/common/fifo"
)

type s3ACKHandler struct {
	sync.Mutex
	pendingACKs fifo.FIFO[publishedS3Object]
	ackedCount  int
}

type publishedS3Object struct {
	publishCount int
	ackCallback  func()
}

func (ah *s3ACKHandler) Add(publishCount int, ackCallback func()) {
	ah.Lock()
	defer ah.Unlock()
	ah.pendingACKs.Add(publishedS3Object{
		publishCount: publishCount,
		ackCallback:  ackCallback,
	})
}

func (ah *s3ACKHandler) ACK(count int) {
	ah.Lock()
	ah.ackedCount += count
	callbacks := ah.advance()
	ah.Unlock()
	for _, c := range callbacks {
		c()
	}
}

// Advance the acks list based on the current ackedCount, invoking any
// acknowledgment callbacks for completed objects.
func (ah *s3ACKHandler) advance() []func() {
	var callbacks []func()
	for !ah.pendingACKs.Empty() {
		nextObj := ah.pendingACKs.First()
		if nextObj.publishCount > ah.ackedCount {
			// This object hasn't been fully acknowledged yet
			break
		}
		callbacks = append(callbacks, nextObj.ackCallback)
		ah.pendingACKs.Remove()
	}
	return callbacks
}

func (ah *s3ACKHandler) pipelineEventListener() beat.EventListener {
	return acker.TrackingCounter(func(_ int, total int) {
		ah.ACK(total)
	})
}
