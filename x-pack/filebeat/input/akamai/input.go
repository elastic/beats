// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// This file was contributed to by generative AI

package akamai

import (
	"context"
	"encoding/json"
	"errors"
	"io"
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

const (
	chainOverlap    = 10 * time.Second
	maxLookback     = 12 * time.Hour
	apiSafetyBuffer = 60 // seconds subtracted from "now" for the `to` parameter
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

	var cur cursor
	if !crsr.IsNew() {
		if err := crsr.Unpack(&cur); err != nil {
			env.UpdateStatus(status.Failed, "failed to unpack cursor: "+err.Error())
			return err
		}
	}

	ctx := ctxtool.FromCanceller(env.Cancelation)

	metrics := newInputMetrics(env.MetricsRegistry, cfg.NumberOfWorkers, log)
	if metrics != nil {
		defer metrics.Close()
		metrics.SetResource(cfg.Resource.URL.String() + siemAPIPath + cfg.ConfigIDs)
	}

	client, err := NewClient(cfg, log, env.MetricsRegistry, WithMetrics(metrics))
	if err != nil {
		env.UpdateStatus(status.Failed, "failed to create client: "+err.Error())
		return err
	}
	defer client.Close()

	poller := &siemPoller{
		cfg:     cfg,
		client:  client,
		log:     log,
		pub:     pub,
		cursor:  cur,
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

// cursor holds the chain-based state for resuming event collection.
type cursor struct {
	ChainFrom        int64     `json:"chain_from,omitempty"`
	ChainTo          int64     `json:"chain_to,omitempty"`
	CaughtUp         bool      `json:"caught_up,omitempty"`
	LastOffset       string    `json:"last_offset,omitempty"`
	OffsetObtainedAt time.Time `json:"offset_obtained_at,omitempty"`
}

// isOffsetStale returns true if the stored offset has exceeded the given TTL.
// Returns false if TTL is zero (disabled) or no offset exists.
func (c *cursor) isOffsetStale(ttl time.Duration) bool {
	if ttl == 0 || c.LastOffset == "" {
		return false
	}
	return !c.OffsetObtainedAt.IsZero() && time.Since(c.OffsetObtainedAt) > ttl
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

// run starts the polling loop.
func (p *siemPoller) run(ctx context.Context) error {
	p.log.Info("starting akamai SIEM poller")

	return timed.Periodic(ctx, p.cfg.Interval, func() error {
		return p.poll(ctx)
	})
}

// poll performs a single polling iteration, fetching pages until the chain is
// drained (events returned < event_limit).
func (p *siemPoller) poll(ctx context.Context) error {
	start := time.Now()
	p.log.Debugw("starting poll iteration",
		"cursor.chain_from", p.cursor.ChainFrom,
		"cursor.chain_to", p.cursor.ChainTo,
		"cursor.caught_up", p.cursor.CaughtUp,
		"cursor.last_offset", p.cursor.LastOffset,
		"cursor.offset_obtained_at", p.cursor.OffsetObtainedAt,
	)

	params := p.buildFetchParams()
	pageCount := 0
	recoveryAttempts := 0

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

		body, err := p.fetchWithTimestampRetry(ctx, params)
		if err != nil {
			if !p.handleFetchError(err, &params, pageCount) {
				return nil
			}
			recoveryAttempts++
			if p.cfg.MaxRecoveryAttempts > 0 && recoveryAttempts >= p.cfg.MaxRecoveryAttempts {
				p.log.Errorw("max recovery attempts reached, ending poll cycle",
					"recovery_attempts", recoveryAttempts,
					"max_recovery_attempts", p.cfg.MaxRecoveryAttempts,
					"cursor.chain_from", p.cursor.ChainFrom,
					"cursor.chain_to", p.cursor.ChainTo,
					"cursor.last_offset", p.cursor.LastOffset,
				)
				p.env.UpdateStatus(status.Degraded, "max recovery attempts reached")
				return nil
			}
			continue
		}
		recoveryAttempts = 0

		eventCount, pageCtx, processErr := p.processPage(ctx, body)
		if processErr != nil {
			p.log.Errorw("failed to process page",
				"error", processErr,
				"page", pageCount,
				"mode", fetchMode(params),
				"cursor.last_offset", p.cursor.LastOffset,
			)
			p.env.UpdateStatus(status.Degraded, "failed to process page: "+processErr.Error())
			return nil
		}

		if eventCount == 0 {
			p.log.Debug("no events received, poll cycle complete")
			break
		}

		p.log.Debugw("received events", "count", eventCount, "page", pageCount)

		// Update cursor with page offset
		if pageCtx.Offset != "" {
			prev := p.cursor.LastOffset
			p.cursor.LastOffset = pageCtx.Offset
			p.cursor.OffsetObtainedAt = time.Now()
			p.log.Debugw("cursor advanced after page",
				"page", pageCount,
				"cursor.previous", prev,
				"cursor.current", p.cursor.LastOffset,
			)
		}

		// Drain detection: fewer events than limit means chain is drained
		if eventCount < p.cfg.EventLimit {
			p.cursor.CaughtUp = true
			p.log.Debugw("received fewer events than limit, chain drained",
				"events", eventCount,
				"limit", p.cfg.EventLimit,
			)
			break
		}

		if pageCtx.Offset == "" {
			p.log.Errorw("missing next offset in paginated response; ending current cycle",
				"page", pageCount,
				"events", eventCount,
			)
			p.env.UpdateStatus(status.Degraded, "missing next offset in paginated response")
			break
		}

		// Continue draining chain with next page
		params.Offset = pageCtx.Offset
		params.From = 0
		params.To = 0
	}

	elapsed := time.Since(start)
	if p.metrics != nil {
		p.metrics.RecordRequestTime(elapsed)
	}
	p.log.Debugw("poll iteration complete",
		"duration", elapsed,
		"pages", pageCount,
		"cursor.last_offset", p.cursor.LastOffset,
		"cursor.caught_up", p.cursor.CaughtUp,
	)

	return nil
}

// handleFetchError processes API errors from a fetch attempt. Returns true if
// the poll loop should continue (recoverable), false if it should stop.
func (p *siemPoller) handleFetchError(err error, params *FetchParams, pageCount int) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		p.log.Errorw("failed to fetch events",
			"error", err,
			"page", pageCount,
			"mode", fetchMode(*params),
			"cursor.chain_from", p.cursor.ChainFrom,
			"cursor.chain_to", p.cursor.ChainTo,
			"cursor.last_offset", p.cursor.LastOffset,
		)
		p.env.UpdateStatus(status.Degraded, "failed to fetch events: "+err.Error())
		return false
	}

	switch {
	case apiErr.IsOffsetOutOfRange():
		p.log.Warnw("416 offset expired; clearing offset for chain replay",
			"status_code", apiErr.StatusCode,
			"page", pageCount,
			"cursor.last_offset", p.cursor.LastOffset,
			"cursor.chain_from", p.cursor.ChainFrom,
			"cursor.chain_to", p.cursor.ChainTo,
			"error.message", apiErr.Detail,
			"error.body", apiErr.Body,
		)
		p.cursor.LastOffset = ""
		p.cursor.OffsetObtainedAt = time.Time{}
		if p.metrics != nil {
			p.metrics.AddOffsetExpired()
			p.metrics.AddCursorDrop()
		}
		*params = p.buildFetchParams()
		return true

	case apiErr.IsInvalidTimestamp():
		p.log.Warnw("invalid timestamp after retries; clearing offset for chain replay",
			"status_code", apiErr.StatusCode,
			"page", pageCount,
			"mode", fetchMode(*params),
			"cursor.last_offset", p.cursor.LastOffset,
			"retry_attempts", p.cfg.InvalidTimestampRetries,
			"error.message", apiErr.Detail,
			"error.body", apiErr.Body,
		)
		p.cursor.LastOffset = ""
		p.cursor.OffsetObtainedAt = time.Time{}
		if p.metrics != nil {
			p.metrics.AddCursorDrop()
		}
		*params = p.buildFetchParams()
		return true

	case apiErr.IsFromTooOld():
		p.log.Warnw("from timestamp too old, delegating to chain replay with clamp",
			"status_code", apiErr.StatusCode,
			"page", pageCount,
			"cursor.chain_from", p.cursor.ChainFrom,
			"cursor.chain_to", p.cursor.ChainTo,
			"error.message", apiErr.Detail,
			"error.body", apiErr.Body,
		)
		if p.metrics != nil {
			p.metrics.AddFromClamped()
		}
		*params = p.buildFetchParams()
		return true

	case apiErr.StatusCode == 400:
		p.log.Errorw("non-recoverable 400 response",
			"status_code", apiErr.StatusCode,
			"page", pageCount,
			"error", apiErr,
			"error.body", apiErr.Body,
		)
		if p.metrics != nil {
			p.metrics.AddAPI400Fatal()
		}
		p.env.UpdateStatus(status.Degraded, "received 400 response from Akamai API: "+apiErr.Error())
		return false

	default:
		p.log.Errorw("failed to fetch events",
			"status_code", apiErr.StatusCode,
			"page", pageCount,
			"error", apiErr,
			"error.body", apiErr.Body,
		)
		p.env.UpdateStatus(status.Degraded, "failed to fetch events: "+apiErr.Error())
		return false
	}
}

func (p *siemPoller) fetchWithTimestampRetry(ctx context.Context, params FetchParams) (io.ReadCloser, error) {
	maxRetries := p.cfg.InvalidTimestampRetries
	var lastErr error
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

		body, err := p.client.FetchResponse(ctx, params)
		if err == nil {
			return body, nil
		}
		lastErr = err

		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.IsInvalidTimestamp() && attempt < maxRetries {
			continue
		}
		return nil, err
	}
	return nil, lastErr
}

