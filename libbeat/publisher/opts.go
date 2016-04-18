package publisher

import "github.com/elastic/beats/libbeat/common/op"

// ClientOption allows API users to set additional options when publishing events.
type ClientOption func(option Context) Context

// Guaranteed option will retry publishing the event, until send attempt have
// been ACKed by output plugin.
func Guaranteed(o Context) Context {
	o.Guaranteed = true
	return o
}

// Sync option will block the event publisher until an event has been ACKed by
// the output plugin or failed.
func Sync(o Context) Context {
	o.Sync = true
	return o
}

func Signal(signaler op.Signaler) ClientOption {
	return func(ctx Context) Context {
		if ctx.Signal == nil {
			ctx.Signal = signaler
		} else {
			ctx.Signal = op.CombineSignalers(ctx.Signal, signaler)
		}
		return ctx
	}
}
