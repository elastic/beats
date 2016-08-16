package modetest

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/outputs"
)

type EventInfo struct {
	Single bool
	Data   []outputs.Data
}

func SingleEvent(e common.MapStr) []EventInfo {
	data := []outputs.Data{{Event: e}}
	return []EventInfo{
		{Single: true, Data: data},
	}
}

func MultiEvent(n int, event common.MapStr) []EventInfo {
	var data []outputs.Data
	for i := 0; i < n; i++ {
		data = append(data, outputs.Data{Event: event})
	}
	return []EventInfo{{Single: false, Data: data}}
}

func Repeat(n int, evt []EventInfo) []EventInfo {
	var events []EventInfo
	for _, e := range evt {
		events = append(events, e)
	}
	return events
}
