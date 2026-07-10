// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package otelmanager

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

// fakeActionExtension implements ActionExtension for testing OtelManager's
// forwarding of RegisterAction/UnregisterAction.
type fakeActionExtension struct {
	registeredName   string
	handler          func(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error)
	unregisteredName string
	registerErr      error
}

func (f *fakeActionExtension) RegisterActionHandler(name string, handler func(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error)) error {
	f.registeredName = name
	f.handler = handler
	return f.registerErr
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
	m := &OtelManager{logger: logp.NewNopLogger()}
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

// TestOtelManager_RegisterAction_LogsExtensionError verifies that when the
// extension rejects a registration (for example because another receiver
// already registered an action for the same elastic-agent component — see
// ActionExtension's doc comment), OtelManager.RegisterAction logs the error
// so it is visible from this beat's own process. RegisterAction itself
// cannot return an error: its signature is fixed by management.Manager.
func TestOtelManager_RegisterAction_LogsExtensionError(t *testing.T) {
	logger, observedLogs := logptest.NewTestingLoggerWithObserver(t, "otelmanager")

	m := &OtelManager{logger: logger}
	ext := &fakeActionExtension{registerErr: assert.AnError}
	m.SetActionExtension("filebeatreceiver/_agent-component/filestream-default/stream-2", ext)

	assert.NotPanics(t, func() { m.RegisterAction(&fakeAction{name: "some-action"}) })

	require.Equal(t, 1, observedLogs.Len(), "the extension's registration error should be logged")
	logged := observedLogs.All()[0]
	assert.Contains(t, logged.Message, "failed to register action")
	assert.Contains(t, logged.Message, assert.AnError.Error())
}

func TestOtelManager_SetActionExtension_SharesReceiverNameWithDiagnostics(t *testing.T) {
	m := &OtelManager{logger: logp.NewNopLogger()}
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
