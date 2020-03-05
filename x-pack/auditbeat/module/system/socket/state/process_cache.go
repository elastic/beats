// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux,386 linux,amd64

package state

import (
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
)

type ProcessCache struct {
	*common.Cache
}

func NewProcessCache(d time.Duration) *ProcessCache {
	return &ProcessCache{common.NewCache(d, 8)}
}

func (p *ProcessCache) Put(value *Process) {
	if value.pid == 0 {
		// no-op for uninitialized processes
		return
	}

	p.Cache.Put(value.pid, value)
}

func (p *ProcessCache) Get(pid uint32) *Process {
	if pid == 0 {
		// no-op for uninitialized processes
		return nil
	}

	if value := p.Cache.Get(pid); value != nil {
		return value.(*Process)
	}
	return nil
}
