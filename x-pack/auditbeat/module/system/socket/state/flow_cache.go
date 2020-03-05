// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

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
