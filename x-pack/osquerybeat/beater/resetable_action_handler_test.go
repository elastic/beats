// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-libs/logp"

	"github.com/google/go-cmp/cmp"
)

const testActionHandlerTimeout = 400 * time.Millisecond

type mockActionHandler struct {
	err error
}

func (a *mockActionHandler) Execute(ctx context.Context, req map[string]interface{}) (map[string]interface{}, error) {
	now := time.Now().UTC()
	res := map[string]interface{}{
		"started_at":   now.Format(time.RFC3339Nano),
		"completed_at": now.Format(time.RFC3339Nano),
	}
	// Fire error only if count
	if a.err != nil {
		res["error"] = a.err.Error()
	}

	return res, nil
}

func (a *mockActionHandler) Name() string {
	return "osquery"
}

type mockActionResultPublisher struct {
	actionResult map[string]interface{}
}

func (p *mockActionResultPublisher) PublishActionResult(req map[string]interface{}, res map[string]interface{}) {
}

func TestResetableActionHandler(t *testing.T) {
	ctx, cn := context.WithCancel(context.Background())
	defer cn()

	log := logp.NewLogger("resetable_action_handler_test")

	tests := []struct {
		name   string
		ah     client.Action
		nextAh client.Action
		err    error
	}{
		{
			name: "without action handler",
			err:  errActionHandlerIsNotSet,
		},
		{
			name: "with success action handler",
			ah:   &mockActionHandler{},
		},
		{
			name: "broken pipe followed with timeout",
			ah: &mockActionHandler{
				err: errors.New("write: broken pipe"),
			},
			err: errActionTimeout,
		},
		{
			name: "broken pipe followed success",
			ah: &mockActionHandler{
				err: errors.New("write: broken pipe"),
			},
			nextAh: &mockActionHandler{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pub := &mockActionResultPublisher{}
			rah := newResetableActionHandler(pub, log, resetableActionHandlerWithTimeout(testActionHandlerTimeout))
			defer rah.Clear()

			if tc.ah != nil {
				rah.Attach(tc.ah)
			}

			// To test the next handler reattach
			if tc.nextAh != nil {
				go func() {
					time.Sleep(testActionHandlerTimeout - testActionHandlerTimeout/2)
					rah.Attach(tc.nextAh)
				}()
			}

			res, err := rah.Execute(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			if len(res) == 0 {
				t.Fatal("unexpected nil or empty result map")
			}

			var resErr string
			if v, ok := res["error"]; ok {
				resErr = v.(string)
			}
			if tc.err != nil {
				diff := cmp.Diff(resErr, tc.err.Error())
				if diff != "" {
					t.Error(diff)
				}
			} else {
				if resErr != "" {
					t.Errorf("unexpected error: %v", resErr)
				}
			}

			var wantName string
			if tc.ah != nil {
				wantName = tc.ah.Name()
			}
			gotName := rah.Name()
			diff := cmp.Diff(wantName, gotName)
			if diff != "" {
				t.Error(diff)
			}
		})
	}

}
