// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// This file was contributed to by generative AI

package akamai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	inputcursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/go-concert/ctxtool"
	"github.com/elastic/go-concert/timed"
)

// Plugin creates the akamai input plugin.
func Plugin(log *logp.Logger, store statestore.States) v2.Plugin {
	return v2.Plugin{
		Name:       inputName,
		Stability:  feature.Beta,
		Deprecated: false,
		Info:       "Akamai SIEM API Input",
		Doc:        "Collect security events from Akamai SIEM API",
		Manager:    NewInputManager(log, store),
	}
}

// input implements the akamai input.
type input struct{}

func (input) Name() string { return inputName }

func (input) Test(src inputcursor.Source, _ v2.TestContext) error {
	cfg := src.(*source).cfg //nolint:errcheck // If this assertion fails, the program is incorrect and should panic.

	// Validate that we can build a URL
	if cfg.Resource == nil || cfg.Resource.URL == nil || cfg.Resource.URL.URL == nil {
		return errors.New("resource.url is required")
	}

	return nil
}

// Run starts the input and blocks until it completes.
func (input) Run(env v2.Context, src inputcursor.Source, crsr inputcursor.Cursor, pub inputcursor.Publisher) error {
	cfg := src.(*source).cfg //nolint:errcheck // If this assertion fails, the program is incorrect and should panic.
	log := env.Logger.With("input_url", cfg.Resource.URL.String())

	env.UpdateStatus(status.Starting, "")

	// Unpack cursor
	var cursor cursor
	if !crsr.IsNew() {
		if err := crsr.Unpack(&cursor); err != nil {
			env.UpdateStatus(status.Failed, "failed to unpack cursor: "+err.Error())
			return err
		}
	}

	ctx := ctxtool.FromCanceller(env.Cancelation)

	// Initialize metrics
	metrics := newInputMetrics(env.MetricsRegistry, cfg.NumberOfWorkers, log)
	if metrics != nil {
		defer metrics.Close()
		metrics.SetResource(cfg.Resource.URL.String() + siemAPIPath + cfg.ConfigIDs)
	}

	// Create API client
	client, err := NewClient(cfg, log, env.MetricsRegistry, WithMetrics(metrics))
	if err != nil {
		env.UpdateStatus(status.Failed, "failed to create client: "+err.Error())
		return err
	}
	defer client.Close()

	// Create and run the poller
	poller := &siemPoller{
		cfg:     cfg,
		client:  client,
		log:     log,
		pub:     pub,
		cursor:  cursor,
		metrics: metrics,
		env:     env,
	}

	env.UpdateStatus(status.Running, "")
	err = poller.run(ctx)

	if err != nil && !errors.Is(err, context.Canceled) {
		env.UpdateStatus(status.Failed, err.Error())
		return err
	}

	env.UpdateStatus(status.Stopped, "")
	return nil
}

// cursor holds the state for resuming event collection.
type cursor struct {
	LastOffset   string `json:"last_offset,omitempty"`
	RecoveryMode bool   `json:"recovery_mode,omitempty"`
}

// siemPoller handles polling the Akamai SIEM API.
type siemPoller struct {
	cfg     config
	client  *Client
	log     *logp.Logger
	pub     inputcursor.Publisher
	cursor  cursor
	metrics *inputMetrics
	env     v2.Context
}

// eventWork represents work to be processed by a worker.
type eventWork struct {
	index      int
	event      SIEMEvent
	pageCursor interface{}
}

type pagePublishSummary struct {
	published uint64
	failed    uint64
	samples   []string
}

// run starts the polling loop.
func (p *siemPoller) run(ctx context.Context) error {
	p.log.Info("starting akamai SIEM poller")

	return timed.Periodic(ctx, p.cfg.Interval, func() error {
		return p.poll(ctx)
	})
}

