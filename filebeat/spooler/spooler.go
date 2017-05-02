package spooler

import (
	"sync"
	"time"

	cfg "github.com/elastic/beats/filebeat/config"
	"github.com/elastic/beats/filebeat/util"
	"github.com/elastic/beats/libbeat/logp"
)

var debugf = logp.MakeDebug("spooler")

// channelSize is the number of events Channel can buffer before blocking will occur.
const channelSize = 16

// Spooler aggregates the events and sends the aggregated data to the publisher.
type Spooler struct {
	Channel chan *util.Data // Channel is the input to the Spooler.
	config  spoolerConfig
	output  Output         // batch event output on flush
	spool   []*util.Data   // Events being held by the Spooler.
	wg      sync.WaitGroup // WaitGroup used to control the shutdown.
}

// Output spooler sends event to through Send method
type Output interface {
	Send(events []*util.Data) bool
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
		Channel: make(chan *util.Data, channelSize),
		config: spoolerConfig{
			idleTimeout: config.IdleTimeout,
			spoolSize:   config.SpoolSize,
		},
		output: out,
		spool:  make([]*util.Data, 0, config.SpoolSize),
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

	defer s.flush()
	defer s.wg.Done()

	timer := time.NewTimer(s.config.idleTimeout)
	defer timer.Stop()

	for {
		select {
		case data, ok := <-s.Channel:
			if !ok {
				return
			}
			if data != nil {
				flushed := s.queue(data)
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
func (s *Spooler) queue(data *util.Data) bool {
	flushed := false
	s.spool = append(s.spool, data)
	if len(s.spool) == cap(s.spool) {
		debugf("Flushing spooler because spooler full. Events flushed: %v", len(s.spool))
		s.flush()
		flushed = true
	}
	return flushed
}

// flush flushes all events to the publisher.
func (s *Spooler) flush() int {

	count := len(s.spool)
	if count == 0 {
		return 0
	}

	// copy buffer
	tmpCopy := make([]*util.Data, count)
	copy(tmpCopy, s.spool)

	// clear buffer
	s.spool = s.spool[:0]

	// send batched events to output
	s.output.Send(tmpCopy)

	return count
}
