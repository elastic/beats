package scheduler

import (
	"math"

	"github.com/elastic/beats/libbeat/common/atomic"
)

type Throttler struct {
	limit          uint
	availableSlots uint
	active         atomic.Int
	starts         chan chan bool
	stops          chan bool
	done           chan bool
}

func NewThrottler(limit uint) *Throttler {
	if limit < 1 { // assume unlimited
		limit = math.MaxUint32
	}

	t := &Throttler{
		limit:          limit,
		availableSlots: limit,
		active:         atomic.Int{},
		starts:         make(chan chan bool),
		stops:          make(chan bool),
		done:           make(chan bool),
	}
	return t
}

func (t *Throttler) start() {
	go func() {
		for {
			// If no slots are available, we just wait for jobs to stop, in which case
			// we can increase the number of slots for next time through the loop
			if t.availableSlots < 1 {
				select {
				case <-t.stops:
					t.availableSlots++
				case <-t.done:
					return
				}
			} else {
				select {
				case <-t.stops:
					t.availableSlots++
				case ch := <-t.starts:
					t.availableSlots--
					ch <- true
				case <-t.done:
					return
				}
			}
		}
	}()
}

func (t *Throttler) stop() {
	close(t.done)
}

// acquireSlot attempts to acquire a resource. It returns whether acquisition was successful.
// If acquisition was successful releaseSlotFn must be invoked, otherwise it may be ignored.
func (t *Throttler) acquireSlot() (acquired bool, releaseSlotFn func()) {
	startedCh := make(chan bool)
	t.starts <- startedCh

	select {
	case <-t.done:
		return false, func() {}
	case <-startedCh:
		t.active.Inc()
		return true, func() {
			t.stops <- true
			t.active.Dec()
		}
	}
}
