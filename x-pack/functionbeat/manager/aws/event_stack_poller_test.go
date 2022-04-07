// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/cloudformationiface"
	"github.com/stretchr/testify/assert"
	"go.uber.org/atomic"

	"github.com/elastic/beats/v8/libbeat/logp"
)

type mockEventHandler struct {
	skipEvents int32
	skipCount  atomic.Int32
	count      atomic.Int32
	events     chan cloudformation.StackEvent
}

func (m *mockEventHandler) sync(event cloudformation.StackEvent) bool {
	if m.skipCount.Load() >= m.skipEvents {
		return false
	}
	m.skipCount.Inc()
	return true
}

func (m *mockEventHandler) handle(event cloudformation.StackEvent) {
	m.events <- event
}

type mockCloudFormationClient struct {
	cloudformationiface.ClientAPI
	Responses []cloudformation.DescribeStackEventsOutput
	Index     int
}

func (m *mockCloudFormationClient) DescribeStackEventsRequest(
	input *cloudformation.DescribeStackEventsInput,
) cloudformation.DescribeStackEventsRequest {
	defer func() {
		// This minic the fact that the last token will be nil.
		if m.Index < len(m.Responses)-1 {
			m.Index++
		}
	}()
	httpReq, _ := http.NewRequest("", "", nil)
	return cloudformation.DescribeStackEventsRequest{
		Request: &aws.Request{Data: &m.Responses[m.Index], HTTPRequest: httpReq, Retryer: aws.NoOpRetryer{}},
	}
}

func TestEventStackPoller(t *testing.T) {
	t.Run("emits all events", testEmitAllEvents)
	t.Run("skip events", testSkipEvents)
	t.Run("skip duplicates", testSkipDuplicates)
	t.Run("return time ordered events", testReturnTimeOrdered)
}

