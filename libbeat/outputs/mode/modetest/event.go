package modetest

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/outputs"
)

type EventInfo struct {
	Single bool
	Events []outputs.Data
}

func SingleEvent(e common.MapStr) []EventInfo {
	events := []outputs.Data{{Event: e}}
	return []EventInfo{
		{Single: true, Events: events},
	}
}

func MultiEvent(n int, event common.MapStr) []EventInfo {
	var events []outputs.Data
	for i := 0; i < n; i++ {
		events = append(events, outputs.Data{Event: event})
	}
	return []EventInfo{{Single: false, Events: events}}
}

func Repeat(n int, evt []EventInfo) []EventInfo {
	var events []EventInfo
	for _, e := range evt {
		events = append(events, e)
	}
	return events
}
