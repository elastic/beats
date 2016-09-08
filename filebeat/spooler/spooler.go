package spooler

import (
	"sync"
	"time"

	cfg "github.com/elastic/beats/filebeat/config"
	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/libbeat/logp"
)

var debugf = logp.MakeDebug("spooler")

// channelSize is the number of events Channel can buffer before blocking will occur.
const channelSize = 16

// Spooler aggregates the events and sends the aggregated data to the publisher.
type Spooler struct {
	Channel chan *input.Event // Channel is the input to the Spooler.
	config  spoolerConfig
	exit    chan struct{}  // Channel used to signal shutdown.
	output  Output         // batch event output on flush
	spool   []*input.Event // Events being held by the Spooler.
	wg      sync.WaitGroup // WaitGroup used to control the shutdown.
}

type Output interface {
	Send(events []*input.Event) bool
}

type spoolerConfig struct {
	idleTimeout time.Duration // How often to flush the spooler if spoolSize is not reached.
	spoolSize   uint64        // Maximum number of events that are stored before a flush occurs.
}

// New creates and returns a new Spooler. The returned Spooler must be
// started by calling Start before it can be used.
func New(
	config *cfg.Config,
	out Output,
) (*Spooler, error) {
	return &Spooler{
		Channel: make(chan *input.Event, channelSize),
		config: spoolerConfig{
			idleTimeout: config.IdleTimeout,
			spoolSize:   config.SpoolSize,
		},
		exit:   make(chan struct{}),
		output: out,
		spool:  make([]*input.Event, 0, config.SpoolSize),
	}, nil
}

// Start starts the Spooler. Stop must be called to stop the Spooler.
func (s *Spooler) Start() {
	s.wg.Add(1)
	go s.run()
}

// run queues events that it reads from Channel and flushes them when either the
// queue reaches its capacity (which is spoolSize) or a timeout period elapses.
func (s *Spooler) run() {
	logp.Info("Starting spooler: spool_size: %v; idle_timeout: %s",
		s.config.spoolSize, s.config.idleTimeout)

	defer s.wg.Done()

	timer := time.NewTimer(s.config.idleTimeout)
	defer timer.Stop()

	for {
		select {
		case event, ok := <-s.Channel:
			if !ok {
				return
			}

			if event != nil {
				flushed := s.queue(event)
				if flushed {
					// Stop timer and drain channel. See https://golang.org/pkg/time/#Timer.Reset
					if !timer.Stop() {
						<-timer.C
					}
					timer.Reset(s.config.idleTimeout)
				}
			}
		case <-timer.C:
			debugf("Flushing spooler because of timeout. Events flushed: %v", len(s.spool))
			s.flush()
			timer.Reset(s.config.idleTimeout)
		}
	}
}

// Stop stops this Spooler. This method blocks until all events have been
// flushed to the publisher. The method should only be invoked one time after
// Start has been invoked.
func (s *Spooler) Stop() {
	logp.Info("Stopping spooler")

	// Signal to the run method that it should stop.
	// Stop accepting writes. Any events in the channel will be flushed.
	close(s.Channel)

	// Wait for spooler shutdown to complete.
	s.wg.Wait()
	debugf("Spooler has stopped")
}

// queue queues a single event to be spooled. If the queue reaches spoolSize
// while calling this method then all events in the queue will be flushed to
// the publisher.
func (s *Spooler) queue(event *input.Event) bool {
	flushed := false
	s.spool = append(s.spool, event)
	if len(s.spool) == cap(s.spool) {
		debugf("Flushing spooler because spooler full. Events flushed: %v", len(s.spool))
		s.flush()
		flushed = true
	}
	return flushed
}

// flush flushes all events to the publisher.
func (s *Spooler) flush() {
	if len(s.spool) > 0 {
		// copy buffer
		tmpCopy := make([]*input.Event, len(s.spool))
		copy(tmpCopy, s.spool)

		// clear buffer
		s.spool = s.spool[:0]

		// send batched events to output
		s.output.Send(tmpCopy)
	}
}
