// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package streaming

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	inputcursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/transport/httpcommon"
)

type falconHoseStream struct {
	processor

	id     string
	cfg    config
	cursor map[string]any

	status status.StatusReporter

	creds         *clientcredentials.Config
	authTransport *rateLimitTransport
	discoverURL   string
	plainClient   *http.Client

	time func() time.Time
}

// refreshSessionWait returns the delay between session refresh attempts.
//
// It targets 90% of the requested interval to provide a buffer for network
// delays and retries. For invalid or extremely short intervals, it enforces a
// minimum delay to prevent tight refresh loops.
func refreshSessionWait(refreshAfter time.Duration) time.Duration {
	// Use a 90% refresh interval (similar to the official gofalcon SDK).
	wait := refreshAfter * 9 / 10

	// Enforce a minimum safety delay to prevent spinning on zero/short intervals.
	if wait < 15*time.Second {
		return 15 * time.Second
	}
	return wait
}

// runRefreshLoopWithAfter runs periodic refresh attempts until the context is
// canceled or refresh returns an error. The after callback is injectable to
// allow deterministic tests without sleeping.
func runRefreshLoopWithAfter(ctx context.Context, wait time.Duration, after func(time.Duration) <-chan time.Time, refresh func() error) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-after(wait):
			if err := refresh(); err != nil {
				return
			}
		}
	}
}

// NewFalconHoseFollower performs environment construction including CEL
// program and regexp compilation, and input metrics set-up for a Crowdstrike
// FalconHose stream follower.
func NewFalconHoseFollower(ctx context.Context, env v2.Context, cfg config, cursor map[string]any, pub inputcursor.Publisher, stat status.StatusReporter, log *logp.Logger, now func() time.Time) (StreamFollower, error) {
	if stat == nil {
		stat = noopReporter{}
	}
	stat.UpdateStatus(status.Configuring, "")
	s := falconHoseStream{
		id:     env.ID,
		cfg:    cfg,
		cursor: cursor,
		status: stat,
		processor: processor{
			ns:      "falcon_hose",
			pub:     pub,
			log:     log,
			redact:  cfg.Redact,
			metrics: newInputMetrics(env.MetricsRegistry, log),
		},
		creds: &clientcredentials.Config{
			ClientID:       cfg.Auth.OAuth2.ClientID,
			ClientSecret:   cfg.Auth.OAuth2.ClientSecret,
			TokenURL:       cfg.Auth.OAuth2.TokenURL,
			Scopes:         cfg.Auth.OAuth2.Scopes,
			EndpointParams: cfg.Auth.OAuth2.EndpointParams,
		},
	}
	s.metrics.url.Set(cfg.URL.String())
	s.metrics.errorsTotal.Set(0)

	patterns, err := regexpsFromConfig(cfg)
	if err != nil {
		s.metrics.errorsTotal.Inc()
		stat.UpdateStatus(status.Failed, "invalid regular expression: "+err.Error())
		s.Close()
		return nil, err
	}

	s.prg, s.ast, err = newProgram(ctx, cfg.Program, root, patterns, log)
	if err != nil {
		s.metrics.errorsTotal.Inc()
		stat.UpdateStatus(status.Failed, err.Error())
		s.Close()
		return nil, err
	}

	u, err := url.Parse(s.cfg.URL.String())
	if err != nil {
		err = fmt.Errorf("failed to parse url: %w", err)
		stat.UpdateStatus(status.Failed, err.Error())
		return nil, err
	}
	query := url.Values{"appId": []string{cfg.CrowdstrikeAppID}}
	u.RawQuery = query.Encode()
	s.discoverURL = u.String()

	// Build the auth transport before zeroing timeouts for the streaming
	// client. The oauth2 token endpoint needs normal request timeouts;
	// moving this after the timeout zeroing will cause auth requests to
	// hang indefinitely.
	authClient, err := cfg.Transport.Client(httpcommon.WithAPMHTTPInstrumentation(), httpcommon.WithLogger(log))
	if err != nil {
		stat.UpdateStatus(status.Failed, "failed to configure auth client: "+err.Error())
		return nil, err
	}
	if now == nil {
		now = time.Now
	}
	s.authTransport = &rateLimitTransport{
		base:     authClient.Transport,
		timeout:  authClient.Timeout,
		maxRetry: 3,
		wait:     60 * time.Second,
		log:      log,
		now:      now,
	}

	cfg.Transport.Timeout = 0
	cfg.Transport.IdleConnTimeout = 0
	s.plainClient, err = cfg.Transport.Client(httpcommon.WithAPMHTTPInstrumentation(), httpcommon.WithLogger(log))
	if err != nil {
		stat.UpdateStatus(status.Failed, "failed to configure client: "+err.Error())
		return nil, err
	}

	return &s, nil
}

