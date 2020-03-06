// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux,386 linux,amd64

package events

import (
	"fmt"
	"os"

	"github.com/elastic/beats/v7/x-pack/auditbeat/module/system/socket/common"
	"github.com/elastic/beats/v7/x-pack/auditbeat/tracing"
)

type ClockSyncCall struct {
	Meta      tracing.Metadata `kprobe:"metadata"`
	Timestamp uint64           `kprobe:"timestamp"`
}

// String returns a representation of the event.
func (e *ClockSyncCall) String() string {
	return fmt.Sprintf("%s sys_uname[clock-sync](ts=0x%x)", header(e.Meta), e.Timestamp)
}

// Update the state with the contents of this event.
func (e *ClockSyncCall) Update(s common.EventTracker) {
	if int(e.Meta.PID) == os.Getpid() {
		s.SyncClocks(e.Meta.Timestamp, e.Timestamp)
	}
}