func testEmitAllEvents(t *testing.T) {
	response1 := cloudformation.DescribeStackEventsOutput{
		NextToken: ptr("12345"),
		StackEvents: []cloudformation.StackEvent{
			cloudformation.StackEvent{EventId: ptr("1"), Timestamp: ptrTime(time.Now())},
			cloudformation.StackEvent{EventId: ptr("2"), Timestamp: ptrTime(time.Now())},
			cloudformation.StackEvent{EventId: ptr("3"), Timestamp: ptrTime(time.Now())},
			cloudformation.StackEvent{EventId: ptr("4"), Timestamp: ptrTime(time.Now())},
			cloudformation.StackEvent{EventId: ptr("5"), Timestamp: ptrTime(time.Now())},
		},
	}
	response2 := cloudformation.DescribeStackEventsOutput{
		StackEvents: []cloudformation.StackEvent{
			cloudformation.StackEvent{EventId: ptr("6"), Timestamp: ptrTime(time.Now())},
			cloudformation.StackEvent{EventId: ptr("7"), Timestamp: ptrTime(time.Now())},
			cloudformation.StackEvent{EventId: ptr("8"), Timestamp: ptrTime(time.Now())},
			cloudformation.StackEvent{EventId: ptr("9"), Timestamp: ptrTime(time.Now())},
			cloudformation.StackEvent{EventId: ptr("10"), Timestamp: ptrTime(time.Now())},
		},
	}

	client := &mockCloudFormationClient{Responses: []cloudformation.DescribeStackEventsOutput{
		response1,
		response2,
	}}

	handler := &mockEventHandler{events: make(chan cloudformation.StackEvent)}
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
	response1 := cloudformation.DescribeStackEventsOutput{
		NextToken: ptr("12345"),
		StackEvents: []cloudformation.StackEvent{
			cloudformation.StackEvent{EventId: ptr("1"), Timestamp: ptrTime(time.Now())},
			cloudformation.StackEvent{EventId: ptr("2"), Timestamp: ptrTime(time.Now())},
			cloudformation.StackEvent{EventId: ptr("3"), Timestamp: ptrTime(time.Now())},
			cloudformation.StackEvent{EventId: ptr("4"), Timestamp: ptrTime(time.Now())},
			cloudformation.StackEvent{EventId: ptr("5"), Timestamp: ptrTime(time.Now())},
		},
	}
	response2 := cloudformation.DescribeStackEventsOutput{
		StackEvents: []cloudformation.StackEvent{
			cloudformation.StackEvent{EventId: ptr("6"), Timestamp: ptrTime(time.Now())},
			cloudformation.StackEvent{EventId: ptr("7"), Timestamp: ptrTime(time.Now())},
			cloudformation.StackEvent{EventId: ptr("8"), Timestamp: ptrTime(time.Now())},
			cloudformation.StackEvent{EventId: ptr("9"), Timestamp: ptrTime(time.Now())},
			cloudformation.StackEvent{EventId: ptr("10"), Timestamp: ptrTime(time.Now())},
		},
	}

	client := &mockCloudFormationClient{Responses: []cloudformation.DescribeStackEventsOutput{
		response1,
		response2,
	}}

	handler := &mockEventHandler{skipEvents: 3, events: make(chan cloudformation.StackEvent)}
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
	response1 := cloudformation.DescribeStackEventsOutput{
		NextToken: ptr("12345"),
		StackEvents: []cloudformation.StackEvent{
			cloudformation.StackEvent{EventId: ptr("1"), Timestamp: ptrTime(time.Now())},
			cloudformation.StackEvent{EventId: ptr("2"), Timestamp: ptrTime(time.Now())},
			cloudformation.StackEvent{EventId: ptr("3"), Timestamp: ptrTime(time.Now())},
			cloudformation.StackEvent{EventId: ptr("4"), Timestamp: ptrTime(time.Now())},
			cloudformation.StackEvent{EventId: ptr("4"), Timestamp: ptrTime(time.Now())},
		},
	}
	response2 := cloudformation.DescribeStackEventsOutput{
		StackEvents: []cloudformation.StackEvent{
			cloudformation.StackEvent{EventId: ptr("1"), Timestamp: ptrTime(time.Now())},
			cloudformation.StackEvent{EventId: ptr("2"), Timestamp: ptrTime(time.Now())},
			cloudformation.StackEvent{EventId: ptr("2"), Timestamp: ptrTime(time.Now())},
			cloudformation.StackEvent{EventId: ptr("4"), Timestamp: ptrTime(time.Now())},
			cloudformation.StackEvent{EventId: ptr("5"), Timestamp: ptrTime(time.Now())},
		},
	}

	client := &mockCloudFormationClient{Responses: []cloudformation.DescribeStackEventsOutput{
		response1,
		response2,
	}}

	handler := &mockEventHandler{skipEvents: 3, events: make(chan cloudformation.StackEvent)}
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
	response1 := cloudformation.DescribeStackEventsOutput{
		NextToken: ptr("12345"),
		StackEvents: []cloudformation.StackEvent{
			cloudformation.StackEvent{EventId: ptr("5"), Timestamp: ptrTime(time.Now())},
			cloudformation.StackEvent{EventId: ptr("4"), Timestamp: ptrTime(time.Now())},
			cloudformation.StackEvent{EventId: ptr("3"), Timestamp: ptrTime(time.Now())},
			cloudformation.StackEvent{EventId: ptr("2"), Timestamp: ptrTime(time.Now())},
			cloudformation.StackEvent{EventId: ptr("1"), Timestamp: ptrTime(time.Now())},
		},
	}

	client := &mockCloudFormationClient{Responses: []cloudformation.DescribeStackEventsOutput{
		response1,
	}}

	handler := &mockEventHandler{events: make(chan cloudformation.StackEvent)}
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
		event cloudformation.StackEvent
		sync  bool
	}{
		{
			name: "is stack event but happened before",
			event: cloudformation.StackEvent{
				ResourceType: ptr("AWS::CloudFormation::Stack"),
				EventId:      ptr("1"),
				Timestamp:    ptrTime(now.Add(-10 * time.Second)),
			},
			sync: true,
		},
		{
			name: "is not a stack event",
			event: cloudformation.StackEvent{
				ResourceType: ptr("AWS::S3::Bucket"),
				EventId:      ptr("2"),
				Timestamp:    ptrTime(now.Add(10 * time.Second)),
			},
			sync: true,
		},
		{
			name: "is a stack event and happens after but with wrong status",
			event: cloudformation.StackEvent{
				ResourceType:   ptr("AWS::CloudFormation::Stack"),
				ResourceStatus: cloudformation.ResourceStatusDeleteFailed,
				EventId:        ptr("2"),
				Timestamp:      ptrTime(now.Add(11 * time.Second)),
			},
			sync: true,
		},
		{
			name: "is a stack event and happens after with a CREATE_IN_PROGRESS status",
			event: cloudformation.StackEvent{
				ResourceType:   ptr("AWS::CloudFormation::Stack"),
				ResourceStatus: cloudformation.ResourceStatusCreateInProgress,
				EventId:        ptr("2"),
				Timestamp:      ptrTime(now.Add(11 * time.Second)),
			},
			sync: false,
		},
		{
			name: "is a stack event and happens after with an UPDATE_IN_PROGRESS status",
			event: cloudformation.StackEvent{
				ResourceType:   ptr("AWS::CloudFormation::Stack"),
				ResourceStatus: cloudformation.ResourceStatusUpdateInProgress,
				EventId:        ptr("2"),
				Timestamp:      ptrTime(now.Add(11 * time.Second)),
			},
			sync: false,
		},
		{
			name: "is a stack event and happens after with an DELETE_IN_PROGRESS status",
			event: cloudformation.StackEvent{
				ResourceType:   ptr("AWS::CloudFormation::Stack"),
				ResourceStatus: cloudformation.ResourceStatusDeleteInProgress,
				EventId:        ptr("2"),
				Timestamp:      ptrTime(now.Add(11 * time.Second)),
			},
			sync: false,
		},
	}

	reporter := reportStackEvent{
		skipBefore: now,
		callback:   func(event cloudformation.StackEvent) {},
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
		callback:   func(event cloudformation.StackEvent) { received = true },
	}

	reporter.handle(cloudformation.StackEvent{})
	assert.True(t, received)
}

func ptr(v string) *string {
	return &v
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
