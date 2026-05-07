// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package streaming

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"

	cursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
)

func TestFollowSession_FirehoseHTTPError(t *testing.T) {
	logp.TestingSetup()

	tests := []struct {
		name       string
		statusCode int
		body       string
	}{
		{name: "400_plain_text", statusCode: 400, body: "400 Bad Request"},
		{name: "401_unauthorized", statusCode: 401, body: `{"errors":[{"code":401,"message":"access denied"}]}`},
		{name: "500_internal", statusCode: 500, body: "Internal Server Error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				fmt.Fprint(w, tt.body)
			}))
			defer srv.Close()

			discoverResp := discoverResponse(t, srv.URL+"/firehose")
			discoverSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, discoverResp)
			}))
			defer discoverSrv.Close()

			s := newTestStream(t, discoverSrv.URL, srv.Client())
			state := map[string]any{}
			state, err := s.followSession(context.Background(), discoverSrv.Client(), state)
			if err == nil {
				t.Fatal("expected error from followSession, got nil")
			}
			if !strings.Contains(err.Error(), "unsuccessful firehose request") {
				t.Errorf("expected 'unsuccessful firehose request' error, got: %v", err)
			}
			if state == nil {
				t.Error("expected non-nil state on non-hard error")
			}
		})
	}
}

func TestFollowSession_NonObjectMessage(t *testing.T) {
	logp.TestingSetup()

	validEvent := `{"metadata":{"eventType":"TestEvent","offset":1},"event":{"TestField":"value"}}`

	tests := []struct {
		name          string
		body          string
		wantPublished int
	}{
		{
			name:          "bare_number_skipped",
			body:          "400\n",
			wantPublished: 0,
		},
		{
			name:          "bare_string_skipped",
			body:          `"error"` + "\n",
			wantPublished: 0,
		},
		{
			name:          "array_skipped",
			body:          `[1,2,3]` + "\n",
			wantPublished: 0,
		},
		{
			name:          "non_object_then_valid_event",
			body:          "400\n" + validEvent + "\n",
			wantPublished: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			firehoseSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, tt.body)
			}))
			defer firehoseSrv.Close()

			discoverResp := discoverResponse(t, firehoseSrv.URL+"/firehose")
			discoverSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, discoverResp)
			}))
			defer discoverSrv.Close()

			pub := new(countingPublisher)
			s := newTestStreamWithPublisher(t, discoverSrv.URL, firehoseSrv.Client(), pub)
			state := map[string]any{}
			_, err := s.followSession(context.Background(), discoverSrv.Client(), state)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if pub.published() != tt.wantPublished {
				t.Errorf("expected %d published events, got %d", tt.wantPublished, pub.published())
			}
		})
	}
}

func discoverResponse(t *testing.T, feedURL string) string {
	t.Helper()
	resp := map[string]any{
		"resources": []map[string]any{
			{
				"dataFeedURL": feedURL,
				"sessionToken": map[string]any{
					"token":      "test-token",
					"expiration": "2099-01-01T00:00:00Z",
				},
				"refreshActiveSessionURL":      "http://localhost/refresh",
				"refreshActiveSessionInterval": 1800,
			},
		},
		"meta": map[string]any{},
	}
	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal discover response: %v", err)
	}
	return string(b)
}

func newTestStream(t *testing.T, discoverURL string, firehoseClient *http.Client) *falconHoseStream {
	t.Helper()
	return newTestStreamWithPublisher(t, discoverURL, firehoseClient, new(countingPublisher))
}

func newTestStreamWithPublisher(t *testing.T, discoverURL string, firehoseClient *http.Client, pub cursor.Publisher) *falconHoseStream {
	t.Helper()
	log := logp.L()
	reg := monitoring.NewRegistry()
	m := newInputMetrics(reg, log)

	ctx := context.Background()
	prg, ast, err := newProgram(ctx, `
		state.response.decode_json().as(body, {
			"events": [body],
			?"cursor": body.?metadata.optMap(m, {"offset": m.offset}),
		})
	`, root, nil, log)
	if err != nil {
		t.Fatalf("failed to compile CEL program: %v", err)
	}

	return &falconHoseStream{
		cfg:         config{},
		discoverURL: discoverURL,
		plainClient: firehoseClient,
		status:      noopReporter{},
		processor: processor{
			ns:      "test",
			pub:     pub,
			log:     log,
			metrics: m,
			prg:     prg,
			ast:     ast,
		},
		time: time.Now,
	}
}

type countingPublisher int

func (p *countingPublisher) Publish(beat.Event, any) error {
	*p++
	return nil
}

func (p *countingPublisher) published() int {
	return int(*p)
}
