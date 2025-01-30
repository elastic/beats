// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awscloudwatch

import (
	"context"
	"testing"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/logp"
)

type clock struct {
	time time.Time
}

func (c *clock) now() time.Time {
	return c.time
}

type receiveTestStep struct {
	expected []workResponse
	nextTime time.Time
}

type receiveTestCase struct {
	name            string
	logGroupIDs     []string
	configOverrides func(*config)
	startTime       time.Time
	steps           []receiveTestStep
}

func TestReceive(t *testing.T) {
	// We use a mocked clock so scan frequency can be any positive value.
	const defaultScanFrequency = time.Microsecond
	t0 := time.Time{}
	t1 := t0.Add(time.Hour)
	t2 := t1.Add(time.Minute)
	t3 := t2.Add(time.Hour)
	testCases := []receiveTestCase{
		{
			name:        "Default config with one log group",
			logGroupIDs: []string{"a"},
			startTime:   t1,
			steps: []receiveTestStep{
				{
					expected: []workResponse{
						{logGroupId: "a", startTime: t0, endTime: t1},
					},
					nextTime: t2,
				},
				{
					expected: []workResponse{
						{logGroupId: "a", startTime: t1, endTime: t2},
					},
					nextTime: t3,
				},
				{
					expected: []workResponse{
						{logGroupId: "a", startTime: t2, endTime: t3},
					},
				},
			},
		},
		{
			name:        "Default config with two log groups",
			logGroupIDs: []string{"a", "b"},
			startTime:   t1,
			steps: []receiveTestStep{
				{
					expected: []workResponse{
						{logGroupId: "a", startTime: t0, endTime: t1},
					},
					nextTime: t2,
				},
				{
					expected: []workResponse{
						// start/end times for the second log group should be the same
						// even though the clock has changed.
						{logGroupId: "b", startTime: t0, endTime: t1},
					},
				},
				{
					expected: []workResponse{
						{logGroupId: "a", startTime: t1, endTime: t2},
						{logGroupId: "b", startTime: t1, endTime: t2},
					},
					nextTime: t3,
				},
				{
					expected: []workResponse{
						{logGroupId: "a", startTime: t2, endTime: t3},
						{logGroupId: "b", startTime: t2, endTime: t3},
					},
				},
			},
		},
		{
			name:        "One log group with start_position: end",
			logGroupIDs: []string{"a"},
			startTime:   t1,
			configOverrides: func(c *config) {
				c.StartPosition = "end"
			},
			steps: []receiveTestStep{
				{
					expected: []workResponse{
						{logGroupId: "a", startTime: t1.Add(-defaultScanFrequency), endTime: t1},
					},
					nextTime: t2,
				},
				{
					expected: []workResponse{
						{logGroupId: "a", startTime: t1, endTime: t2},
					},
				},
			},
		},
		{
			name:        "Two log group with start_position: end and latency",
			logGroupIDs: []string{"a", "b"},
			startTime:   t1,
			configOverrides: func(c *config) {
				c.StartPosition = "end"
				c.Latency = time.Second
			},
			steps: []receiveTestStep{
				{
					expected: []workResponse{
						{logGroupId: "a", startTime: t1.Add(-defaultScanFrequency - time.Second), endTime: t1.Add(-time.Second)},
						{logGroupId: "b", startTime: t1.Add(-defaultScanFrequency - time.Second), endTime: t1.Add(-time.Second)},
					},
					nextTime: t2,
				},
				{
					expected: []workResponse{
						{logGroupId: "a", startTime: t1.Add(-time.Second), endTime: t2.Add(-time.Second)},
						{logGroupId: "b", startTime: t1.Add(-time.Second), endTime: t2.Add(-time.Second)},
					},
				},
			},
		},
		{
			name:        "Three log groups with latency",
			logGroupIDs: []string{"a", "b", "c"},
			startTime:   t1,
			configOverrides: func(c *config) {
				c.Latency = time.Second
			},
			steps: []receiveTestStep{
				{
					expected: []workResponse{
						{logGroupId: "a", startTime: t0, endTime: t1.Add(-time.Second)},
						{logGroupId: "b", startTime: t0, endTime: t1.Add(-time.Second)},
						{logGroupId: "c", startTime: t0, endTime: t1.Add(-time.Second)},
					},
					nextTime: t2,
				},
				{
					expected: []workResponse{
						{logGroupId: "a", startTime: t1.Add(-time.Second), endTime: t2.Add(-time.Second)},
						{logGroupId: "b", startTime: t1.Add(-time.Second), endTime: t2.Add(-time.Second)},
						{logGroupId: "c", startTime: t1.Add(-time.Second), endTime: t2.Add(-time.Second)},
					},
				},
			},
		},
	}
	clock := &clock{}
	for stepIndex, test := range testCases {
		ctx, cancel := context.WithCancel(context.Background())
		p := &cloudwatchPoller{
			workRequestChan: make(chan struct{}),
			// Unlike the live cwPoller, we make workResponseChan unbuffered,
			// so we can guarantee that clock updates happen when cwPoller has already
			// decided on its output
			workResponseChan: make(chan workResponse),
			log:              logp.NewLogger("test"),
		}

		p.config = defaultConfig()
		p.config.ScanFrequency = defaultScanFrequency
		if test.configOverrides != nil {
			test.configOverrides(&p.config)
		}
		clock.time = test.startTime
		go p.receive(ctx, test.logGroupIDs, clock.now)
		for _, step := range test.steps {
			for i, expected := range step.expected {
				p.workRequestChan <- struct{}{}
				if i+1 == len(step.expected) && !step.nextTime.Equal(time.Time{}) {
					// On the last request of the step, we advance the clock if a
					// time is set
					clock.time = step.nextTime
				}
				response := <-p.workResponseChan
				assert.Equalf(t, expected, response, "%v: step %v response %v doesn't match", test.name, stepIndex, i)
			}
		}
		cancel()
	}
}

type filterLogEventsTestCase struct {
	name       string
	logGroupId string
	startTime  time.Time
	endTime    time.Time
	expected   *cloudwatchlogs.FilterLogEventsInput
}

func TestFilterLogEventsInput(t *testing.T) {
	now, _ := time.Parse(time.RFC3339, "2024-07-12T13:00:00+00:00")
	id := "myLogGroup"

	testCases := []filterLogEventsTestCase{
		{
			name:       "StartPosition: beginning, first iteration",
			logGroupId: id,
			// The zero value of type time.Time{} is January 1, year 1, 00:00:00.000000000 UTC
			// Events with a timestamp before the time - January 1, 1970, 00:00:00 UTC are not returned by AWS API
			// make sure zero value of time.Time{} was converted
			startTime: time.Time{},
			endTime:   now,
			expected: &cloudwatchlogs.FilterLogEventsInput{
				LogGroupIdentifier: awssdk.String(id),
				StartTime:          awssdk.Int64(0),
				EndTime:            awssdk.Int64(1720789200000),
			},
		},
	}
	for _, test := range testCases {
		p := cloudwatchPoller{}
		result := p.constructFilterLogEventsInput(test.startTime, test.endTime, test.logGroupId)
		assert.Equal(t, test.expected, result)
	}

}
