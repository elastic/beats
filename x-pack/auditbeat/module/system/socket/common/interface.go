// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux,386 linux,amd64

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
