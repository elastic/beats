package processor

import (
	"errors"
	"time"
)

var (
	errTimeout = errors.New("timeout")
)

// timeoutProcessor will signal some configurable timeout error if no
// new line can be returned in time.
type timeoutProcessor struct {
	reader  LineProcessor
	timeout time.Duration
	signal  error

	running bool
	ch      chan lineMessage
}

type lineMessage struct {
	line Line
	err  error
}

// newTimeoutProcessor returns a new timeoutProcessor from an input line processor.
func newTimeoutProcessor(in LineProcessor, signal error, timeout time.Duration) *timeoutProcessor {
	if signal == nil {
		signal = errTimeout
	}

	return &timeoutProcessor{
		reader:  in,
		signal:  signal,
		timeout: timeout,
		ch:      make(chan lineMessage, 1),
	}
}

// Next returns the next line. If no line was returned before timeout, the
// configured timeout error is returned.
// For handline timeouts a goroutine is started for reading lines from
// configured line processor. Only when underlying processor returns an error, the
// goroutine will be finished.
func (p *timeoutProcessor) Next() (Line, error) {
	if !p.running {
		p.running = true
		go func() {
			for {
				line, err := p.reader.Next()
				p.ch <- lineMessage{line, err}
				if err != nil {
					break
				}
			}
		}()
	}

	select {
	case msg := <-p.ch:
		if msg.err != nil {
			p.running = false
		}
		return msg.line, msg.err
	case <-time.After(p.timeout):
		return Line{}, p.signal
	}
}
