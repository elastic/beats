// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"context"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/cloudformationiface"

	"github.com/elastic/elastic-agent-libs/logp"
)

// Resource type to look for.
const eventAWSCloudFormationStack = "AWS::CloudFormation::Stack"

type eventStackHandler interface {
	sync(event cloudformation.StackEvent) bool
	handle(event cloudformation.StackEvent)
}

// eventStackPoller takes a stack id and will report any events coming from it.
// The event stream for a stack will return all the events for a specific existance of a stack,
// its important to be able to skip some events and only report the meaningful events.
type eventStackPoller struct {
	log           *logp.Logger
	svc           cloudformationiface.ClientAPI
	stackID       *string
	periodicCheck time.Duration
	handler       eventStackHandler
	done          chan struct{}
	wg            sync.WaitGroup
}

func newEventStackPoller(
	log *logp.Logger,
	svc cloudformationiface.ClientAPI,
	stackID *string,
	periodicCheck time.Duration,
	handler eventStackHandler,
) *eventStackPoller {
	return &eventStackPoller{
		log:           log,
		svc:           svc,
		stackID:       stackID,
		periodicCheck: periodicCheck,
		handler:       handler,
		done:          make(chan struct{}),
	}
}

func (e *eventStackPoller) Start() {
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		e.poll()
	}()
}

func (e *eventStackPoller) Stop() {
	close(e.done)
	e.wg.Wait()
}

func (e *eventStackPoller) poll() {
	var nextToken *string
	var foundFirstEvent bool
	var alreadyLoggedEvents = make(map[string]struct{})
	for {
		input := &cloudformation.DescribeStackEventsInput{
			NextToken: nextToken,
			StackName: e.stackID,
		}

		// Currently no way to skip items based on time.
		// doc: https://docs.aws.amazon.com/AWSCloudFormation/latest/APIReference/API_DescribeStackEvents.html
		req := e.svc.DescribeStackEventsRequest(input)
		resp, err := req.Send(context.TODO())
		if err != nil {
			// This is not a fatal error because the check is made out of bound from the current status logic.
			// I wanted to keep them separate so it is easier to deal with states and reporting.
			e.log.Errorf("Could not retrieve the events for stack, error: %+v", err)
		}

		// Events are in reverse order. we need older -> new, but we do not rely on the time but just
		// the position in the slice.
		for i, j := 0, len(resp.StackEvents)-1; i < j; i, j = i+1, j-1 {
			resp.StackEvents[i], resp.StackEvents[j] = resp.StackEvents[j], resp.StackEvents[i]
		}

		for _, event := range resp.StackEvents {
			// Since we receive all the events from the beginning of the stack we have
			// to first position ourself to an event of interest.
			if !foundFirstEvent {
				if !e.handler.sync(event) {
					// keep current event and position to the first meaningful event.
					foundFirstEvent = true
				} else {
					//discard current event.
					continue
				}
			}

			// When the stack is in progress we will receive an empty token, so we have to make another
			// call with the current token this mean we probably have already see the events so we have to
			// ignore them. I am using ids here instance of time because I think 2 events might have
			// 	the same time. I am assuming this map should stay relatively small.
			if _, ok := alreadyLoggedEvents[*event.EventId]; ok {
				continue
			}

			e.handler.handle(event)
			alreadyLoggedEvents[*event.EventId] = struct{}{}
		}

		select {
		case <-e.done:
			// if nextToken is nil it mean we are at the end of the current pages.
			// if not it mean we still have log to get and we need to report them before quitting.
			if nextToken == nil {
				return
			}
		case <-time.After(e.periodicCheck):
		}
	}
}

type reportStackEvent struct {
	skipBefore time.Time
	callback   func(event cloudformation.StackEvent)
}

func (r *reportStackEvent) sync(event cloudformation.StackEvent) bool {
	// Ignore anything before the Start pointer and everything which is not AWS::CloudFormation::Stack
	if r.skipBefore.Before(*event.Timestamp) && *event.ResourceType == eventAWSCloudFormationStack {
		// We are only interested in events thats `START` a request.
		switch event.ResourceStatus {
		case cloudformation.ResourceStatusCreateInProgress:
			return false
		case cloudformation.ResourceStatusDeleteInProgress:
			return false
		case cloudformation.ResourceStatusUpdateInProgress:
			return false
		default:
			return true
		}
	}
	return true
}

func (r *reportStackEvent) handle(event cloudformation.StackEvent) {
	r.callback(event)
}
