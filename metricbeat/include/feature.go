package include

import (
	"github.com/elastic/beats/filebeat/processor/add_kubernetes_metadata"
	"github.com/elastic/beats/libbeat/feature"
)

var bundle = feature.MustBundle(
	// processors
	feature.MustBundle(
		add_kubernetes_metadata.Feature,
	),
)

func init() {
	feature.MustOverwriteBundle(bundle)
}
