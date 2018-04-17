package pipeline

import (
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common/atomic"
)

type clientACKer struct {
	acker
	active atomic.Bool
}

func (p *Pipeline) makeACKer(
	canDrop bool,
	cfg *beat.ClientConfig,
	waitClose time.Duration,
) acker {
	var (
		bld   = p.ackBuilder
		acker acker
	)

	sema := p.eventSema
	switch {
	case cfg.ACKCount != nil:
		acker = bld.createCountACKer(canDrop, sema, cfg.ACKCount)
	case cfg.ACKEvents != nil:
		acker = bld.createEventACKer(canDrop, sema, cfg.ACKEvents)
	case cfg.ACKLastEvent != nil:
		cb := lastEventACK(cfg.ACKLastEvent)
		acker = bld.createEventACKer(canDrop, sema, cb)
	default:
		if waitClose <= 0 {
			return bld.createPipelineACKer(canDrop, sema)
		}
		acker = bld.createCountACKer(canDrop, sema, func(_ int) {})
	}

	if waitClose <= 0 {
		return acker
	}
	return newWaitACK(acker, waitClose)
}

func lastEventACK(fn func(interface{})) func([]interface{}) {
	return func(events []interface{}) {
		fn(events[len(events)-1])
	}
}

func (a *clientACKer) lift(acker acker) {
	a.active = atomic.MakeBool(true)
	a.acker = acker
}

func (a *clientACKer) Active() bool {
	return a.active.Load()
}

func (a *clientACKer) close() {
	a.active.Store(false)
	a.acker.close()
}

func (a *clientACKer) addEvent(event beat.Event, published bool) bool {
	if a.active.Load() {
		return a.acker.addEvent(event, published)
	}
	return false
}

func (a *clientACKer) ackEvents(n int) {
	a.acker.ackEvents(n)
}

func buildClientCountACK(
	pipeline *Pipeline,
	canDrop bool,
	sema *sema,
	mk func(*clientACKer) func(int, int),
) acker {
	guard := &clientACKer{}
	cb := mk(guard)
	guard.lift(makeCountACK(pipeline, canDrop, sema, cb))
	return guard
}

func buildClientEventACK(
	pipeline *Pipeline,
	canDrop bool,
	sema *sema,
	mk func(*clientACKer) func([]interface{}, int),
) acker {
	guard := &clientACKer{}
	guard.lift(newEventACK(pipeline, canDrop, sema, mk(guard)))
	return guard
}
