// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awscloudwatch

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"

	"github.com/elastic/beats/v8/libbeat/beat"
	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/logp"
	"github.com/elastic/beats/v8/libbeat/monitoring"
	awscommon "github.com/elastic/beats/v8/x-pack/libbeat/common/aws"
)

type logProcessor struct {
	log       *logp.Logger
	metrics   *inputMetrics
	publisher beat.Client
	ack       *awscommon.EventACKTracker
}

func newLogProcessor(log *logp.Logger, metrics *inputMetrics, publisher beat.Client, ctx context.Context) *logProcessor {
	if metrics == nil {
		metrics = newInputMetrics(monitoring.NewRegistry(), "")
	}
	return &logProcessor{
		log:       log,
		metrics:   metrics,
		publisher: publisher,
		ack:       awscommon.NewEventACKTracker(ctx),
	}
}

func (p *logProcessor) processLogEvents(logEvents []cloudwatchlogs.FilteredLogEvent, logGroup string, regionName string) error {
	for _, logEvent := range logEvents {
		event := createEvent(logEvent, logGroup, regionName)
		p.publish(p.ack, &event)
	}
	return nil
}

func (p *logProcessor) publish(ack *awscommon.EventACKTracker, event *beat.Event) {
	ack.Add()
	event.Private = ack
	p.metrics.cloudwatchEventsCreatedTotal.Inc()
	p.publisher.Publish(*event)
}

func createEvent(logEvent cloudwatchlogs.FilteredLogEvent, logGroup string, regionName string) beat.Event {
	event := beat.Event{
		Timestamp: time.Unix(*logEvent.Timestamp/1000, 0).UTC(),
		Fields: common.MapStr{
			"message":       *logEvent.Message,
			"log.file.path": logGroup + "/" + *logEvent.LogStreamName,
			"event": common.MapStr{
				"id":       *logEvent.EventId,
				"ingested": time.Now(),
			},
			"awscloudwatch": common.MapStr{
				"log_group":      logGroup,
				"log_stream":     *logEvent.LogStreamName,
				"ingestion_time": time.Unix(*logEvent.IngestionTime/1000, 0),
			},
			"cloud": common.MapStr{
				"provider": "aws",
				"region":   regionName,
			},
		},
	}
	event.SetID(*logEvent.EventId)

	return event
}
