// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"fmt"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/google/go-cmp/cmp"

	"github.com/menderesk/beats/v7/libbeat/logp"
	"github.com/menderesk/beats/v7/x-pack/osquerybeat/internal/ecs"
	"github.com/menderesk/beats/v7/x-pack/osquerybeat/internal/osqdcli"
)

type mockExecutor struct {
	result []map[string]interface{}
	err    error

	receivedSql string
}

func (e *mockExecutor) Query(ctx context.Context, sql string) ([]map[string]interface{}, error) {
	e.receivedSql = sql

	return e.result, e.err
}

type mockPublisher struct {
	index      string
	actionID   string
	responseID string
	hits       []map[string]interface{}
	ecsm       ecs.Mapping
	reqData    interface{}
}

func (p *mockPublisher) Publish(index, actionID, responseID string, hits []map[string]interface{}, ecsm ecs.Mapping, reqData interface{}) {
	p.index = index
	p.actionID = actionID
	p.responseID = responseID
	p.hits = hits
	p.ecsm = ecsm
	p.reqData = reqData
}

func TestActionHandlerExecute(t *testing.T) {
	validLogger := logp.NewLogger("action_test")
	inputType := osqueryInputType

	ctx := context.Background()

	actionID := uuid.Must(uuid.NewV4()).String()
	actionSQL := "select * from uptime"
	request := map[string]interface{}{
		"id": actionID,
		"data": map[string]interface{}{
			"query": actionSQL,
		},
	}

	tests := []struct {
		Name          string
		QueryExecutor queryExecutor
		Publisher     publisher

		Request map[string]interface{}
		Err     error
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
				} else {
					diff := cmp.Diff(tc.QueryExecutor.(*mockExecutor).receivedSql, actionSQL)
					if diff != "" {
						t.Error(diff)
					}

					diff = cmp.Diff(actionID, tc.Publisher.(*mockPublisher).actionID)
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

			fmt.Println(res)
			_ = res
		})
	}
}
