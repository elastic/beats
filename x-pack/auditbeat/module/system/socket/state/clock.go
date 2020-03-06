// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux,386 linux,amd64

package state

import (
	"sync"
	"time"

	"github.com/elastic/beats/v7/x-pack/auditbeat/module/system/socket/helper"
)

type clock struct {
	sync.Mutex
	// Used to convert kernel time to user time
	epoch    time.Time
	maxDrift time.Duration
	log      helper.Logger
}

func newClock(log helper.Logger, maxDrift time.Duration) *clock {
	return &clock{
		maxDrift: maxDrift,
		log:      log,
	}
}

func (c *clock) sync(kernelNanos, userNanos uint64) {
	userTime := time.Unix(int64(time.Duration(userNanos)/time.Second), int64(time.Duration(userNanos)%time.Second))
	bootTime := userTime.Add(-time.Duration(kernelNanos))

	c.Lock()
	defer c.Unlock()
	if c.epoch == (time.Time{}) {
		c.epoch = bootTime
	}

	drift := c.epoch.Sub(bootTime)
	if drift < -c.maxDrift || drift > c.maxDrift {
		c.epoch = bootTime
		c.log.Debugf("adjusted internal clock drift=%s", drift)
	}
}

func (c *clock) kernelToTime(ts uint64) time.Time {
	if ts == 0 {
		return time.Time{}
	}

	c.Lock()
	defer c.Unlock()
	if c.epoch == (time.Time{}) {
		// This is the first event and time sync hasn't happened yet.
		// Take a temporary epoch relative to time.Now()
		now := time.Now()
		c.epoch = now.Add(-time.Duration(ts))
		return now
	}
	return c.epoch.Add(time.Duration(ts))
}
