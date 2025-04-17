// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package streaming

import (
	"context"
	"errors"
	"flag"
	"net/url"
	"os"
	"testing"
	"time"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	cursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

var (
	timeout = flag.Duration("crowdstrike_timeout", time.Minute, "time to allow Crowdstrike FalconHose test to run")
	offset  = flag.Int("crowdstrike_offset", -1, "offset into stream (negative to ignore)")
)

func TestCrowdstrikeFalconHose(t *testing.T) {
	logp.TestingSetup()
	logger := logp.L()

	feedURL, ok := os.LookupEnv("CROWDSTRIKE_URL")
	if !ok {
		t.Skip("okta tests require ${CROWDSTRIKE_URL} to be set")
	}
	tokenURL, ok := os.LookupEnv("CROWDSTRIKE_TOKEN_URL")
	if !ok {
		t.Skip("okta tests require ${CROWDSTRIKE_TOKEN_URL} to be set")
	}
	clientID, ok := os.LookupEnv("CROWDSTRIKE_CLIENT_ID")
	if !ok {
		t.Skip("okta tests require ${CROWDSTRIKE_CLIENT_ID} to be set")
	}
	clientSecret, ok := os.LookupEnv("CROWDSTRIKE_CLIENT_SECRET")
	if !ok {
		t.Skip("okta tests require ${CROWDSTRIKE_CLIENT_SECRET} to be set")
	}
	appID, ok := os.LookupEnv("CROWDSTRIKE_APPID")
	if !ok {
		t.Skip("okta tests require ${CROWDSTRIKE_APPID} to be set")
	}

	u, err := url.Parse(feedURL)
	if err != nil {
		t.Fatalf("unexpected error parsing feed url: %v", err)
	}
	cfg := config{
		Type: "crowdstrike",
		URL:  &urlConfig{u},
		Program: `
				state.response.decode_json().as(body,{
					"events": [body],
					?"cursor": has(body.?metadata.offset) ?
						optional.of({"offset": body.metadata.offset})
					:
						optional.none(),
				})`,
		Auth: authConfig{
			OAuth2: oAuth2Config{
				ClientID:     clientID,
				ClientSecret: clientSecret,
				TokenURL:     tokenURL,
			},
		},
		CrowdstrikeAppID: appID,
	}

	err = cfg.Validate()
	if err != nil {
		t.Fatalf("unexpected error validating config: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	var cursor map[string]any
	if *offset >= 0 {
		cursor = map[string]any{"offset": *offset}
	}
	env := v2.Context{ID: "crowdstrike_testing",
		MetricsRegistry: monitoring.NewRegistry()}
	s, err := NewFalconHoseFollower(
		ctx, env, cfg, cursor, &testPublisher{logger}, logger, time.Now)
	if err != nil {
		t.Fatalf("unexpected error constructing follower: %v", err)
	}
	err = s.FollowStream(ctx)
	if errors.Is(err, context.DeadlineExceeded) {
		err = nil
	}
	if err != nil {
		t.Errorf("unexpected error following stream: %v", err)
	}
}

type testPublisher struct {
	log *logp.Logger
}

var _ cursor.Publisher = testPublisher{}

func (p testPublisher) Publish(e beat.Event, cursor any) error {
	p.log.Infow("publish", "event", e.Fields, "cursor", cursor)
	return nil
}
