package webserver

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/iis"
)


func eventsMapping(r mb.ReporterV2, events []mb.Event) {

	//remapping process details to match the naming format
	groupedEvents := make(map[interface{}][]mb.Event)
	for _, event := range events {
		website, err := event.MetricSetFields.GetValue("website.name")
		if err != nil {
			r.Error(err)
			return
		}
		groupedEvents[website] = append(groupedEvents[website], event)
	}
	for _, grouped := range groupedEvents {
		counters := common.MapStr{}
		for _, event := range grouped {
			counterResults, err := event.MetricSetFields.GetValue("webserver")
			if err != nil {
				r.Error(err)
				return
			}
			mapCounterResults, err := iis.MapCounter(counterResults)
			if err != nil {
				r.Error(err)
				return
			}
			common.MergeFields(counters, mapCounterResults, true)
		}
		groupedEvent := mb.Event{MetricSetFields: counters}
		r.Event(groupedEvent)
	}
	}




