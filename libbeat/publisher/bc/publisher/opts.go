package publisher

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/op"
)

// ClientOption allows API users to set additional options when publishing events.
type ClientOption func(option Context) ([]common.MapStr, Context)

// Guaranteed option will retry publishing the event, until send attempt have
// been ACKed by output plugin.
func Guaranteed(o Context) ([]common.MapStr, Context) {
	o.Guaranteed = true
	return nil, o
}

// Sync option will block the event publisher until an event has been ACKed by
// the output plugin or failed.
func Sync(o Context) ([]common.MapStr, Context) {
	o.Sync = true
	return nil, o
}

func Signal(signaler op.Signaler) ClientOption {
	return func(ctx Context) ([]common.MapStr, Context) {
		if ctx.Signal == nil {
			ctx.Signal = signaler
		} else {
			ctx.Signal = op.CombineSignalers(ctx.Signal, signaler)
		}
		return nil, ctx
	}
}

func Metadata(m common.MapStr) ClientOption {
	if len(m) == 0 {
		return nilOption
	}
	return func(ctx Context) ([]common.MapStr, Context) {
		return []common.MapStr{m}, ctx
	}
}

func MetadataBatch(m []common.MapStr) ClientOption {
	if len(m) == 0 {
		return nilOption
	}
	return func(ctx Context) ([]common.MapStr, Context) {
		return m, ctx
	}
}

func nilOption(o Context) ([]common.MapStr, Context) {
	return nil, o
}
