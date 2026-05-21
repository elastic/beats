// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package otelmanager

import (
	"testing"

	"github.com/elastic/beats/v7/libbeat/management"
)

func TestNewOtelManagerSetsUnderAgentAndEnabled(t *testing.T) {
	prev := management.UnderAgent()
	management.SetUnderAgent(false)
	defer management.SetUnderAgent(prev)

	mgr, err := NewOtelManager(nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error creating otel manager: %v", err)
	}

	if !management.UnderAgent() {
		t.Fatal("expected under agent to be enabled")
	}

	if !mgr.Enabled() {
		t.Fatal("expected otel manager to report enabled")
	}
}
