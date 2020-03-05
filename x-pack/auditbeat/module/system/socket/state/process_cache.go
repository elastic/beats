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
