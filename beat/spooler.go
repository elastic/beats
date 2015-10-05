package beat

import (
	"time"

	cfg "github.com/elastic/filebeat/config"
	"github.com/elastic/filebeat/input"
	"github.com/elastic/libbeat/logp"
)

type Spooler struct {
	Filebeat *Filebeat
	running  bool
}

func NewSpooler(filebeat *Filebeat) *Spooler {
	return &Spooler{
		Filebeat: filebeat,
		running:  false,
	}
}

func (spooler *Spooler) Config() error {
	config := &spooler.Filebeat.FbConfig.Filebeat

	// Set default pool size if value not set
	if config.SpoolSize == 0 {
		config.SpoolSize = cfg.DefaultSpoolSize
	}

	// Set default idle timeout if not set
	if config.IdleTimeout == "" {
		logp.Debug("spooler", "Set idleTimeoutDuration to %s", cfg.DefaultIdleTimeout)
		// Set it to default
		config.IdleTimeoutDuration = cfg.DefaultIdleTimeout
	} else {
		var err error

		config.IdleTimeoutDuration, err = time.ParseDuration(config.IdleTimeout)

		if err != nil {
			logp.Warn("Failed to parse idle timeout duration '%s'. Error was: %v", config.IdleTimeout, err)
			return err
		}
	}

	return nil
}

func (s *Spooler) Start() {
	// heartbeat periodically. If the last flush was longer than
	// 'idle_timeout' time ago, then we'll force a flush to prevent us from
	// holding on to spooled events for too long.

	config := &s.Filebeat.FbConfig.Filebeat

	// Enable running
	s.running = true

	ticker := time.NewTicker(config.IdleTimeoutDuration / 2)

	// slice for spooling into
	// TODO(sissel): use container.Ring?
	spool := make([]*input.FileEvent, config.SpoolSize)

	// Current write position in the spool
	var spool_i int = 0

	next_flush_time := time.Now().Add(config.IdleTimeoutDuration)
	for {
		select {
		case event := <-s.Filebeat.SpoolChan:
			spool[spool_i] = event
			spool_i++

			// Flush if full
			if spool_i == cap(spool) {
				//spoolcopy := make([]*FileEvent, max_size)
				var spoolcopy []*input.FileEvent
				//fmt.Println(spool[0])
				spoolcopy = append(spoolcopy, spool[:]...)

				// Send events to publisher
				s.Filebeat.publisherChan <- spoolcopy
				next_flush_time = time.Now().Add(config.IdleTimeoutDuration)

				spool_i = 0
			}
		case <-ticker.C:
			if now := time.Now(); now.After(next_flush_time) {
				// if current time is after the next_flush_time, flush!
				//fmt.Printf("timeout: %d exceeded by %d\n", idle_timeout,
				//now.Sub(next_flush_time))

				// Flush what we have, if anything
				if spool_i > 0 {
					var spoolcopy []*input.FileEvent
					spoolcopy = append(spoolcopy, spool[0:spool_i]...)
					s.Filebeat.publisherChan <- spoolcopy
					next_flush_time = now.Add(config.IdleTimeoutDuration)
					spool_i = 0
				}
			}
		}

		if !s.running {
			break
		}
	}
}

func (s *Spooler) Stop() {
	s.running = false
}
