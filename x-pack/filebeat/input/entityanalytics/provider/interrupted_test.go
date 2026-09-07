// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package provider

import (
	"context"
	"errors"
	"fmt"
	"testing"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
)

func TestSyncInterrupted(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	inputCtx := v2.Context{Cancelation: ctx}

	if SyncInterrupted(inputCtx, context.Canceled) {
		t.Error("SyncInterrupted = true for context.Canceled before the input is cancelled")
	}
	if SyncInterrupted(inputCtx, errors.New("boom")) {
		t.Error("SyncInterrupted = true for unrelated error before the input is cancelled")
	}

	cancel()

	if !SyncInterrupted(inputCtx, fmt.Errorf("okta: get users: %w", context.Canceled)) {
		t.Error("SyncInterrupted = false for wrapped context.Canceled after the input is cancelled")
	}
	if SyncInterrupted(inputCtx, errors.New("boom")) {
		t.Error("SyncInterrupted = true for unrelated error after the input is cancelled")
	}
	if SyncInterrupted(inputCtx, context.DeadlineExceeded) {
		t.Error("SyncInterrupted = true for context.DeadlineExceeded")
	}
}
