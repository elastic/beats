package include

import (
	"github.com/elastic/beats/libbeat/feature"
	"github.com/elastic/beats/libbeat/processors/actions"
	"github.com/elastic/beats/libbeat/processors/add_cloud_metadata"
	"github.com/elastic/beats/libbeat/processors/add_docker_metadata"
	"github.com/elastic/beats/libbeat/processors/add_host_metadata"
	"github.com/elastic/beats/libbeat/processors/add_kubernetes_metadata"
	"github.com/elastic/beats/libbeat/processors/add_locale"
	"github.com/elastic/beats/libbeat/processors/dissect"
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

	// Processors
	feature.MustBundle(actions.Bundle,
		add_cloud_metadata.Feature,
		add_docker_metadata.Feature,
		add_host_metadata.Feature,
		add_kubernetes_metadata.Feature,
		add_locale.Feature,
		dissect.Feature,
	),
)

func init() {
	feature.RegisterBundle(Bundle)
}
