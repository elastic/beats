// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"context"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/logp"
)

type mockEventHandler struct {
	skipEvents int32
	skipCount  atomic.Int32
	events     chan types.StackEvent
}

func (m *mockEventHandler) sync(event types.StackEvent) bool {
	if m.skipCount.Load() >= m.skipEvents {
		return false
	}
	m.skipCount.Add(1)
	return true
}

func (m *mockEventHandler) handle(event types.StackEvent) {
	m.events <- event
}

type mockCloudFormationClient struct {
	Responses []*cloudformation.DescribeStackEventsOutput
	Index     int
}

func (m *mockCloudFormationClient) DescribeStackEvents(context.Context, *cloudformation.DescribeStackEventsInput, ...func(*cloudformation.Options)) (*cloudformation.DescribeStackEventsOutput, error) {
	defer func() {
		// This minic the fact that the last token will be nil.
		if m.Index < len(m.Responses)-1 {
			m.Index++
		}
	}()

	return m.Responses[m.Index], nil
}

func TestEventStackPoller(t *testing.T) {
	t.Run("emits all events", testEmitAllEvents)
	t.Run("skip events", testSkipEvents)
	t.Run("skip duplicates", testSkipDuplicates)
	t.Run("return time ordered events", testReturnTimeOrdered)
}

func testEmitAllEvents(t *testing.T) {
	response1 := &cloudformation.DescribeStackEventsOutput{
		NextToken: ptr("12345"),
		StackEvents: []types.StackEvent{
			types.StackEvent{EventId: ptr("1"), Timestamp: ptrTime(time.Now())},
			types.StackEvent{EventId: ptr("2"), Timestamp: ptrTime(time.Now())},
			types.StackEvent{EventId: ptr("3"), Timestamp: ptrTime(time.Now())},
			types.StackEvent{EventId: ptr("4"), Timestamp: ptrTime(time.Now())},
			types.StackEvent{EventId: ptr("5"), Timestamp: ptrTime(time.Now())},
		},
	}
	response2 := &cloudformation.DescribeStackEventsOutput{
		StackEvents: []types.StackEvent{
			types.StackEvent{EventId: ptr("6"), Timestamp: ptrTime(time.Now())},
			types.StackEvent{EventId: ptr("7"), Timestamp: ptrTime(time.Now())},
			types.StackEvent{EventId: ptr("8"), Timestamp: ptrTime(time.Now())},
			types.StackEvent{EventId: ptr("9"), Timestamp: ptrTime(time.Now())},
			types.StackEvent{EventId: ptr("10"), Timestamp: ptrTime(time.Now())},
		},
	}

	client := &mockCloudFormationClient{Responses: []*cloudformation.DescribeStackEventsOutput{
		response1,
		response2,
	}}

	handler := &mockEventHandler{events: make(chan types.StackEvent)}
	poller := newEventStackPoller(
		logp.NewLogger(""),
		client,
		ptr("1235"),
		1,
		handler,
	)
	poller.Start()
	defer poller.Stop()

	var c int
	for range handler.events {
		c++
		if c == 10 {
			return
		}
	}
}

func testSkipEvents(t *testing.T) {
	response1 := &cloudformation.DescribeStackEventsOutput{
		NextToken: ptr("12345"),
		StackEvents: []types.StackEvent{
			types.StackEvent{EventId: ptr("1"), Timestamp: ptrTime(time.Now())},
			types.StackEvent{EventId: ptr("2"), Timestamp: ptrTime(time.Now())},
			types.StackEvent{EventId: ptr("3"), Timestamp: ptrTime(time.Now())},
			types.StackEvent{EventId: ptr("4"), Timestamp: ptrTime(time.Now())},
			types.StackEvent{EventId: ptr("5"), Timestamp: ptrTime(time.Now())},
		},
	}
	response2 := &cloudformation.DescribeStackEventsOutput{
		StackEvents: []types.StackEvent{
			types.StackEvent{EventId: ptr("6"), Timestamp: ptrTime(time.Now())},
			types.StackEvent{EventId: ptr("7"), Timestamp: ptrTime(time.Now())},
			types.StackEvent{EventId: ptr("8"), Timestamp: ptrTime(time.Now())},
			types.StackEvent{EventId: ptr("9"), Timestamp: ptrTime(time.Now())},
			types.StackEvent{EventId: ptr("10"), Timestamp: ptrTime(time.Now())},
		},
	}

	client := &mockCloudFormationClient{Responses: []*cloudformation.DescribeStackEventsOutput{
		response1,
		response2,
	}}

	handler := &mockEventHandler{skipEvents: 3, events: make(chan types.StackEvent)}
	poller := newEventStackPoller(
		logp.NewLogger(""),
		client,
		ptr("1235"),
		0,
		handler,
	)
	poller.Start()
	defer poller.Stop()

	var c int
	for range handler.events {
		c++
		if c == 7 {
			return
		}
	}
}

