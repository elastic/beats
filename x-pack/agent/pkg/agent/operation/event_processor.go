// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package operation

import "context"

// EventProcessor is an processor of application event
type callbackHooks interface {
	OnStarting(ctx context.Context, app string)
	OnRunning(ctx context.Context, app string)
	OnFailing(ctx context.Context, app string, err error)
	OnStopping(ctx context.Context, app string)
	OnStopped(ctx context.Context, app string)
	OnFatal(ctx context.Context, app string, err error)
}

type noopCallbackHooks struct{}

func (*noopCallbackHooks) OnStarting(ctx context.Context, app string)           {}
func (*noopCallbackHooks) OnRunning(ctx context.Context, app string)            {}
func (*noopCallbackHooks) OnFailing(ctx context.Context, app string, err error) {}
func (*noopCallbackHooks) OnStopping(ctx context.Context, app string)           {}
func (*noopCallbackHooks) OnStopped(ctx context.Context, app string)            {}
func (*noopCallbackHooks) OnFatal(ctx context.Context, app string, err error)   {}
