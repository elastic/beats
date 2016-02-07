package mode

import "time"

type backoff struct {
	duration time.Duration
	done     <-chan struct{}

	init time.Duration
	max  time.Duration
}

func newBackoff(done <-chan struct{}, init, max time.Duration) *backoff {
	return &backoff{
		duration: init,
		done:     done,
		init:     init,
		max:      max,
	}
}

func (b *backoff) Reset() {
	b.duration = b.init
}

func (b *backoff) Wait() bool {

	backoff := b.duration
	b.duration *= 2
	if b.duration > b.max {
		b.duration = b.max
	}

	debug("backoff: wait for %v", b.duration)

	select {
	case <-b.done:
		return false
	case <-time.After(backoff):
		return true
	}
}

func (b *backoff) WaitOnError(err error) bool {
	if err == nil {
		b.Reset()
		return true
	}
	return b.Wait()
}
