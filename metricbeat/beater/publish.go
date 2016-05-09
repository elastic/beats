package beater

import (
	"sync"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/publisher"
)

// PublishChannels publishes the events read from each channel to the given
// publisher client. If the publisher client blocks for any reason then events
// will not be read from the given channels.
//
// This method blocks until all of the channels have been closed
// and are fully read. To stop the method immediately, close the channels and
// close the publisher client to ensure that publishing does not block. This
// may result is some events being discarded.
func PublishChannels(client publisher.Client, cs ...<-chan common.MapStr) {
	var wg sync.WaitGroup

	// output publishes values from c until c is closed, then calls wg.Done.
	output := func(c <-chan common.MapStr) {
		defer wg.Done()
		for event := range c {
			client.PublishEvent(event)
		}
	}

	// Start an output goroutine for each input channel in cs.
	wg.Add(len(cs))
	for _, c := range cs {
		go output(c)
	}

	wg.Wait()
}
