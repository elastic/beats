package common

import (
	"fmt"

	"github.com/elastic/beats/v7/x-pack/auditbeat/module/system/socket/dns"
)

// EventTracker is the interface that holds the state of past events.
type EventTracker interface {
	UpdateFlow(*Flow)
	UpdateFlowWithCondition(*Flow, func(*Flow) bool)
	SocketEnd(uintptr, uint32)
	ProcessStart(*Process)
	ProcessEnd(uint32)
	PushThreadEvent(uint32, Event)
	PopThreadEvent(uint32) Event
	SyncClocks(uint64, uint64)
	OnDNSTransaction(dns.Transaction) error
}

// Event is the interface that all the deserialized events from the ring-buffer
// have to conform to in order to be processed by state.
type Event interface {
	fmt.Stringer
	Update(EventTracker)
}
