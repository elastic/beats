// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sqs/types"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/elastic-agent-libs/logp"
)

// sqsDiscoveryV2 implements the SQS receive loop and notification parsing for
// the V2 input. It delegates S3 object processing to objectProcessorV2.
type sqsDiscoveryV2 struct {
	sqs               sqsAPI
	s3Move            s3Mover
	queueURL          string
	visibilityTimeout time.Duration
	maxReceiveCount   int
	script            *script
	processor         *objectProcessorV2
	metrics           *inputMetrics
	log               *logp.Logger
	status            status.StatusReporter
	warnOnce          sync.Once
}

// sqsDiscoveryV2Config holds the parameters for creating V2 SQS discovery.
type sqsDiscoveryV2Config struct {
	SQS               sqsAPI
	S3Move            s3Mover
	QueueURL          string
	VisibilityTimeout time.Duration
	MaxReceiveCount   int
	Script            *script
	Processor         *objectProcessorV2
	Metrics           *inputMetrics
	Log               *logp.Logger
	Status            status.StatusReporter
}

func newSQSDiscoveryV2(cfg sqsDiscoveryV2Config) *sqsDiscoveryV2 {
	return &sqsDiscoveryV2{
		sqs:               cfg.SQS,
		s3Move:            cfg.S3Move,
		queueURL:          cfg.QueueURL,
		visibilityTimeout: cfg.VisibilityTimeout,
		maxReceiveCount:   cfg.MaxReceiveCount,
		script:            cfg.Script,
		processor:         cfg.Processor,
		metrics:           cfg.Metrics,
		log:               cfg.Log,
		status:            cfg.Status,
	}
}

// sqsResultV2 captures the outcome of processing a single SQS message.
type sqsResultV2 struct {
	disc         *sqsDiscoveryV2
	msg          *types.Message
	receiveCount int
	eventCount   int
	cancelKeep   context.CancelFunc
	err          error
	finalizers   []func() error
}

// ProcessMessage parses an SQS message, processes each S3 object notification,
// and returns a result that must be finalized via Done() after all events are
// ACKed.
func (d *sqsDiscoveryV2) ProcessMessage(ctx context.Context, msg *types.Message, pub func(beat.Event)) sqsResultV2 {
	log := d.log.With("message_id", *msg.MessageId, "message_receipt_time", time.Now().UTC())

	keepCtx, keepCancel := context.WithCancel(ctx)
	defer keepCancel()

	var keepWg sync.WaitGroup
	keepWg.Add(1)
	go func() {
		defer keepWg.Done()
		d.keepalive(keepCtx, log, msg)
	}()

	receiveCount := getSQSReceiveCount(msg.Attributes)
	if receiveCount == 1 {
		if s, found := msg.Attributes[sqsSentTimestampAttribute]; found {
			if millis, err := strconv.ParseInt(s, 10, 64); err == nil {
				d.metrics.sqsLagTime.Update(time.Since(time.UnixMilli(millis)).Nanoseconds())
			}
		}
	}

	eventCount := 0
	finalizers, procErr := d.processNotifications(ctx, log, *msg.Body, func(e beat.Event) {
		eventCount++
		pub(e)
	})

	return sqsResultV2{
		disc:         d,
		msg:          msg,
		receiveCount: receiveCount,
		eventCount:   eventCount,
		cancelKeep:   keepCancel,
		err:          procErr,
		finalizers:   finalizers,
	}
}

// Done finalizes the SQS message: stops keepalive, decides disposition
// (delete or return), and runs S3 finalizers on success.
func (r sqsResultV2) Done() {
	d := r.disc
	r.cancelKeep()

	if r.err == nil {
		if err := d.sqs.DeleteMessage(context.Background(), r.msg); err != nil {
			d.log.Errorf("failed deleting SQS message (may be reprocessed): %v", err)
			d.status.UpdateStatus(status.Degraded, fmt.Sprintf("SQS delete failed: %s", err))
			return
		}
		d.metrics.sqsMessagesDeletedTotal.Inc()
		for _, fin := range r.finalizers {
			if err := fin(); err != nil {
				d.log.Errorf("S3 finalization failed (manual cleanup required): %v", err)
				d.status.UpdateStatus(status.Degraded, fmt.Sprintf("S3 finalization failed: %s", err))
			}
		}
		return
	}

	procErr := r.err
	if d.maxReceiveCount > 0 && r.receiveCount >= d.maxReceiveCount {
		procErr = nonRetryableErrorWrap(fmt.Errorf(
			"sqs ApproximateReceiveCount <%v> exceeds threshold %v: %w",
			r.receiveCount, d.maxReceiveCount, procErr))
	}

	if errors.Is(procErr, &nonRetryableError{}) {
		if err := d.sqs.DeleteMessage(context.Background(), r.msg); err != nil {
			d.log.Errorf("failed deleting non-retryable SQS message: %v", err)
			d.status.UpdateStatus(status.Degraded, fmt.Sprintf("SQS delete failed for non-retryable message: %s", err))
			return
		}
		d.metrics.sqsMessagesDeletedTotal.Inc()
		d.log.Warnf("deleted non-retryable SQS message: %v", procErr)
		return
	}

	d.metrics.sqsMessagesReturnedTotal.Inc()
	d.log.Warnf("SQS message will return after visibility timeout: %v", procErr)
	d.status.UpdateStatus(status.Degraded, fmt.Sprintf("SQS processing failed (will retry): %s", procErr))
}

