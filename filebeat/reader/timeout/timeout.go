package timeout

import (
	"errors"
	"time"

	"github.com/elastic/beats/filebeat/reader"
)

var (
	errTimeout = errors.New("timeout")
)

// timeoutProcessor will signal some configurable timeout error if no
// new line can be returned in time.
type Reader struct {
	reader  reader.Reader
	timeout time.Duration
	signal  error
	running bool
	ch      chan lineMessage
}

type lineMessage struct {
	line reader.Message
	err  error
}

// New returns a new timeout reader from an input line reader.
func New(reader reader.Reader, signal error, t time.Duration) *Reader {
	if signal == nil {
		signal = errTimeout
	}

	return &Reader{
		reader:  reader,
		signal:  signal,
		timeout: t,
		ch:      make(chan lineMessage, 1),
	}
}

// Next returns the next line. If no line was returned before timeout, the
// configured timeout error is returned.
// For handline timeouts a goroutine is started for reading lines from
// configured line reader. Only when underlying reader returns an error, the
// goroutine will be finished.
func (r *Reader) Next() (reader.Message, error) {
	if !r.running {
		r.running = true
		go func() {
			for {
				message, err := r.reader.Next()
				r.ch <- lineMessage{message, err}
				if err != nil {
					break
				}
			}
		}()
	}

	select {
	case msg := <-r.ch:
		if msg.err != nil {
			r.running = false
		}
		return msg.line, msg.err
	case <-time.After(r.timeout):
		return reader.Message{}, r.signal
	}
}
