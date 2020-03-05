// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux,386 linux,amd64

package state

import (
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
)

type FlowRemovalListener func(v *Flow)
type FlowCache struct {
	*common.Cache
}

func NewFlowCache(d time.Duration, l FlowRemovalListener) *FlowCache {
	return &FlowCache{common.NewCacheWithRemovalListener(d, 8, func(_ common.Key, value common.Value) {
		if l != nil {
			l(value.(*Flow))
		}
	})}
}

func (f *FlowCache) PutIfAbsent(value *Flow) *Flow {
	if value.hasKey() {
		v := f.Cache.PutIfAbsent(value.key(), value)
		if v == nil {
			return value
		}
		return v.(*Flow)
	}
	return nil
}

func (f *FlowCache) Evict(value *Flow) *Flow {
	if value.hasKey() {
		v := f.Cache.Evict(value.key())
		if v == nil {
			return nil
		}
		return v.(*Flow)
	}
	return nil
}