// poll performs a single polling iteration.
// It continues fetching pages until the number of returned events is less than
// the event_limit, indicating we've caught up with the available data.
func (p *siemPoller) poll(ctx context.Context) error {
	p.log.Debug("starting poll iteration")
	start := time.Now()
	p.log.Debugw("cursor state before poll",
		"cursor.last_offset", p.cursor.LastOffset,
		"cursor.recovery_mode", p.cursor.RecoveryMode,
	)

	// Determine fetch parameters based on cursor state
	params := p.buildFetchParams()
	pageCount := 0

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		pageCount++
		p.log.Debugw("fetching page",
			"page", pageCount,
			"mode", fetchMode(params),
			"limit", params.Limit,
			"has_offset", params.Offset != "",
		)

		response, err := p.fetchWithTimestampRetry(ctx, params)
		if err != nil {
			var apiErr *APIError
			if errors.As(err, &apiErr) {
				switch {
				case apiErr.IsOffsetOutOfRange():
					p.log.Warnw("received 416 offset expired; dropping cursor and entering recovery mode",
						"cursor.last_offset", p.cursor.LastOffset,
					)
					p.cursor = cursor{RecoveryMode: true}
					if p.metrics != nil {
						p.metrics.AddRecoveryModeEntry()
						p.metrics.AddOffsetExpired()
						p.metrics.AddCursorDrop()
					}
					params = p.buildFetchParams()
					continue
				case apiErr.IsInvalidTimestamp():
					p.log.Warnw("invalid timestamp persisted after retry attempts; dropping cursor and entering recovery mode",
						"cursor.last_offset", p.cursor.LastOffset,
						"retry_attempts", p.cfg.InvalidTimestampRetries,
					)
					p.cursor = cursor{RecoveryMode: true}
					if p.metrics != nil {
						p.metrics.AddRecoveryModeEntry()
						p.metrics.AddCursorDrop()
					}
					params = p.buildFetchParams()
					continue
				case apiErr.StatusCode == 400:
					p.log.Errorw("received non-recoverable 400 response from Akamai API", "error", apiErr)
					if p.metrics != nil {
						p.metrics.AddAPI400Fatal()
					}
					p.env.UpdateStatus(status.Degraded, "received 400 response from Akamai API: "+apiErr.Error())
					return nil
				}
			}
			p.log.Errorw("failed to fetch events", "error", err)
			p.env.UpdateStatus(status.Degraded, "failed to fetch events: "+err.Error())
			return nil // Don't return error to allow retry on next interval
		}

		eventCount := len(response.Events)
		if eventCount == 0 {
			p.log.Debug("no events received, poll cycle complete")
			break
		}

		p.log.Infow("received events", "count", eventCount, "page", pageCount)

		// Process events with workers
		summary, err := p.processEvents(ctx, response)
		if err != nil {
			p.log.Errorw("failed to process events", "error", err)
			p.env.UpdateStatus(status.Degraded, "failed to process events: "+err.Error())
			return nil
		}
		if summary.failed > 0 {
			p.log.Errorw("one or more events failed to publish in page",
				"page", pageCount,
				"failed_events", summary.failed,
				"published_events", summary.published,
				"samples", summary.samples,
			)
		}

		// Update cursor
		if response.LastOffset != "" {
			prev := p.cursor.LastOffset
			p.cursor.LastOffset = response.LastOffset
			p.cursor.RecoveryMode = false
			p.log.Debugw("cursor advanced after page",
				"page", pageCount,
				"cursor.previous", prev,
				"cursor.current", p.cursor.LastOffset,
			)
		}

		// Stop if we received fewer events than the limit - we've caught up
		if eventCount < p.cfg.EventLimit {
			p.log.Debugw("received fewer events than limit, poll cycle complete",
				"events", eventCount,
				"limit", p.cfg.EventLimit,
			)
			break
		}
		if response.LastOffset == "" {
			p.log.Errorw("missing next offset in paginated response; ending current cycle",
				"page", pageCount,
				"events", eventCount,
			)
			p.env.UpdateStatus(status.Degraded, "missing next offset in paginated response")
			break
		}

		// Continue with next page using the last offset
		params.Offset = response.LastOffset
		params.From = 0
		params.To = 0
	}

	p.log.Debugw("poll iteration complete",
		"duration", time.Since(start),
		"pages", pageCount,
		"cursor.last_offset", p.cursor.LastOffset,
		"cursor.recovery_mode", p.cursor.RecoveryMode,
	)

	return nil
}

func (p *siemPoller) fetchWithTimestampRetry(ctx context.Context, params FetchParams) (*SIEMResponse, error) {
	maxRetries := p.cfg.InvalidTimestampRetries
	var err error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			if p.metrics != nil {
				p.metrics.AddHMACRefresh()
			}
			p.log.Debugw("retrying request after invalid timestamp",
				"attempt", attempt,
				"mode", fetchMode(params),
			)
		}

		response, reqErr := p.client.FetchEvents(ctx, params)
		if reqErr == nil {
			return response, nil
		}
		err = reqErr

		var apiErr *APIError
		if errors.As(reqErr, &apiErr) && apiErr.IsInvalidTimestamp() && attempt < maxRetries {
			continue
		}
		return nil, reqErr
	}
	return nil, err
}

