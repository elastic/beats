// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package otelmanager

import (
	"sync"

	"github.com/elastic/beats/v7/libbeat/common/reload"
	"github.com/elastic/beats/v7/libbeat/management"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

var _ management.Manager = (*OtelManager)(nil)

func NewOtelManager(cfg *config.C, registry *reload.Registry, logger *logp.Logger) (management.Manager, error) {
	management.SetUnderAgent(true)
	return &OtelManager{}, nil
}

// OtelManager is the main manager for managing beatreceivers
type OtelManager struct {
	logger *logp.Logger
	lock   sync.Mutex
	status status.Status
	msg    string
}

func (n *OtelManager) UpdateStatus(status status.Status, msg string) {
	n.lock.Lock()
	defer n.lock.Unlock()
	if n.status != status || n.msg != msg {
		n.status = status
		n.msg = msg
		n.logger.Infof("Status changed to %s: %s", status, msg)
	}
}

func (n *OtelManager) SetStopCallback(func()) {
}

func (n *OtelManager) Stop() {}

// Enabled returns false because many places inside beats call manager.Enabled() for various purposes
// Returning true might lead to side effects.
func (n *OtelManager) Enabled() bool                         { return false }
func (n *OtelManager) AgentInfo() client.AgentInfo           { return client.AgentInfo{} }
func (n *OtelManager) Start() error                          { return nil }
func (n *OtelManager) CheckRawConfig(cfg *config.C) error    { return nil }
func (n *OtelManager) RegisterAction(action client.Action)   {}
func (n *OtelManager) UnregisterAction(action client.Action) {}
func (n *OtelManager) SetPayload(map[string]interface{})     {}
func (n *OtelManager) RegisterDiagnosticHook(_ string, _ string, _ string, _ string, _ client.DiagnosticHook) {
}
