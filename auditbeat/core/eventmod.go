package core

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
)

// AddDatasetToEvent adds dataset information to the event. In particular this
// adds the module name under dataset.module.
func AddDatasetToEvent(module, metricSet string, event *mb.Event) {
	if event.RootFields == nil {
		event.RootFields = common.MapStr{}
	}

	event.RootFields.Put("event.module", module)
}
