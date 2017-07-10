package pipeline

import (
	"time"

	"github.com/elastic/beats/libbeat/common/atomic"
	"github.com/elastic/beats/libbeat/publisher/beat"
)

type clientACKer struct {
	acker
	active atomic.Bool
}

func (p *Pipeline) makeACKer(
	withProcessors bool,
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
		acker = bld.createCountACKer(withProcessors, sema, cfg.ACKCount)
	case cfg.ACKEvents != nil:
		acker = bld.createEventACKer(withProcessors, sema, cfg.ACKEvents)
	case cfg.ACKLastEvent != nil:
		cb := lastEventACK(cfg.ACKLastEvent)
		acker = bld.createEventACKer(withProcessors, sema, cb)
	default:
		if waitClose <= 0 {
			return bld.createPipelineACKer(withProcessors, sema)
		}
		acker = bld.createCountACKer(withProcessors, sema, func(_ int) {})
	}

	if waitClose <= 0 {
		return acker
	}
	return newWaitACK(acker, waitClose)
}

func lastEventACK(fn func(beat.Event)) func([]beat.Event) {
	return func(events []beat.Event) {
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
	mk func(*clientACKer) func([]beat.Event, int),
) acker {
	guard := &clientACKer{}
	guard.lift(newEventACK(pipeline, canDrop, sema, mk(guard)))
	return guard
}
