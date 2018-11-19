package system

import (
	"github.com/elastic/beats/metricbeat/mb"
)



func eventsMapping(r mb.ReporterV2, events []mb.Event) {

	//remapping process details to match the naming format
	for _, event := range events {


		r.Event(event)
	}
}