func testSkipDuplicates(t *testing.T) {
	response1 := &cloudformation.DescribeStackEventsOutput{
		NextToken: ptr("12345"),
		StackEvents: []types.StackEvent{
			types.StackEvent{EventId: ptr("1"), Timestamp: ptrTime(time.Now())},
			types.StackEvent{EventId: ptr("2"), Timestamp: ptrTime(time.Now())},
			types.StackEvent{EventId: ptr("3"), Timestamp: ptrTime(time.Now())},
			types.StackEvent{EventId: ptr("4"), Timestamp: ptrTime(time.Now())},
			types.StackEvent{EventId: ptr("4"), Timestamp: ptrTime(time.Now())},
		},
	}
	response2 := &cloudformation.DescribeStackEventsOutput{
		StackEvents: []types.StackEvent{
			types.StackEvent{EventId: ptr("1"), Timestamp: ptrTime(time.Now())},
			types.StackEvent{EventId: ptr("2"), Timestamp: ptrTime(time.Now())},
			types.StackEvent{EventId: ptr("2"), Timestamp: ptrTime(time.Now())},
			types.StackEvent{EventId: ptr("4"), Timestamp: ptrTime(time.Now())},
			types.StackEvent{EventId: ptr("5"), Timestamp: ptrTime(time.Now())},
		},
	}

	client := &mockCloudFormationClient{Responses: []*cloudformation.DescribeStackEventsOutput{
		response1,
		response2,
	}}

	handler := &mockEventHandler{skipEvents: 3, events: make(chan types.StackEvent)}
	poller := newEventStackPoller(
		logp.NewLogger(""),
		client,
		ptr("1235"),
		0,
		handler,
	)
	poller.Start()
	defer poller.Stop()

	var c int
	for range handler.events {
		c++
		if c == 4 {
			return
		}
	}
}

func testReturnTimeOrdered(t *testing.T) {
	response1 := &cloudformation.DescribeStackEventsOutput{
		NextToken: ptr("12345"),
		StackEvents: []types.StackEvent{
			types.StackEvent{EventId: ptr("5"), Timestamp: ptrTime(time.Now())},
			types.StackEvent{EventId: ptr("4"), Timestamp: ptrTime(time.Now())},
			types.StackEvent{EventId: ptr("3"), Timestamp: ptrTime(time.Now())},
			types.StackEvent{EventId: ptr("2"), Timestamp: ptrTime(time.Now())},
			types.StackEvent{EventId: ptr("1"), Timestamp: ptrTime(time.Now())},
		},
	}

	client := &mockCloudFormationClient{Responses: []*cloudformation.DescribeStackEventsOutput{
		response1,
	}}

	handler := &mockEventHandler{events: make(chan types.StackEvent)}
	poller := newEventStackPoller(
		logp.NewLogger(""),
		client,
		ptr("1235"),
		0,
		handler,
	)
	poller.Start()
	defer poller.Stop()

	c := 1
	for event := range handler.events {
		if c == 5 {
			return
		}

		assert.Equal(t, strconv.Itoa(c), *event.EventId)
		c++
	}
}

func TestReportStackEvent(t *testing.T) {
	t.Run("test skip event", testReportSkipEvents)
	t.Run("test that handle forward the event to callback", testReportCallback)
}

func testReportSkipEvents(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name  string
		event types.StackEvent
		sync  bool
	}{
		{
			name: "is stack event but happened before",
			event: types.StackEvent{
				ResourceType: ptr("AWS::CloudFormation::Stack"),
				EventId:      ptr("1"),
				Timestamp:    ptrTime(now.Add(-10 * time.Second)),
			},
			sync: true,
		},
		{
			name: "is not a stack event",
			event: types.StackEvent{
				ResourceType: ptr("AWS::S3::Bucket"),
				EventId:      ptr("2"),
				Timestamp:    ptrTime(now.Add(10 * time.Second)),
			},
			sync: true,
		},
		{
			name: "is a stack event and happens after but with wrong status",
			event: types.StackEvent{
				ResourceType:   ptr("AWS::CloudFormation::Stack"),
				ResourceStatus: types.ResourceStatusDeleteFailed,
				EventId:        ptr("2"),
				Timestamp:      ptrTime(now.Add(11 * time.Second)),
			},
			sync: true,
		},
		{
			name: "is a stack event and happens after with a CREATE_IN_PROGRESS status",
			event: types.StackEvent{
				ResourceType:   ptr("AWS::CloudFormation::Stack"),
				ResourceStatus: types.ResourceStatusCreateInProgress,
				EventId:        ptr("2"),
				Timestamp:      ptrTime(now.Add(11 * time.Second)),
			},
			sync: false,
		},
		{
			name: "is a stack event and happens after with an UPDATE_IN_PROGRESS status",
			event: types.StackEvent{
				ResourceType:   ptr("AWS::CloudFormation::Stack"),
				ResourceStatus: types.ResourceStatusUpdateInProgress,
				EventId:        ptr("2"),
				Timestamp:      ptrTime(now.Add(11 * time.Second)),
			},
			sync: false,
		},
		{
			name: "is a stack event and happens after with an DELETE_IN_PROGRESS status",
			event: types.StackEvent{
				ResourceType:   ptr("AWS::CloudFormation::Stack"),
				ResourceStatus: types.ResourceStatusDeleteInProgress,
				EventId:        ptr("2"),
				Timestamp:      ptrTime(now.Add(11 * time.Second)),
			},
			sync: false,
		},
	}

	reporter := reportStackEvent{
		skipBefore: now,
		callback:   func(event types.StackEvent) {},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.sync, reporter.sync(test.event))
		})
	}
}

func testReportCallback(t *testing.T) {
	var received bool
	reporter := reportStackEvent{
		skipBefore: time.Now(),
		callback:   func(event types.StackEvent) { received = true },
	}

	reporter.handle(types.StackEvent{})
	assert.True(t, received)
}

func ptr(v string) *string {
	return &v
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
