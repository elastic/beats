package beat

import (
	cfg "github.com/elastic/filebeat/config"
	"github.com/elastic/filebeat/input"
	"time"
)

// startSpooler Starts up the spooler and starts listening on the spool channel from the harvester
// Sends then bulk updates to the publisher channel
func (fb *Filebeat) startSpooler(options *cfg.FilebeatConfig) {
	// heartbeat periodically. If the last flush was longer than
	// 'idle_timeout' time ago, then we'll force a flush to prevent us from
	// holding on to spooled events for too long.

	ticker := time.NewTicker(options.IdleTimeout / 2)

	// slice for spooling into
	// TODO(sissel): use container.Ring?
	spool := make([]*input.FileEvent, options.SpoolSize)

	// Current write position in the spool
	var spool_i int = 0

	next_flush_time := time.Now().Add(options.IdleTimeout)
	for {
		select {
		case event := <-fb.SpoolChan:
			spool[spool_i] = event
			spool_i++

			// Flush if full
			if spool_i == cap(spool) {
				//spoolcopy := make([]*FileEvent, max_size)
				var spoolcopy []*input.FileEvent
				//fmt.Println(spool[0])
				spoolcopy = append(spoolcopy, spool[:]...)

				// Send events to publisher
				fb.publisherChan <- spoolcopy
				next_flush_time = time.Now().Add(options.IdleTimeout)

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
					fb.publisherChan <- spoolcopy
					next_flush_time = now.Add(options.IdleTimeout)
					spool_i = 0
				}
			}
		}
	}
}

func (fb *Filebeat) stopSpooler() {

}

