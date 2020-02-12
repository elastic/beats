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

// +build linux darwin windows

package add_kubernetes_metadata

import (
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/common"
)

type cache struct {
	sync.Mutex
	timeout  time.Duration
	deleted  map[string]time.Time // key ->  when should this obj be deleted
	metadata map[string]common.MapStr
}

func newCache(cleanupTimeout time.Duration) *cache {
	c := &cache{
		timeout:  cleanupTimeout,
		deleted:  make(map[string]time.Time),
		metadata: make(map[string]common.MapStr),
	}
	go c.cleanup()
	return c
}

func (c *cache) get(key string) common.MapStr {
	c.Lock()
	defer c.Unlock()
	// add lifecycle if key was queried
	if t, ok := c.deleted[key]; ok {
		c.deleted[key] = t.Add(c.timeout)
	}
	return c.metadata[key]
}

func (c *cache) delete(key string) {
	c.Lock()
	defer c.Unlock()
	c.deleted[key] = time.Now().Add(c.timeout)
}

func (c *cache) set(key string, data common.MapStr) {
	c.Lock()
	defer c.Unlock()
	delete(c.deleted, key)
	c.metadata[key] = data
}

func (c *cache) cleanup() {
	ticker := time.Tick(timeout)
	for now := range ticker {
		c.Lock()
		for k, t := range c.deleted {
			if now.After(t) {
				delete(c.deleted, k)
				delete(c.metadata, k)
			}
		}
		c.Unlock()
	}
}
