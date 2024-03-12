// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/beat"
)

func TestEventACKTracker(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	deletionWg := new(sync.WaitGroup)
	deletionWg.Add(1)

	acker := NewEventACKTracker(ctx, deletionWg)
	_, keepaliveCancel := context.WithCancel(ctx)
	log := log.Named("sqs_s3_event")
	ctrl, ctx := gomock.WithContext(ctx, t)
	defer ctrl.Finish()
	mockMsgHandler := NewMockSQSProcessor(ctrl)
	msg := newSQSMessage(newS3Event("log.json"))
	mockMsgHandler.EXPECT().DeleteSQS(gomock.Eq(&msg), gomock.Eq(-1), gomock.Nil(), gomock.Nil()).Return(nil)
	acker.MarkSQSProcessedWithData(&msg, 1, -1, time.Now(), nil, nil, keepaliveCancel, new(sync.WaitGroup), mockMsgHandler, log)
	acker.Track(0, 1)
	acker.cancelAndFlush()

	assert.EqualValues(t, true, acker.FullyAcked())
	assert.ErrorIs(t, acker.ctx.Err(), context.Canceled)
}

func TestEventACKTrackerNoTrack(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	deletionWg := new(sync.WaitGroup)
	acker := NewEventACKTracker(ctx, deletionWg)
	cancel()
	<-acker.ctx.Done()

	assert.EqualValues(t, false, acker.FullyAcked())
	assert.ErrorIs(t, acker.ctx.Err(), context.Canceled)
}

func TestEventACKHandler(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	// Create acker. Add one ACK to event listener.
	deletionWg := new(sync.WaitGroup)
	deletionWg.Add(1)
	acker := NewEventACKTracker(ctx, deletionWg)
	_, keepaliveCancel := context.WithCancel(ctx)
	log := log.Named("sqs_s3_event")
	ctrl, ctx := gomock.WithContext(ctx, t)
	defer ctrl.Finish()
	mockMsgHandler := NewMockSQSProcessor(ctrl)
	msg := newSQSMessage(newS3Event("log.json"))
	mockMsgHandler.EXPECT().DeleteSQS(gomock.Eq(&msg), gomock.Eq(-1), gomock.Nil(), gomock.Nil()).Return(nil)
	acker.MarkSQSProcessedWithData(&msg, 1, -1, time.Now(), nil, nil, keepaliveCancel, new(sync.WaitGroup), mockMsgHandler, log)

	// Create an ACK handler and simulate one ACKed event.
	ackHandler := NewEventACKHandler()
	ackHandler.AddEvent(beat.Event{Private: acker}, true)
	ackHandler.ACKEvents(1)
	acker.cancelAndFlush()

	assert.EqualValues(t, true, acker.FullyAcked())
	assert.ErrorIs(t, acker.ctx.Err(), context.Canceled)
}

func TestEventACKHandlerFullyAcked(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	// Create acker. Add one Track.
	deletionWg := new(sync.WaitGroup)
	deletionWg.Add(1)

	acker := NewEventACKTracker(ctx, deletionWg)
	_, keepaliveCancel := context.WithCancel(ctx)
	log := log.Named("sqs_s3_event")
	ctrl, ctx := gomock.WithContext(ctx, t)
	defer ctrl.Finish()
	mockMsgHandler := NewMockSQSProcessor(ctrl)
	msg := newSQSMessage(newS3Event("log.json"))
	mockMsgHandler.EXPECT().DeleteSQS(gomock.Eq(&msg), gomock.Eq(-1), gomock.Nil(), gomock.Nil()).Return(nil)
	acker.MarkSQSProcessedWithData(&msg, 1, -1, time.Now(), nil, nil, keepaliveCancel, new(sync.WaitGroup), mockMsgHandler, log)
	acker.Track(0, 1)
	acker.cancelAndFlush()
	assert.EqualValues(t, true, acker.FullyAcked())

	assert.ErrorIs(t, acker.ctx.Err(), context.Canceled)

	acker.EventsToBeTracked.Inc()

	assert.EqualValues(t, false, acker.FullyAcked())
}
