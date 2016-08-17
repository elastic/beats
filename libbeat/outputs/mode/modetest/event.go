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

func EventsList(in []EventInfo) [][]outputs.Data {
	var out [][]outputs.Data
	for _, pubEvents := range in {
		if pubEvents.Single {
			for _, event := range pubEvents.Data {
				out = append(out, []outputs.Data{event})
			}
		} else {
			out = append(out, pubEvents.Data)
		}
	}
	return out
}

func FlatEventsList(in []EventInfo) []outputs.Data {
	return FlattenEvents(EventsList(in))
}

func FlattenEvents(data [][]outputs.Data) []outputs.Data {
	var out []outputs.Data
	for _, inner := range data {
		for _, d := range inner {
			out = append(out, d)
		}
	}
	return out
}
