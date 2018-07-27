package include

import (
	"github.com/elastic/beats/libbeat/feature"
	"github.com/elastic/beats/libbeat/publisher/queue/memqueue"
	"github.com/elastic/beats/libbeat/publisher/queue/spool"
)

// Bundle expose the main plugins.
var Bundle = feature.MustBundle(
	// Queues types
	feature.MustBundle(
		memqueue.Feature,
		spool.Feature,
	),
)

func init() {
	feature.RegisterBundle(Bundle)
}
