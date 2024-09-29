// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awscloudwatch

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type logProcessor struct {
	log       *logp.Logger
	metrics   *inputMetrics
	publisher beat.Client
}

func newLogProcessor(log *logp.Logger, metrics *inputMetrics, publisher beat.Client, ctx context.Context) *logProcessor {
	if metrics == nil {
		metrics = newInputMetrics("", nil)
	}
	return &logProcessor{
		log:       log,
		metrics:   metrics,
		publisher: publisher,
	}
}

func (p *logProcessor) processLogEvents(logEvents []types.FilteredLogEvent, logGroup string, regionName string) {
	for _, logEvent := range logEvents {
		event := createEvent(logEvent, logGroup, regionName)
		p.metrics.cloudwatchEventsCreatedTotal.Inc()
		p.publisher.Publish(event)
	}
}

func createEvent(logEvent types.FilteredLogEvent, logGroup string, regionName string) beat.Event {
	event := beat.Event{
		Timestamp: time.Unix(*logEvent.Timestamp/1000, 0).UTC(),
		Fields: mapstr.M{
			"message":       *logEvent.Message,
			"log.file.path": logGroup + "/" + *logEvent.LogStreamName,
			"event": mapstr.M{
				"id":       *logEvent.EventId,
				"ingested": time.Now(),
			},
			"awscloudwatch": mapstr.M{
				"log_group":      logGroup,
				"log_stream":     *logEvent.LogStreamName,
				"ingestion_time": time.Unix(*logEvent.IngestionTime/1000, 0),
			},
			"aws.cloudwatch": mapstr.M{
				"log_group":      logGroup,
				"log_stream":     *logEvent.LogStreamName,
				"ingestion_time": time.Unix(*logEvent.IngestionTime/1000, 0),
			},
			"cloud": mapstr.M{
				"provider": "aws",
				"region":   regionName,
			},
		},
	}
	event.SetID(*logEvent.EventId)

	return event
}