// keepalive extends the SQS message visibility timeout periodically.
func (d *sqsDiscoveryV2) keepalive(ctx context.Context, log *logp.Logger, msg *types.Message) {
	t := time.NewTicker(d.visibilityTimeout / 2)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			d.metrics.sqsVisibilityTimeoutExtensionsTotal.Inc()
			if err := d.sqs.ChangeMessageVisibility(ctx, msg, d.visibilityTimeout); err != nil {
				log.Warnw("Failed to extend SQS visibility timeout.", "error", err)
				return
			}
		}
	}
}

// processNotifications parses the SQS body into S3 events and processes each.
func (d *sqsDiscoveryV2) processNotifications(ctx context.Context, log *logp.Logger, body string, pub func(beat.Event)) ([]func() error, error) {
	s3Events, err := d.getS3Notifications(body)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, err
		}
		return nil, &nonRetryableError{Err: err}
	}
	if len(s3Events) == 0 {
		return nil, nil
	}
	log.Debugf("SQS message contained %d S3 event notifications.", len(s3Events))

	var errs []error
	var finalizers []func() error
	for i, evt := range s3Events {
		n, procErr := d.processor.ProcessObject(ctx, log, evt, pub)
		if procErr != nil {
			errs = append(errs, fmt.Errorf(
				"failed processing S3 object %q in bucket %q (record %d of %d): %w",
				evt.S3.Object.Key, evt.S3.Bucket.Name, i+1, len(s3Events), procErr))
			d.status.UpdateStatus(status.Degraded, fmt.Sprintf("S3 processing failure: %s", procErr))
			continue
		}
		if n > 0 {
			finalizers = append(finalizers, func() error {
				return d.processor.Finalize(ctx, d.s3Move, evt)
			})
		}
	}
	return finalizers, errors.Join(errs...)
}

// getS3Notifications parses the SQS body into S3 event references.
// Notification format auto-detection order: custom script, S3 v2.2, SNS
// wrapper, EventBridge.
func (d *sqsDiscoveryV2) getS3Notifications(body string) ([]s3EventV2, error) {
	if d.script != nil {
		return d.script.run(body)
	}

	var events s3EventsV2
	if err := json.NewDecoder(strings.NewReader(body)).Decode(&events); err != nil {
		return nil, fmt.Errorf("failed to decode SQS body as S3 notification: %w", err)
	}

	if events.Event == "s3:TestEvent" {
		return nil, nil
	}

	if events.TopicArn != "" {
		var inner s3EventsV2
		if err := json.NewDecoder(strings.NewReader(events.Message)).Decode(&inner); err != nil {
			return nil, fmt.Errorf("failed to decode SNS-wrapped S3 notification: %w", err)
		}
		if inner.Event == "s3:TestEvent" {
			return nil, nil
		}
		events = inner
	}

	if events.Records == nil {
		var eb eventBridgeEvent
		if err := json.NewDecoder(strings.NewReader(body)).Decode(&eb); err == nil {
			convertEventBridge(&eb, &events)
		}
	}

	if events.Records == nil {
		return nil, errors.New("invalid S3 notification: missing Records field")
	}

	return d.filterRecords(events.Records)
}

// filterRecords retains only ObjectCreated events and unescapes keys.
func (d *sqsDiscoveryV2) filterRecords(records []s3EventV2) ([]s3EventV2, error) {
	out := make([]s3EventV2, 0, len(records))
	for _, rec := range records {
		if rec.EventSource != "aws:s3" || !strings.HasPrefix(rec.EventName, "ObjectCreated:") {
			d.warnOnce.Do(func() {
				d.log.Warnf("Received S3 notification for %q event type; only ObjectCreated:* events are processed.", rec.EventName)
			})
			continue
		}
		key, err := url.QueryUnescape(rec.S3.Object.Key)
		if err != nil {
			return nil, fmt.Errorf("url unescape failed for %q: %w", rec.S3.Object.Key, err)
		}
		rec.S3.Object.Key = key
		out = append(out, rec)
	}
	return out, nil
}

// ReceiveLoop is the main SQS receive loop. It blocks until ctx is cancelled.
// For each received message, it calls handle which is responsible for
// processing and finalizing the message.
func (d *sqsDiscoveryV2) ReceiveLoop(ctx context.Context, maxMessages int, handle func(context.Context, types.Message)) {
	for ctx.Err() == nil {
		msgs := readSQSMessages(ctx, d.log, d.status, d.sqs, d.metrics, maxMessages, d.queueURL)
		for _, msg := range msgs {
			if ctx.Err() != nil {
				return
			}
			handle(ctx, msg)
		}
	}
}
