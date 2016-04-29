package modetest

import "github.com/elastic/beats/libbeat/common"

type EventInfo struct {
	Single bool
	Events []common.MapStr
}

func SingleEvent(e common.MapStr) []EventInfo {
	events := []common.MapStr{e}
	return []EventInfo{
		{Single: true, Events: events},
	}
}

func MultiEvent(n int, event common.MapStr) []EventInfo {
	var events []common.MapStr
	for i := 0; i < n; i++ {
		events = append(events, event)
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
