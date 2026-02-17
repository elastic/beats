// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package otelmanager

import (
	"sync"

	"github.com/elastic/beats/v7/libbeat/common/reload"
	"github.com/elastic/beats/v7/libbeat/management"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

type DiagnosticExtension interface {
	RegisterDiagnosticHook(name string, description string, filename string, contentType string, hook func() []byte)
}

type WithDiagnosticExtension interface {
	// name is the beat name
	// ext is the extension that implements the DiagnosticExtension interface
	SetDiagnosticExtension(name string, ext DiagnosticExtension)
}

var _ management.Manager = (*OtelManager)(nil)
var _ WithDiagnosticExtension = (*OtelManager)(nil)

func NewOtelManager(cfg *config.C, registry *reload.Registry, logger *logp.Logger) (management.Manager, error) {
	management.SetUnderAgent(true)
	return &OtelManager{}, nil
}

// OtelManager is the main manager for managing beatreceivers
type OtelManager struct {
	ext          DiagnosticExtension
	receiverName string
	stopFn       func()
	stopOnce     sync.Once
}

func (n *OtelManager) UpdateStatus(_ status.Status, _ string) {
	// a stub implemtation for now.
	// TODO(@VihasMakwana): Explore the option to tidy and refactor the status reporting for beatsreceivers.
}

func (n *OtelManager) SetStopCallback(fn func()) {
	n.stopFn = fn
}

func (n *OtelManager) Stop() {
	if n.stopFn != nil {
		n.stopOnce.Do(n.stopFn)
	}
}

// Enabled returns false because many places inside beats call manager.Enabled() for various purposes
// Returning true might lead to side effects.
func (n *OtelManager) Enabled() bool                             { return false }
func (n *OtelManager) AgentInfo() management.AgentInfo           { return management.AgentInfo{} }
func (n *OtelManager) Start() error                              { return nil }
func (n *OtelManager) CheckRawConfig(cfg *config.C) error        { return nil }
func (n *OtelManager) RegisterAction(action management.Action)   {}
func (n *OtelManager) UnregisterAction(action management.Action) {}
func (n *OtelManager) SetPayload(map[string]interface{})         {}
func (n *OtelManager) RegisterDiagnosticHook(_ string, description string, filename string, contentType string, hook management.DiagnosticHook) {
	if n.ext != nil {
		n.ext.RegisterDiagnosticHook(n.receiverName, description, filename, contentType, hook)
	}
}
func (n *OtelManager) SetDiagnosticExtension(receiverName string, ext DiagnosticExtension) {
	n.ext = ext
	n.receiverName = receiverName
}
