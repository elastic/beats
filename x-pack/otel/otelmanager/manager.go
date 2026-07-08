// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package otelmanager

import (
	"context"
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

// ActionExtension exposes the ability for beat receivers to register a handler that
// is invoked when elastic-agent routes a Fleet action to this receiver instance.
// NOTE: Changing the function signature will require changes to elastic-agent's
// elasticdiagnostics extension. Proceed with caution.
type ActionExtension interface {
	RegisterActionHandler(name string, handler func(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error))
	UnregisterActionHandler(name string)
}

type WithActionExtension interface {
	// name is the beat name
	// ext is the extension that implements the ActionExtension interface
	SetActionExtension(name string, ext ActionExtension)
}

var _ management.Manager = (*OtelManager)(nil)
var _ WithDiagnosticExtension = (*OtelManager)(nil)
var _ WithActionExtension = (*OtelManager)(nil)

func NewOtelManager(cfg *config.C, registry *reload.Registry, logger *logp.Logger) (management.Manager, error) {
	management.SetUnderAgent(true)
	return &OtelManager{}, nil
}

// OtelManager is the main manager for managing beatreceivers
type OtelManager struct {
	ext          DiagnosticExtension
	actionExt    ActionExtension
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
func (n *OtelManager) Enabled() bool                      { return false }
func (n *OtelManager) AgentInfo() management.AgentInfo    { return management.AgentInfo{} }
func (n *OtelManager) PreInit() error                     { return nil }
func (n *OtelManager) PostInit()                          {}
func (n *OtelManager) Start() error                       { return nil }
func (n *OtelManager) CheckRawConfig(cfg *config.C) error { return nil }
func (n *OtelManager) RegisterAction(action management.Action) {
	if n.actionExt != nil {
		n.actionExt.RegisterActionHandler(n.receiverName, action.Execute)
	}
}
func (n *OtelManager) UnregisterAction(action management.Action) {
	if n.actionExt != nil {
		n.actionExt.UnregisterActionHandler(n.receiverName)
	}
}
func (n *OtelManager) SetPayload(map[string]interface{}) {}
func (n *OtelManager) RegisterDiagnosticHook(_ string, description string, filename string, contentType string, hook management.DiagnosticHook) {
	if n.ext != nil {
		n.ext.RegisterDiagnosticHook(n.receiverName, description, filename, contentType, hook)
	}
}
func (n *OtelManager) SetDiagnosticExtension(receiverName string, ext DiagnosticExtension) {
	n.ext = ext
	n.receiverName = receiverName
}

// SetActionExtension sets the extension used to route Fleet actions to this receiver
// instance. receiverName is the OTel component ID for this beat receiver instance
// (elastic-agent correlates actions back to it).
func (n *OtelManager) SetActionExtension(receiverName string, ext ActionExtension) {
	n.actionExt = ext
	n.receiverName = receiverName
}
