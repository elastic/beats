// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package otelmode

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/elastic/beats/v7/libbeat/management"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

var otelManagementEnabled atomic.Bool

var _ management.Manager = (*otelManager)(nil)

func SetOtelMode(enabled bool) {
	otelManagementEnabled.Store(enabled)
}

// Enabled() returns true if beatreceiver is running under Elastic Agent
func Enabled() bool {
	return otelManagementEnabled.Load()
}

func NewOtelManager(cfg *config.C) (management.Manager, error) {
	otelManaged := struct {
		Enabled bool `config:"management.otel.enabled"`
	}{}
	if err := cfg.Unpack(&otelManaged); err != nil {
		return nil, fmt.Errorf("failed to unpack config: %w", err)
	}
	if otelManaged.Enabled {
		// the beatreceiver is a part of elastic-agent

		SetOtelMode(true)
		return &otelManager{}, nil
	}
	// not a part of elastic-agent
	return &management.FallbackManager{}, nil
}

// otelManager is the main manager for managing beatreceivers
type otelManager struct {
	logger *logp.Logger
	lock   sync.Mutex
	status status.Status
	msg    string
}

func (n *otelManager) UpdateStatus(status status.Status, msg string) {
	n.lock.Lock()
	defer n.lock.Unlock()
	if n.status != status || n.msg != msg {
		n.status = status
		n.msg = msg
		n.logger.Infof("Status change to %s: %s", status, msg)
	}
}

func (n *otelManager) SetStopCallback(func()) {
}

func (n *otelManager) Stop() {}

// Enabled returns false because many places inside beats call manager.Enabled() for various purposes
// Returning true might lead to side effects.
func (n *otelManager) Enabled() bool                         { return false }
func (n *otelManager) AgentInfo() client.AgentInfo           { return client.AgentInfo{} }
func (n *otelManager) Start() error                          { return nil }
func (n *otelManager) CheckRawConfig(cfg *config.C) error    { return nil }
func (n *otelManager) RegisterAction(action client.Action)   {}
func (n *otelManager) UnregisterAction(action client.Action) {}
func (n *otelManager) SetPayload(map[string]interface{})     {}
func (n *otelManager) RegisterDiagnosticHook(_ string, _ string, _ string, _ string, _ client.DiagnosticHook) {
}
