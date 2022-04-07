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

//go:build windows
// +build windows

package decode_xml_wineventlog

import (
	"sync"

	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/logp"
	"github.com/elastic/beats/v8/winlogbeat/sys/winevent"
	"github.com/elastic/beats/v8/winlogbeat/sys/wineventlog"
)

type winDecoder struct {
	cache *metadataCache
}

func newDecoder() decoder {
	return &winDecoder{
		cache: &metadataCache{
			store: map[string]*winevent.WinMeta{},
			log:   logp.NewLogger(logName),
		},
	}
}

func (dec *winDecoder) decode(data []byte) (common.MapStr, common.MapStr, error) {
	evt, err := winevent.UnmarshalXML(data)
	if err != nil {
		return nil, nil, err
	}
	md := dec.cache.getPublisherMetadata(evt.Provider.Name)
	winevent.EnrichRawValuesWithNames(md, &evt)
	win, ecs := fields(evt)
	return win, ecs, nil
}

type metadataCache struct {
	store map[string]*winevent.WinMeta
	mutex sync.RWMutex

	log *logp.Logger
}

func (c *metadataCache) getPublisherMetadata(publisher string) *winevent.WinMeta {
	// NOTE: This code uses double-check locking to elevate to a write-lock
	// when a cache value needs initialized.
	c.mutex.RLock()

	// Lookup cached value.
	md, found := c.store[publisher]
	if !found {
		// Elevate to write lock.
		c.mutex.RUnlock()
		c.mutex.Lock()
		defer c.mutex.Unlock()

		// Double-check if the condition changed while upgrading the lock.
		md, found = c.store[publisher]
		if found {
			return md
		}

		// Load metadata from the publisher.
		md, err := wineventlog.NewPublisherMetadataStore(wineventlog.NilHandle, publisher, c.log)
		if err != nil {
			// Return an empty store on error (can happen in cases where the
			// log was forwarded and the provider doesn't exist on collector).
			md = wineventlog.NewEmptyPublisherMetadataStore(publisher, c.log)
		}
		c.store[publisher] = &md.WinMeta
	} else {
		c.mutex.RUnlock()
	}

	return md
}