// FollowStream receives, processes and publishes events from the subscribed
// FalconHose stream.
func (s *falconHoseStream) FollowStream(ctx context.Context) error {
	state := s.cfg.State
	if state == nil {
		state = make(map[string]any)
	}
	if s.cursor != nil {
		state["cursor"] = s.cursor
	}

	ctx = context.WithValue(ctx, oauth2.HTTPClient, &http.Client{Transport: s.authTransport})
	cli := s.creds.Client(ctx)
	// Normally we would not bother with this, but since connections
	// are in keep-alive in normal operation, let's clean up.
	defer cli.CloseIdleConnections()

	var err error
	attempt := 0
	const maxAttemptsUnconfigured = 10
	for {
		state, err = s.followSession(ctx, cli, state)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				s.status.UpdateStatus(status.Stopping, "")
				return nil
			}
			s.metrics.errorsTotal.Inc()
			if errors.Is(err, hardError{}) {
				return err
			}

			attempt++

			if s.cfg.Retry != nil && !s.cfg.Retry.InfiniteRetries && attempt >= s.cfg.Retry.MaxAttempts {
				return fmt.Errorf("max retry attempts (%d) exceeded: %w", s.cfg.Retry.MaxAttempts, err)
			} else if attempt >= maxAttemptsUnconfigured {
				return fmt.Errorf("max retry attempts (%d unconfigured) exceeded: %w", maxAttemptsUnconfigured, err)
			}

			var waitTime time.Duration
			if s.cfg.Retry != nil {
				waitTime = calculateWaitTime(s.cfg.Retry.WaitMin, s.cfg.Retry.WaitMax, attempt)
			} else {
				s.log.Warnw("no retry configured: using linear back-off")
				waitTime = min(time.Duration(attempt)*time.Second, 30*time.Second)
			}

			var rle *rateLimitError
			if errors.As(err, &rle) && rle.wait > waitTime {
				waitTime = rle.wait
			}

			s.status.UpdateStatus(status.Degraded, err.Error())
			s.log.Warnw("session warning", "error", err, "attempt", attempt, "wait", waitTime.String())

			select {
			case <-ctx.Done():
				return nil
			case <-time.After(waitTime):
			}
			continue
		}

		// Reset for success.
		attempt = 0
		s.status.UpdateStatus(status.Running, "")
	}
}

