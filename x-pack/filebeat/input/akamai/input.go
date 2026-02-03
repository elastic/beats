// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// This file was contributed to by generative AI

package akamai

import (
	"context"
	"encoding/json"
	"errors"
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
		Stability:  feature.Stable,
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
	if cfg.APIHost == nil || cfg.APIHost.URL == nil {
		return errors.New("api_host is required")
	}

	return nil
}

// Run starts the input and blocks until it completes.
func (input) Run(env v2.Context, src inputcursor.Source, crsr inputcursor.Cursor, pub inputcursor.Publisher) error {
	cfg := src.(*source).cfg //nolint:errcheck // If this assertion fails, the program is incorrect and should panic.
	log := env.Logger.With("input_url", cfg.APIHost.String())

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
		metrics.SetResource(cfg.APIHost.String() + siemAPIPath + cfg.ConfigIDs)
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
	event  SIEMEvent
	cursor interface{}
	isLast bool
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
		p.log.Debugw("fetching events", "page", pageCount, "params", params)

		response, err := p.client.FetchEvents(ctx, params)
		if err != nil {
			if IsRecoverableError(err) {
				p.log.Warnw("recoverable error, entering recovery mode", "error", err)
				if p.metrics != nil {
					p.metrics.AddRecoveryModeEntry()
				}
				p.cursor.RecoveryMode = true
				p.cursor.LastOffset = ""
				params = p.buildFetchParams()
				continue
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
		if err := p.processEvents(ctx, response); err != nil {
			p.log.Errorw("failed to process events", "error", err)
			p.env.UpdateStatus(status.Degraded, "failed to process events: "+err.Error())
			return nil
		}

		// Update cursor
		if response.LastOffset != "" {
			p.cursor.LastOffset = response.LastOffset
			p.cursor.RecoveryMode = false
		}

		// Stop if we received fewer events than the limit - we've caught up
		if eventCount < p.cfg.EventLimit {
			p.log.Debugw("received fewer events than limit, poll cycle complete",
				"events", eventCount,
				"limit", p.cfg.EventLimit,
			)
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
	)

	return nil
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
func (p *siemPoller) processEvents(ctx context.Context, response *SIEMResponse) error {
	if len(response.Events) == 0 {
		return nil
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

	// Track published events
	var publishedCount uint64
	var publishErr error
	var publishMu sync.Mutex

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
				if err := p.pub.Publish(event, work.cursor); err != nil {
					p.log.Errorw("failed to publish event", "error", err)
					publishMu.Lock()
					if publishErr == nil {
						publishErr = err
					}
					publishMu.Unlock()
					if p.metrics != nil {
						p.metrics.AddError()
					}
					continue
				}

				publishMu.Lock()
				publishedCount++
				publishMu.Unlock()

				if p.metrics != nil {
					p.metrics.AddEventPublished(1)
				}
			}
		}()
	}

	// Send work to workers
	for i, event := range response.Events {
		isLast := i == len(response.Events)-1
		var cursorVal interface{}
		if isLast && response.LastOffset != "" {
			cursorVal = cursor{
				LastOffset:   response.LastOffset,
				RecoveryMode: false,
			}
		}

		select {
		case workChan <- eventWork{
			event:  event,
			cursor: cursorVal,
			isLast: isLast,
		}:
		case <-ctx.Done():
			close(workChan)
			wg.Wait()
			return ctx.Err()
		}
	}

	close(workChan)
	wg.Wait()

	if p.metrics != nil {
		p.metrics.RecordBatchTime(time.Since(start))
		if publishedCount > 0 {
			p.metrics.AddBatchPublished()
		}
	}

	p.log.Infow("published events", "count", publishedCount, "duration", time.Since(start))

	return publishErr
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
