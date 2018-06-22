package include

import (
	"github.com/elastic/beats/libbeat/feature"
	"github.com/elastic/beats/libbeat/publisher/queue/memqueue"
	"github.com/elastic/beats/libbeat/publisher/queue/spool"

	"github.com/elastic/beats/libbeat/outputs/console"
	"github.com/elastic/beats/libbeat/outputs/elasticsearch"
	"github.com/elastic/beats/libbeat/outputs/fileout"
	"github.com/elastic/beats/libbeat/outputs/kafka"
	"github.com/elastic/beats/libbeat/outputs/logstash"
	"github.com/elastic/beats/libbeat/outputs/redis"
)

// Bundle expose the main features.
var Bundle = feature.MustBundle(
	// Queues types
	feature.MustBundle(
		memqueue.Feature,
		spool.Feature,
	),

	// Outputs
	feature.MustBundle(
		elasticsearch.Feature,
		logstash.Feature,
		redis.Feature,
		kafka.Feature,
		fileout.Feature,
		console.Feature,
	),
)

func init() {
	feature.RegisterBundle(Bundle)
}
