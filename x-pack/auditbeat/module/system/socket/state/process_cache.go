// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux,386 linux,amd64

package state

import (
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
	socket_common "github.com/elastic/beats/v7/x-pack/auditbeat/module/system/socket/common"
)

type processCache struct {
	*common.Cache
}

func newProcessCache(d time.Duration) *processCache {
	return &processCache{common.NewCache(d, 8)}
}

func (p *processCache) Put(value *socket_common.Process) {
	if value.PID == 0 {
		// no-op for uninitialized processes
		return
	}

	p.Cache.Put(value.PID, value)
}

func (p *processCache) Get(pid uint32) *socket_common.Process {
	if pid == 0 {
		// no-op for uninitialized processes
		return nil
	}

	if value := p.Cache.Get(pid); value != nil {
		return value.(*socket_common.Process)
	}
	return nil
}