// buildFetchParams creates fetch parameters using three-branch chain logic.
func (p *siemPoller) buildFetchParams() FetchParams {
	now := time.Now().Unix()

	params := FetchParams{
		Limit: p.cfg.EventLimit,
	}

	switch {
	case !p.cursor.CaughtUp && p.cursor.LastOffset != "" && !p.cursor.isOffsetStale(p.cfg.OffsetTTL):
		// Branch 1: Chain in progress, offset valid — continue draining.
		params.Offset = p.cursor.LastOffset
		p.log.Debugw("offset-based fetch (chain draining)",
			"offset", params.Offset,
			"chain_from", p.cursor.ChainFrom,
			"chain_to", p.cursor.ChainTo,
		)

	case !p.cursor.CaughtUp && p.cursor.ChainFrom != 0:
		// Branch 2: Chain in progress but offset gone/stale — replay chain window.
		if p.cursor.isOffsetStale(p.cfg.OffsetTTL) {
			p.log.Warnw("offset stale, replaying chain window",
				"offset_age", time.Since(p.cursor.OffsetObtainedAt),
				"ttl", p.cfg.OffsetTTL,
			)
			if p.metrics != nil {
				p.metrics.AddOffsetTTLDrop()
			}
		}
		p.cursor.LastOffset = ""
		p.cursor.OffsetObtainedAt = time.Time{}

		from := p.cursor.ChainFrom - int64(chainOverlap.Seconds())
		earliest := now - int64(maxLookback.Seconds())
		if from < earliest {
			p.log.Warnw("chain_from clamped to max lookback",
				"original_from", p.cursor.ChainFrom,
				"clamped_from", earliest,
			)
			from = earliest
			if p.metrics != nil {
				p.metrics.AddFromClamped()
			}
		}
		params.From = from
		params.To = p.cursor.ChainTo
		p.log.Debugw("time-based fetch (chain replay)",
			"from", params.From,
			"to", params.To,
		)

	default:
		// Branch 3: Caught up or first run — start a new chain.
		var from int64
		if p.cursor.ChainTo != 0 {
			from = p.cursor.ChainTo - int64(chainOverlap.Seconds())
		} else {
			from = now - int64(p.cfg.InitialInterval.Seconds())
		}
		earliest := now - int64(maxLookback.Seconds())
		if from < earliest {
			from = earliest
		}
		to := now - apiSafetyBuffer

		p.cursor.ChainFrom = from
		p.cursor.ChainTo = to
		p.cursor.CaughtUp = false
		p.cursor.LastOffset = ""
		p.cursor.OffsetObtainedAt = time.Time{}

		params.From = from
		params.To = to
		p.log.Debugw("time-based fetch (new chain)",
			"from", params.From,
			"to", params.To,
		)
	}

	return params
}