// buildFetchParams creates fetch parameters based on cursor state.
func (p *siemPoller) buildFetchParams() FetchParams {
	now := time.Now().Unix()

	params := FetchParams{
		Limit: p.cfg.EventLimit,
	}

	if p.cursor.RecoveryMode {
		// Recovery mode: use time-based fetch
		recoveryDuration := int64(p.cfg.RecoveryInterval.Seconds())
		params.From = now - recoveryDuration
		params.To = now - 60 // 1 minute buffer
		p.log.Debugw("recovery mode fetch", "from", params.From, "to", params.To)
	} else if p.cursor.LastOffset != "" {
		// Continue from last offset
		params.Offset = p.cursor.LastOffset
		p.log.Debugw("offset-based fetch", "offset", params.Offset)
	} else {
		// Initial fetch: use time-based
		initialDuration := int64(p.cfg.InitialInterval.Seconds())
		maxDuration := int64(maxInitialInterval.Seconds())
		if initialDuration > maxDuration {
			initialDuration = maxDuration
		}
		params.From = now - initialDuration
		params.To = now - 60 // 1 minute buffer
		p.log.Debugw("initial time-based fetch", "from", params.From, "to", params.To)
	}

	return params
}

// processEvents processes events using workers.
func (p *siemPoller) processEvents(ctx context.Context, response *SIEMResponse) (pagePublishSummary, error) {
	var summary pagePublishSummary
	if len(response.Events) == 0 {
		return summary, nil
	}

	if p.metrics != nil {
		p.metrics.AddBatchReceived(len(response.Events))
	}

	start := time.Now()

	// Create work channel
	workChan := make(chan eventWork, len(response.Events))

	// Create workers
	var wg sync.WaitGroup
	workerCount := p.cfg.NumberOfWorkers
	if workerCount > len(response.Events) {
		workerCount = len(response.Events)
	}

	// Track published/failed events
	summary.samples = make([]string, 0, 5)
	var publishMu sync.Mutex
	var pageCursor interface{}
	if response.LastOffset != "" {
		pageCursor = cursor{
			LastOffset:   response.LastOffset,
			RecoveryMode: false,
		}
	}

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if p.metrics != nil {
				id := p.metrics.BeginWorker()
				defer p.metrics.EndWorker(id)
			}

			for work := range workChan {
				select {
				case <-ctx.Done():
					return
				default:
				}

				// Create event
				event := p.createBeatEvent(work.event)

				// Publish event
				if err := p.pub.Publish(event, work.pageCursor); err != nil {
					publishMu.Lock()
					summary.failed++
					if len(summary.samples) < cap(summary.samples) {
						summary.samples = append(summary.samples, fmt.Sprintf("index=%d err=%v", work.index, err))
					}
					publishMu.Unlock()
					if p.metrics != nil {
						p.metrics.AddError()
					}
					continue
				}

				publishMu.Lock()
				summary.published++
				publishMu.Unlock()

				if p.metrics != nil {
					p.metrics.AddEventPublished(1)
				}
			}
		}()
	}

	// Send work to workers
	for i, event := range response.Events {
		select {
		case workChan <- eventWork{
			index:      i,
			event:      event,
			pageCursor: pageCursor,
		}:
		case <-ctx.Done():
			close(workChan)
			wg.Wait()
			return summary, ctx.Err()
		}
	}

	close(workChan)
	wg.Wait()

	if p.metrics != nil {
		p.metrics.RecordBatchTime(time.Since(start))
		if summary.published > 0 {
			p.metrics.AddBatchPublished()
		}
		if summary.failed > 0 {
			p.metrics.AddPartialPublishFailures(summary.failed)
		}
	}

	p.log.Infow("finished page publish",
		"published_events", summary.published,
		"failed_events", summary.failed,
		"duration", time.Since(start),
	)

	return summary, nil
}

// createBeatEvent creates a beat.Event from a SIEM event.
func (p *siemPoller) createBeatEvent(event SIEMEvent) beat.Event {
	// Parse the raw JSON into a map
	var fields map[string]interface{}
	if err := json.Unmarshal(event.Raw, &fields); err != nil {
		// If parsing fails, store raw message
		fields = map[string]interface{}{
			"message": string(event.Raw),
		}
	}

	// Add message field if not present (for downstream processing)
	if _, ok := fields["message"]; !ok {
		fields["message"] = string(event.Raw)
	}

	return beat.Event{
		Timestamp: time.Now(),
		Fields:    fields,
	}
}

func fetchMode(params FetchParams) string {
	if params.Offset != "" {
		return "offset"
	}
	return "time"
}
