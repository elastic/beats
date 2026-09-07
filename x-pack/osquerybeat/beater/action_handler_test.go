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
	index          string
	idValue        string
	idFieldKey     string
	responseID     string
	spaceID        string
	packID         string
	packName       string
	queryName      string
	meta           map[string]any
	hits           []map[string]any
	ecsm           ecs.Mapping
	reqData        any
	profile        map[string]any
	profileSpaceID string
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

func (p *mockPublisher) PublishQueryProfile(index, queryName, actionID, responseID, spaceID string, profile map[string]any, reqData any) {
	p.profile = profile
	p.profileSpaceID = spaceID
}

func TestActionHandlerExecute(t *testing.T) {
	validLogger := logptest.NewTestingLogger(t, t.Name())
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

func TestActionHandlerExecuteSpaceID(t *testing.T) {
	// collectRuntimeSnapshot issues its own Query calls, so the executor must
	// return non-empty results or the snapshot will fail and no profile is collected.
	osqueryInfoRow := map[string]any{
		"pid":           "1",
		"resident_size": "1000",
		"user_time":     "10",
		"system_time":   "5",
	}
	exec := &mockExecutor{result: []map[string]any{osqueryInfoRow}}
	pub := &mockPublisher{}

	id, err := uuid.NewV4()
	if err != nil {
		t.Fatal(err)
	}
	req := map[string]any{
		"id":       id.String(),
		"space_id": "my-space",
		"data": map[string]any{
			"query":   "select * from uptime",
			"profile": true,
		},
	}

	ac := &actionHandler{
		log:       logptest.NewTestingLogger(t, t.Name()),
		inputType: osqueryInputType,
		queryExec: exec,
		publisher: pub,
	}

	if _, err := ac.Execute(t.Context(), req); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if pub.spaceID != "my-space" {
		t.Errorf("Publish spaceID = %q; want %q", pub.spaceID, "my-space")
	}
	if pub.profileSpaceID != "my-space" {
		t.Errorf("PublishQueryProfile spaceID = %q; want %q", pub.profileSpaceID, "my-space")
	}
}