// followSession collects events from a crowdstrike stream, publishing them as
// they are received. It always returns a valid state value unless the error
// returned is a hardError.
func (s *falconHoseStream) followSession(ctx context.Context, cli *http.Client, state map[string]any) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.discoverURL, nil)
	if err != nil {
		return state, fmt.Errorf("failed to prepare discover stream request: %w", err)
	}
	resp, err := cli.Do(req)
	if err != nil {
		err = fmt.Errorf("failed GET to discover stream: %w", err)
		s.status.UpdateStatus(status.Degraded, err.Error())
		return state, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, resp.Body)
		wait := parseRetryAfter(resp.Header.Get("Retry-After"), 60*time.Second, s.now())
		s.log.Warnw("rate limited by discover endpoint",
			"status_code", resp.StatusCode,
			"body", buf.String(),
			"retry_after", wait,
		)
		return state, &rateLimitError{
			wait: wait,
			err:  fmt.Errorf("rate limited by discover endpoint: %s", resp.Status),
		}
	}
	if resp.StatusCode != http.StatusOK {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, resp.Body)
		s.log.Errorw("unsuccessful request", "status_code", resp.StatusCode, "status", resp.Status, "body", buf.String())
		return state, fmt.Errorf("unsuccessful request: %s: %s", resp.Status, &buf)
	}

	dec := json.NewDecoder(resp.Body)

	type resource struct {
		FeedURL string `json:"dataFeedURL"`
		Session struct {
			Token   string    `json:"token"`
			Expires time.Time `json:"expiration"`
		} `json:"sessionToken"`
		RefreshURL   string `json:"refreshActiveSessionURL"`
		RefreshAfter int    `json:"refreshActiveSessionInterval"`
	}
	var body struct {
		Resources []resource     `json:"resources"`
		Meta      map[string]any `json:"meta"`
	}
	err = dec.Decode(&body)
	if err != nil {
		return state, fmt.Errorf("failed to decode discover body: %w", err)
	}
	s.log.Debugw("stream discover metadata", logp.Namespace(s.ns), "meta", mapstr.M(body.Meta))

	cursors, _ := state["cursor"].(map[string]any)
	// Clean up state feed annotation. This unfortunate code placement
	// is in order to avoid allocating defers in a loop.
	defer delete(state, "feed")
	for _, r := range body.Resources {
		feedName := r.FeedURL // Retain this since we will mutate it to set the offset.
		var offset int
		if cursor, ok := cursors[feedName].(map[string]any); ok {
			switch off := cursor["offset"].(type) {
			case int:
				offset = off
			case float64:
				offset = int(off)
			}
		}
		refreshAfter := time.Duration(r.RefreshAfter) * time.Second
		go func() {
			runRefreshLoopWithAfter(ctx, refreshSessionWait(refreshAfter), time.After, func() error {
				s.log.Debugw("session refresh", "url", r.RefreshURL)
				req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.RefreshURL, nil)
				if err != nil {
					s.metrics.errorsTotal.Inc()
					s.status.UpdateStatus(status.Failed, "failed to prepare refresh stream request: "+err.Error())
					s.log.Errorw("failed to prepare refresh stream request", "error", err)
					return err
				}
				req.Header.Set("Content-Type", "application/json")
				resp, err := cli.Do(req)
				if err != nil {
					s.metrics.errorsTotal.Inc()
					s.status.UpdateStatus(status.Failed, "failed to refresh stream connection: "+err.Error())
					s.log.Errorw("failed to refresh stream connection", "error", err)
					return err
				}
				err = resp.Body.Close()
				if err != nil {
					s.metrics.errorsTotal.Inc()
					s.status.UpdateStatus(status.Failed, "failed to close refresh response body: "+err.Error())
					s.log.Warnw("failed to close refresh response body", "error", err)
				}
				return nil
			})
		}()

		if offset > 0 {
			feedURL, err := url.Parse(r.FeedURL)
			if err != nil {
				return state, fmt.Errorf("failed to parse feed url: %w", err)
			}
			feedQuery, err := url.ParseQuery(feedURL.RawQuery)
			if err != nil {
				return state, fmt.Errorf("failed to parse feed query: %w", err)
			}
			feedQuery.Set("offset", strconv.Itoa(offset))
			feedURL.RawQuery = feedQuery.Encode()
			r.FeedURL = feedURL.String()
		}

		s.log.Debugw("stream request", "url", r.FeedURL)
		req, err := http.NewRequestWithContext(ctx, "GET", r.FeedURL, nil)
		if err != nil {
			return state, fmt.Errorf("failed to make firehose request to %s: %w", r.FeedURL, err)
		}
		req.Header = make(http.Header)
		req.Header.Add("Accept", "application/json")
		req.Header.Add("Authorization", "Token "+r.Session.Token)

		resp, err := s.plainClient.Do(req)
		if err != nil {
			return state, fmt.Errorf("failed to get firehose from %s: %w", r.FeedURL, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			var buf bytes.Buffer
			_, _ = io.Copy(&buf, resp.Body)
			s.log.Errorw("unsuccessful firehose request", "status_code", resp.StatusCode, "status", resp.Status, "body", buf.String())
			return state, fmt.Errorf("unsuccessful firehose request: %s: %s", resp.Status, &buf)
		}

		// Prepare state to understand which feed is being processed.
		// This is cleared by the deferred delete above the loop.
		state["feed"] = feedName
		dec := json.NewDecoder(resp.Body)
		for {
			var msg json.RawMessage
			err := dec.Decode(&msg)
			if err != nil {
				s.metrics.errorsTotal.Inc()
				//nolint:errorlint // will not be a wrapped error here.
				if err == io.EOF {
					s.log.Info("stream ended, restarting")
					return state, nil
				}
				return state, fmt.Errorf("error decoding event: %w", err)
			}
			s.metrics.receivedBytesTotal.Add(uint64(len(msg)))
			if len(msg) == 0 || msg[0] != '{' {
				s.metrics.errorsTotal.Inc()
				s.log.Warnw("skipping non-object message from firehose", logp.Namespace(s.ns), "msg", debugMsg(msg))
				continue
			}
			state["response"] = []byte(msg)
			s.log.Debugw("received firehose message", logp.Namespace(s.ns), "msg", debugMsg(msg))
			currentCursor, ok := state["cursor"].(map[string]any)
			if !ok {
				currentCursor = s.cursor
			}
			newCursor, err := s.process(ctx, state, currentCursor, s.now().In(time.UTC))
			if newCursor != nil {
				state["cursor"] = newCursor
			}
			if err != nil {
				s.log.Errorw("failed to process and publish data", "error", err)
				s.status.UpdateStatus(status.Failed, "failed to process and publish data: "+err.Error())
				// Fail the input so that we do not attempt to progress
				// while dropping data on the floor.
				return nil, hardError{err}
			}
		}
	}
	return state, nil
}

// rateLimitError carries a retry-after duration from a 429 response so
// the session-level retry loop can use it as a minimum wait.
type rateLimitError struct {
	wait time.Duration
	err  error
}

func (e *rateLimitError) Error() string { return e.err.Error() }
func (e *rateLimitError) Unwrap() error { return e.err }

// hardError is an input-terminating error.
type hardError struct {
	error
}

// Is returns true if target is a hardError.
func (e hardError) Is(target error) bool {
	_, ok := target.(hardError)
	return ok
}

func (e hardError) Unwrap() error {
	return e.error
}

// now is time.Now with a modifiable time source.
func (s *falconHoseStream) now() time.Time {
	if s.time == nil {
		return time.Now()
	}
	return s.time()
}

func (s *falconHoseStream) Close() error {
	return nil
}

type debugMsg []byte

func (b debugMsg) String() string {
	return string(b)
}
