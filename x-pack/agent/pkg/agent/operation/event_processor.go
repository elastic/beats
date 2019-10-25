// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package operation

// EventProcessor is an processor of application event
type callbackHooks interface {
	OnStarting(app string)
	OnRunning(app string)
	OnFailing(app string, err error)
	OnStopping(app string)
	OnStopped(app string)
	OnFatal(app string, err error)
}

type noopCallbackHooks struct{}

func (*noopCallbackHooks) OnStarting(app string)           {}
func (*noopCallbackHooks) OnRunning(app string)            {}
func (*noopCallbackHooks) OnFailing(app string, err error) {}
func (*noopCallbackHooks) OnStopping(app string)           {}
func (*noopCallbackHooks) OnStopped(app string)            {}
func (*noopCallbackHooks) OnFatal(app string, err error)   {}
