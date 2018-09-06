// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package transformer

import (
	"time"

	"github.com/aws/aws-lambda-go/events"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

// CloudwatchLogs Takes an CloudwatchLogsData and transform it into a beat event.
func CloudwatchLogs(request events.CloudwatchLogsData) []beat.Event {
	events := make([]beat.Event, len(request.LogEvents))

	for idx, logEvent := range request.LogEvents {
		events[idx] = beat.Event{
			Timestamp: time.Unix(logEvent.Timestamp, 0),
			Fields: common.MapStr{
				"message":              logEvent.Message,
				"id":                   logEvent.ID,
				"owner":                request.Owner,
				"log_stream":           request.LogStream,
				"log_group":            request.LogGroup,
				"message_type":         request.MessageType,
				"subscription_filters": request.SubscriptionFilters,
			},
		}
	}

	return events
}