// processPage streams events from body through a bounded channel to worker
// goroutines. Returns event count and offset context. The body is always closed.
func (p *siemPoller) processPage(ctx context.Context, body io.ReadCloser) (int, offsetContext, error) {
	defer body.Close()

	eventCh := make(chan json.RawMessage, p.cfg.ChannelBufferSize)
	start := time.Now()

	// Preliminary cursor carries chain state so events published before the
	// offset context is known still persist enough state for restart recovery.
	var pageCursorMu sync.Mutex
	var pageCursor interface{} = cursor{
		ChainFrom: p.cursor.ChainFrom,
		ChainTo:   p.cursor.ChainTo,
	}

	getCursor := func() interface{} {
		pageCursorMu.Lock()
		defer pageCursorMu.Unlock()
		return pageCursor
	}

	var publishMu sync.Mutex
	var published, failed uint64
	samples := make([]string, 0, 5)

	// Start workers
	var wg sync.WaitGroup
	workerCount := p.cfg.NumberOfWorkers
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if p.metrics != nil {
				id := p.metrics.BeginWorker()
				defer p.metrics.EndWorker(id)
			}

			for raw := range eventCh {
				select {
				case <-ctx.Done():
					return
				default:
				}

				event := p.createBeatEvent(raw)
				c := getCursor()

				if err := p.pub.Publish(event, c); err != nil {
					publishMu.Lock()
					failed++
					if len(samples) < cap(samples) {
						samples = append(samples, err.Error())
					}
					publishMu.Unlock()
					if p.metrics != nil {
						p.metrics.AddError()
					}
					continue
				}

				publishMu.Lock()
				published++
				publishMu.Unlock()
				if p.metrics != nil {
					p.metrics.AddEventPublished(1)
				}
			}
		}()
	}

	// Producer: stream events from body into channel
	pageCtx, eventCount, streamErr := StreamEvents(ctx, body, eventCh)

	// Set cursor BEFORE closing channel so remaining events carry it
	if pageCtx.Offset != "" {
		pageCursorMu.Lock()
		pageCursor = cursor{
			ChainFrom:        p.cursor.ChainFrom,
			ChainTo:          p.cursor.ChainTo,
			LastOffset:       pageCtx.Offset,
			OffsetObtainedAt: time.Now(),
		}
		pageCursorMu.Unlock()
	}

	close(eventCh)
	wg.Wait()

	if p.metrics != nil {
		p.metrics.AddBatchReceived(eventCount)
		p.metrics.RecordBatchTime(time.Since(start))
		if published > 0 {
			p.metrics.AddBatchPublished()
		}
		if failed > 0 {
			p.metrics.AddPartialPublishFailures(failed)
		}
	}

	logFields := []interface{}{
		"published_events", published,
		"failed_events", failed,
		"duration", time.Since(start),
	}
	if failed > 0 {
		logFields = append(logFields, "error_samples", samples)
		p.log.Warnw("finished page publish with failures", logFields...)
	} else {
		p.log.Infow("finished page publish", logFields...)
	}

	return eventCount, pageCtx, streamErr
}

// createBeatEvent creates a beat.Event from raw JSON with zero-copy passthrough.
// The raw JSON is stored as the message field; field extraction is handled by
// ingest pipelines downstream.
func (p *siemPoller) createBeatEvent(raw json.RawMessage) beat.Event {
	return beat.Event{
		Timestamp: time.Now(),
		Fields: map[string]interface{}{
			"message": string(raw),
		},
	}
}
