// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix
// +build !aix

package azureeventhub

import (
	"fmt"
	"testing"
	"time"

	eventhub "github.com/Azure/azure-event-hubs-go/v3"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

func TestInputMetricsEventsReceived(t *testing.T) {

	var sn int64 = 12
	now := time.Now()
	var off int64 = 1234
	var pID int16 = 1

	properties := eventhub.SystemProperties{
		SequenceNumber: &sn,
		EnqueuedTime:   &now,
		Offset:         &off,
		PartitionID:    &pID,
		PartitionKey:   nil,
	}

	log := logp.NewLogger(fmt.Sprintf("%s test for input", inputName))

	//

	cases := []struct {
		// Use case definition
		event              []byte
		expectedRecords    []string
		sanitizationOption []string
		// Expected results
		receivedMessages  uint64
		sanitizedMessages uint64
		processedMessages uint64
		receivedEvents    uint64
		sentEvents        uint64
		processingTime    uint64
		decodeErrors      uint64
		processorRestarts uint64
	}{
		{
			event:             []byte("{\"records\": [{\"test\":\"this is some message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}]}"),
			expectedRecords:   []string{"{\"test\":\"this is some message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}"},
			receivedMessages:  1,
			sanitizedMessages: 0,
			processedMessages: 1,
			receivedEvents:    1,
			sentEvents:        1,
			decodeErrors:      0,
			processorRestarts: 0,
		},
		{
			event: []byte("{\"records\": [{\"test\":\"this is some message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}, {\"test\":\"this is some message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}]}"),
			expectedRecords: []string{
				"{\"test\":\"this is some message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}",
				"{\"test\":\"this is some message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}",
			},
			receivedMessages:  1,
			sanitizedMessages: 0,
			processedMessages: 1,
			receivedEvents:    2,
			sentEvents:        2,
			decodeErrors:      0,
			processorRestarts: 0,
		},
		{
			event: []byte("{\"records\": [{'test':'this is some message','time':'2019-12-17T13:43:44.4946995Z'}]}"), // Thank you, Azure Functions logs.
			expectedRecords: []string{
				"{\"test\":\"this is some message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}",
			},
			sanitizationOption: []string{"SINGLE_QUOTES"},
			receivedMessages:   1,
			sanitizedMessages:  1,
			processedMessages:  1,
			receivedEvents:     1,
			sentEvents:         1,
			decodeErrors:       0,
			processorRestarts:  0,
		},
		{
			event: []byte("{\"records\": [{'test':'this is some message','time':'2019-12-17T13:43:44.4946995Z'}]}"),
			expectedRecords: []string{
				"{\"records\": [{'test':'this is some message','time':'2019-12-17T13:43:44.4946995Z'}]}",
			},
			sanitizationOption: []string{}, // no sanitization options
			receivedMessages:   1,
			sanitizedMessages:  0, // Since we have no sanitization options, we don't try to sanitize.
			processedMessages:  1,
			decodeErrors:       1,
			receivedEvents:     0, // If we can't decode the message, we can't count the events in it.
			sentEvents:         1, // The input sends the unmodified message as a string to the outlet.
			processorRestarts:  0,
		},
	}

	for _, tc := range cases {

		inputConfig := azureInputConfig{
			SAKey:            "",
			SAName:           "",
			SAContainer:      ephContainerName,
			ConnectionString: "",
			ConsumerGroup:    "",
			SanitizeOptions:  tc.sanitizationOption,
		}

		reg := monitoring.NewRegistry()
		metrics := newInputMetrics("test", reg)

		fakeClient := fakeClient{}

		input := eventHubInputV1{
			config:         inputConfig,
			metrics:        metrics,
			pipelineClient: &fakeClient,
			log:            log,
			messageDecoder: messageDecoder{
				config:  inputConfig,
				metrics: metrics,
				log:     log,
			},
		}

		ev := eventhub.Event{
			Data:             tc.event,
			SystemProperties: &properties,
		}

		input.processEvents(&ev)

		if ok := assert.Equal(t, len(tc.expectedRecords), len(fakeClient.publishedEvents)); ok {
			for i, e := range fakeClient.publishedEvents {
				msg, err := e.Fields.GetValue("message")
				if err != nil {
					t.Fatal(err)
				}
				assert.Equal(t, msg, tc.expectedRecords[i])
			}
		}

		assert.True(t, metrics.processingTime.Size() > 0) // TODO: is this the right way of checking if we collected some processing time?

		// Messages
		assert.Equal(t, tc.receivedMessages, metrics.receivedMessages.Get())
		assert.Equal(t, uint64(len(tc.event)), metrics.receivedBytes.Get())
		assert.Equal(t, tc.sanitizedMessages, metrics.sanitizedMessages.Get())
		assert.Equal(t, tc.processedMessages, metrics.processedMessages.Get())

		// General
		assert.Equal(t, tc.decodeErrors, metrics.decodeErrors.Get())

		// Events
		assert.Equal(t, tc.receivedEvents, metrics.receivedEvents.Get())
		assert.Equal(t, tc.sentEvents, metrics.sentEvents.Get())

		// Processor
		assert.Equal(t, tc.processorRestarts, metrics.processorRestarts.Get())

		metrics.Close() // Stop the metrics collection.
	}
}
