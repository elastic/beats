// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package streaming

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"golang.org/x/oauth2/clientcredentials"

	inputcursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/transport/httpcommon"
)

type falconHoseStream struct {
	processor

	id     string
	cfg    config
	cursor map[string]any

	creds       *clientcredentials.Config
	discoverURL string
	plainClient *http.Client

	time func() time.Time
}

// NewFalconHoseFollower performs environment construction including CEL
// program and regexp compilation, and input metrics set-up for a Crowdstrike
// FalconHose stream follower.
func NewFalconHoseFollower(ctx context.Context, id string, cfg config, cursor map[string]any, pub inputcursor.Publisher, log *logp.Logger, now func() time.Time) (StreamFollower, error) {
	s := falconHoseStream{
		id:     id,
		cfg:    cfg,
		cursor: cursor,
		processor: processor{
			ns:      "falcon_hose",
			pub:     pub,
			log:     log,
			redact:  cfg.Redact,
			metrics: newInputMetrics(id, nil),
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
		s.Close()
		return nil, err
	}

	s.prg, s.ast, err = newProgram(ctx, cfg.Program, root, patterns, log)
	if err != nil {
		s.metrics.errorsTotal.Inc()
		s.Close()
		return nil, err
	}

	u, err := url.Parse(s.cfg.URL.String())
	if err != nil {
		return nil, fmt.Errorf("failed parse url: %w", err)
	}
	query := url.Values{"appId": []string{cfg.CrowdstrikeAppID}}
	u.RawQuery = query.Encode()
	s.discoverURL = u.String()

	s.plainClient, err = cfg.Transport.Client(httpcommon.WithAPMHTTPInstrumentation())
	if err != nil {
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

	cli := s.creds.Client(ctx)
	// Normally we would not bother with this, but since connections
	// are in keep-alive in normal operation, let's clean up.
	defer cli.CloseIdleConnections()

	var err error
	for {
		state, err = s.followSession(ctx, cli, state)
		if err != nil {
			if !errors.Is(err, Warning{}) {
				if errors.Is(err, context.Canceled) {
					return nil
				}
				s.metrics.errorsTotal.Inc()
				return err
			}
			s.metrics.errorsTotal.Inc()
			s.log.Warnw("session warning", "error", err)
		}
	}
}

func (s *falconHoseStream) followSession(ctx context.Context, cli *http.Client, state map[string]any) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.discoverURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare discover stream request: %w", err)
	}
	resp, err := cli.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed GET to discover stream: %w", err)
	}
	defer resp.Body.Close()

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
		return state, Warning{fmt.Errorf("failed to decode discover body: %w", err)}
	}
	s.log.Debugw("stream discover metadata", logp.Namespace(s.ns), "meta", mapstr.M(body.Meta))

	var offset int
	if cursor, ok := state["cursor"].(map[string]any); ok {
		switch off := cursor["offset"].(type) {
		case int:
			offset = off
		case float64:
			offset = int(off)
		}
	}

	for _, r := range body.Resources {
		refreshAfter := time.Duration(r.RefreshAfter) * time.Second
		go func() {
			const grace = 5 * time.Minute
			for {
				select {
				case <-ctx.Done():
					return
				case <-time.After(refreshAfter - grace):
					s.log.Debugw("session refresh", "url", r.RefreshURL)
					req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.RefreshURL, nil)
					if err != nil {
						s.metrics.errorsTotal.Inc()
						s.log.Errorw("failed to prepare refresh stream request", "error", err)
						return
					}
					req.Header.Set("Content-Type", "application/json")
					resp, err := cli.Do(req)
					if err != nil {
						s.metrics.errorsTotal.Inc()
						s.log.Errorw("failed to refresh stream connection", "error", err)
						return
					}
					err = resp.Body.Close()
					if err != nil {
						s.metrics.errorsTotal.Inc()
						s.log.Warnw("failed to close refresh response body", "error", err)
					}
				}
			}
		}()

		if offset > 0 {
			feedURL, err := url.Parse(r.FeedURL)
			if err != nil {
				return state, Warning{fmt.Errorf("failed to parse feed url: %w", err)}
			}
			feedQuery, err := url.ParseQuery(feedURL.RawQuery)
			if err != nil {
				return state, Warning{fmt.Errorf("failed to parse feed query: %w", err)}
			}
			feedQuery.Set("offset", strconv.Itoa(offset))
			feedURL.RawQuery = feedQuery.Encode()
			r.FeedURL = feedURL.String()
		}

		s.log.Debugw("stream request", "url", r.FeedURL)
		req, err := http.NewRequestWithContext(ctx, "GET", r.FeedURL, nil)
		if err != nil {
			return state, Warning{fmt.Errorf("failed to make firehose request to %s: %w", r.FeedURL, err)}
		}
		req.Header = make(http.Header)
		req.Header.Add("Accept", "application/json")
		req.Header.Add("Authorization", "Token "+r.Session.Token)

		resp, err := s.plainClient.Do(req)
		if err != nil {
			return state, Warning{fmt.Errorf("failed to get firehose from %s: %w", r.FeedURL, err)}
		}
		defer resp.Body.Close()

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
				return state, Warning{fmt.Errorf("error decoding event: %w", err)}
			}
			s.metrics.receivedBytesTotal.Add(uint64(len(msg)))
			state["response"] = []byte(msg)
			s.log.Debugw("received firehose message", logp.Namespace(s.ns), "msg", debugMsg(msg))
			err = s.process(ctx, state, s.cursor, s.now().In(time.UTC))
			if err != nil {
				s.log.Errorw("failed to process and publish data", "error", err)
				return nil, err
			}
		}
	}
	return state, nil
}

// Warning is a warning-only error.
type Warning struct {
	error
}

// Is returns true if target is a Warning.
func (e Warning) Is(target error) bool {
	_, ok := target.(Warning)
	return ok
}

func (e Warning) Unwrap() error {
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
	s.metrics.Close()
	return nil
}

type debugMsg []byte

func (b debugMsg) String() string {
	return string(b)
}
