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

type flowRemovalListener func(v *socket_common.Flow)
type flowCache struct {
	*common.Cache
}

func newFlowCache(d time.Duration, l flowRemovalListener) *flowCache {
	return &flowCache{common.NewCacheWithRemovalListener(d, 8, func(_ common.Key, value common.Value) {
		if l != nil {
			l(value.(*socket_common.Flow))
		}
	})}
}

func (f *flowCache) PutIfAbsent(value *socket_common.Flow) *socket_common.Flow {
	if value.HasKey() {
		v := f.Cache.PutIfAbsent(value.Key(), value)
		if v == nil {
			return value
		}
		return v.(*socket_common.Flow)
	}
	return nil
}

func (f *flowCache) Evict(value *socket_common.Flow) *socket_common.Flow {
	if value.HasKey() {
		v := f.Cache.Evict(value.Key())
		if v == nil {
			return nil
		}
		return v.(*socket_common.Flow)
	}
	return nil
}
