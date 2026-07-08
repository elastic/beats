// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package otelmanager

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeActionExtension implements ActionExtension for testing OtelManager's
// forwarding of RegisterAction/UnregisterAction.
type fakeActionExtension struct {
	registeredName   string
	handler          func(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error)
	unregisteredName string
}

func (f *fakeActionExtension) RegisterActionHandler(name string, handler func(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error)) {
	f.registeredName = name
	f.handler = handler
}

func (f *fakeActionExtension) UnregisterActionHandler(name string) {
	f.unregisteredName = name
	f.handler = nil
}

// fakeAction implements management.Action.
type fakeAction struct {
	name     string
	executed bool
}

func (a *fakeAction) Name() string { return a.name }

func (a *fakeAction) Execute(_ context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	a.executed = true
	return params, nil
}

func TestOtelManager_RegisterAction(t *testing.T) {
	m := &OtelManager{}
	ext := &fakeActionExtension{}

	// Before an extension is set, RegisterAction/UnregisterAction must be safe
	// no-ops (mirrors the behavior before this feature existed).
	action := &fakeAction{name: "osquery"}
	assert.NotPanics(t, func() { m.RegisterAction(action) })
	assert.NotPanics(t, func() { m.UnregisterAction(action) })

	m.SetActionExtension("osquerybeatreceiver/_agent-component/osquery-default/stream", ext)

	m.RegisterAction(action)
	assert.Equal(t, "osquerybeatreceiver/_agent-component/osquery-default/stream", ext.registeredName)
	require.NotNil(t, ext.handler)

	res, err := ext.handler(t.Context(), map[string]interface{}{"id": "abc"})
	require.NoError(t, err)
	assert.Equal(t, map[string]interface{}{"id": "abc"}, res)
	assert.True(t, action.executed, "invoking the registered handler should execute the underlying action")

	m.UnregisterAction(action)
	assert.Equal(t, "osquerybeatreceiver/_agent-component/osquery-default/stream", ext.unregisteredName)
	assert.Nil(t, ext.handler)
}

func TestOtelManager_SetActionExtension_SharesReceiverNameWithDiagnostics(t *testing.T) {
	m := &OtelManager{}
	diagExt := &fakeDiagnosticExtension{}
	actionExt := &fakeActionExtension{}

	// SetDiagnosticExtension and SetActionExtension are called independently by
	// the receiver wiring code, but both must key off the same receiver name.
	m.SetDiagnosticExtension("comp-1", diagExt)
	m.SetActionExtension("comp-1", actionExt)

	action := &fakeAction{name: "osquery"}
	m.RegisterAction(action)
	assert.Equal(t, "comp-1", actionExt.registeredName)

	m.RegisterDiagnosticHook("ignored", "desc", "file.json", "application/json", func() []byte { return nil })
	assert.Equal(t, "comp-1", diagExt.registeredName)
}

// fakeDiagnosticExtension implements DiagnosticExtension for testing.
type fakeDiagnosticExtension struct {
	registeredName string
}

func (f *fakeDiagnosticExtension) RegisterDiagnosticHook(name, _, _, _ string, _ func() []byte) {
	f.registeredName = name
}
