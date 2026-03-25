// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix

package azureeventhub

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

func TestInputMetricsEventsReceived(t *testing.T) {
	log := logp.NewLogger("azureeventhub test for input")

	cases := []struct {
		name string
		// Use case definition
		event              []byte
		expectedRecords    []string
		sanitizationOption []string
		// Expected results
		receivedMessages    uint64
		invalidJSONMessages uint64
		sanitizedMessages   uint64
		processedMessages   uint64
		receivedEvents      uint64
		sentEvents          uint64
		decodeErrors        uint64
	}{
		{
			name:                "single valid record",
			event:               []byte("{\"records\": [{\"test\":\"this is some message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}]}"),
			expectedRecords:     []string{"{\"test\":\"this is some message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}"},
			receivedMessages:    1,
			invalidJSONMessages: 0,
			sanitizedMessages:   0,
			processedMessages:   1,
			receivedEvents:      1,
			sentEvents:          1,
			decodeErrors:        0,
		},
		{
			name:  "two valid records",
			event: []byte("{\"records\": [{\"test\":\"this is some message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}, {\"test\":\"this is some message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}]}"),
			expectedRecords: []string{
				"{\"test\":\"this is some message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}",
				"{\"test\":\"this is some message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}",
			},
			receivedMessages:    1,
			invalidJSONMessages: 0,
			sanitizedMessages:   0,
			processedMessages:   1,
			receivedEvents:      2,
			sentEvents:          2,
			decodeErrors:        0,
		},
		{
			name:  "single quotes sanitized",
			event: []byte("{\"records\": [{'test':'this is some message','time':'2019-12-17T13:43:44.4946995Z'}]}"),
			expectedRecords: []string{
				"{\"test\":\"this is some message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}",
			},
			sanitizationOption:  []string{"SINGLE_QUOTES"},
			receivedMessages:    1,
			invalidJSONMessages: 1,
			sanitizedMessages:   1,
			processedMessages:   1,
			receivedEvents:      1,
			sentEvents:          1,
			decodeErrors:        0,
		},
		{
			name:  "invalid JSON without sanitization returns raw message",
			event: []byte("{\"records\": [{'test':'this is some message','time':'2019-12-17T13:43:44.4946995Z'}]}"),
			expectedRecords: []string{
				"{\"records\": [{'test':'this is some message','time':'2019-12-17T13:43:44.4946995Z'}]}",
			},
			sanitizationOption:  []string{},
			receivedMessages:    1,
			invalidJSONMessages: 1,
			sanitizedMessages:   0,
			processedMessages:   1,
			decodeErrors:        1,
			receivedEvents:      0,
			sentEvents:          1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			inputConfig := azureInputConfig{
				SAName:                "",
				SAContainer:           ephContainerName,
				ConnectionString:      "",
				ConsumerGroup:         "",
				LegacySanitizeOptions: tc.sanitizationOption,
			}

			metrics := newInputMetrics(monitoring.NewRegistry(), logp.NewNopLogger())

			client := fakeClient{}

			sanitizers, err := newSanitizers(inputConfig.Sanitizers, inputConfig.LegacySanitizeOptions)
			require.NoError(t, err)

			decoder := messageDecoder{
				config:     inputConfig,
				metrics:    metrics,
				log:        log,
				sanitizers: sanitizers,
			}

			// Simulate the processing pipeline: decode + publish
			metrics.receivedMessages.Inc()
			metrics.receivedBytes.Add(uint64(len(tc.event)))

			records := decoder.Decode(tc.event)
			for _, record := range records {
				event := beat.Event{
					Fields: mapstr.M{
						"message": record,
					},
				}
				client.Publish(event)
			}
			metrics.processedMessages.Inc()
			metrics.sentEvents.Add(uint64(len(records)))

			// Verify published events
			if ok := assert.Equal(t, len(tc.expectedRecords), len(client.publishedEvents)); ok {
				for i, e := range client.publishedEvents {
					msg, err := e.Fields.GetValue("message")
					if err != nil {
						t.Fatal(err)
					}
					assert.Equal(t, msg, tc.expectedRecords[i])
				}
			}

			// Messages
			assert.Equal(t, tc.receivedMessages, metrics.receivedMessages.Get())
			assert.Equal(t, uint64(len(tc.event)), metrics.receivedBytes.Get())
			assert.Equal(t, tc.invalidJSONMessages, metrics.invalidJSONMessages.Get())
			assert.Equal(t, tc.sanitizedMessages, metrics.sanitizedMessages.Get())
			assert.Equal(t, tc.processedMessages, metrics.processedMessages.Get())

			// General
			assert.Equal(t, tc.decodeErrors, metrics.decodeErrors.Get())

			// Events
			assert.Equal(t, tc.receivedEvents, metrics.receivedEvents.Get())
			assert.Equal(t, tc.sentEvents, metrics.sentEvents.Get())
		})
	}
}
