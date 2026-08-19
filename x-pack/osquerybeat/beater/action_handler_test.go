// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/google/go-cmp/cmp"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/ecs"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/osqdcli"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

type mockExecutor struct {
	result []map[string]any
	err    error

	receivedSql string
}

func (e *mockExecutor) Query(ctx context.Context, sql string, to time.Duration) ([]map[string]any, error) {
	e.receivedSql = sql

	return e.result, e.err
}

type mockPublisher struct {
	index      string
	idValue    string
	idFieldKey string
	responseID string
	spaceID    string
	packID     string
	packName   string
	queryName  string
	meta       map[string]any
	hits       []map[string]any
	ecsm       ecs.Mapping
	reqData    any
	profile    map[string]any
}

func (p *mockPublisher) Publish(index, idValue, idFieldKey, responseID, spaceID, packID, packName, queryName string, meta map[string]any, hits []map[string]any, ecsm ecs.Mapping, reqData any) {
	p.index = index
	p.idValue = idValue
	p.idFieldKey = idFieldKey
	p.responseID = responseID
	p.spaceID = spaceID
	p.packID = packID
	p.packName = packName
	p.queryName = queryName
	p.meta = meta
	p.hits = hits
	p.ecsm = ecsm
	p.reqData = reqData
}

func (p *mockPublisher) PublishQueryProfile(index, queryName, actionID, responseID string, profile map[string]any, reqData any) {
	p.profile = profile
}

// Kibana stamps the originating space on the action document, and the osquery read
// paths filter results by space_id. A result published with an empty space id is
// invisible from any named space, so assert the value survives the round trip.
func TestActionHandlerExecuteSpaceID(t *testing.T) {
	ctx := context.Background()
	actionID := uuid.Must(uuid.NewV4()).String()

	tests := []struct {
		Name    string
		Request map[string]any
		SpaceID string
	}{
		{
			Name: "space id is forwarded to the publisher",
			Request: map[string]any{
				"id":       actionID,
				"space_id": "my-space",
				"data": map[string]any{
					"query": "select * from uptime",
				},
			},
			SpaceID: "my-space",
		},
		{
			Name: "action without a space id publishes an empty one",
			Request: map[string]any{
				"id": actionID,
				"data": map[string]any{
					"query": "select * from uptime",
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			publisher := &mockPublisher{}
			ac := &actionHandler{
				log:       logptest.NewTestingLogger(t, "action_test"),
				inputType: osqueryInputType,
				queryExec: &mockExecutor{},
				publisher: publisher,
			}

			if _, err := ac.Execute(ctx, tc.Request); err != nil {
				t.Fatal("Unexpected error:", err)
			}

			if diff := cmp.Diff(tc.SpaceID, publisher.spaceID); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestActionHandlerExecute(t *testing.T) {
	validLogger := logp.NewLogger("action_test")
	inputType := osqueryInputType

	ctx := context.Background()

	actionID := uuid.Must(uuid.NewV4()).String()
	actionSQL := "select * from uptime"
	nonMatchingPlatform := "windows"
	if runtime.GOOS == "windows" {
		nonMatchingPlatform = "linux"
	}
	request := map[string]any{
		"id": actionID,
		"data": map[string]any{
			"query": actionSQL,
		},
	}

	tests := []struct {
		Name          string
		QueryExecutor queryExecutor
		Publisher     actionQueryPublisher

		Request map[string]any
		Err     error
		Skipped bool
	}{
		{
			Name:    "no executor",
			Request: request,
			Err:     ErrNoQueryExecutor,
		},
		{
			Name:          "no publisher",
			QueryExecutor: &mockExecutor{},
			Request:       request,
			Err:           ErrNoPublisher,
		},
		{
			Name:          "valid",
			QueryExecutor: &mockExecutor{},
			Publisher:     &mockPublisher{},
			Request:       request,
		},
		{
			Name:          "skips non matching platform",
			QueryExecutor: &mockExecutor{},
			Publisher:     &mockPublisher{},
			Request: map[string]any{
				"id": actionID,
				"data": map[string]any{
					"query":    actionSQL,
					"platform": nonMatchingPlatform,
				},
			},
			Skipped: true,
		},
		{
			Name:          "executor error",
			QueryExecutor: &mockExecutor{err: osqdcli.ErrClientClosed},
			Publisher:     &mockPublisher{},
			Request:       request,
			Err:           osqdcli.ErrClientClosed,
		},
	}

	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			ac := &actionHandler{
				log:       validLogger,
				inputType: inputType,
				queryExec: tc.QueryExecutor,
				publisher: tc.Publisher,
			}

			diff := cmp.Diff(inputType, ac.Name())
			if diff != "" {
				t.Fatal(diff)
			}

			res, err := ac.Execute(ctx, tc.Request)

			// The err here is only needed to comply with Action interface, should always be nil
			if err != nil {
				t.Fatal("Unexpected error:", err)
			}

			if res == nil {
				t.Fatal("Unexpected result: nil")
			}

			errVal, ok := res["error"]

			if tc.Err == nil {
				if ok {
					t.Fatal("Unexpected error:", errVal)
				} else if tc.Skipped {
					diff := cmp.Diff("", tc.QueryExecutor.(*mockExecutor).receivedSql)
					if diff != "" {
						t.Error(diff)
					}

					diff = cmp.Diff("", tc.Publisher.(*mockPublisher).idValue)
					if diff != "" {
						t.Error(diff)
					}

					diff = cmp.Diff(0, res["count"])
					if diff != "" {
						t.Error(diff)
					}
				} else {
					diff := cmp.Diff(tc.QueryExecutor.(*mockExecutor).receivedSql, actionSQL)
					if diff != "" {
						t.Error(diff)
					}

					diff = cmp.Diff(actionID, tc.Publisher.(*mockPublisher).idValue)
					if diff != "" {
						t.Error(diff)
					}
					diff = cmp.Diff("action_id", tc.Publisher.(*mockPublisher).idFieldKey)
					if diff != "" {
						t.Error(diff)
					}
					diff = cmp.Diff("", tc.Publisher.(*mockPublisher).responseID)
					if diff != "" {
						t.Error(diff)
					}
				}
			} else {
				if ok {
					errMsg, ok := errVal.(string)
					if !ok {
						t.Fatal("error message is not a string")
					}
					diff := cmp.Diff(tc.Err.Error(), errMsg)
					if diff != "" {
						t.Fatal(diff)
					}
				} else {
					t.Fatal("Unexpected error, got none in the result")
				}
			}
		})
	}
}
