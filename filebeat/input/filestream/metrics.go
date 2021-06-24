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

package filestream

import (
	"context"
	"sync"
	"time"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	"github.com/elastic/beats/v7/libbeat/monitoring"
)

type filesProgress struct {
	sync.Mutex

	table map[string]*progressMetrics
}

func newFilesProgress() *filesProgress {
	return &filesProgress{
		table: make(map[string]*progressMetrics),
	}
}

func (p *filesProgress) add(id string, m *progressMetrics) {
	p.Lock()
	defer p.Unlock()

	p.table[id] = m
}

func (p *filesProgress) isMonitored(id string) bool {
	p.Lock()
	defer p.Unlock()

	_, ok := p.table[id]
	return ok
}

func (p *filesProgress) remove(id string) {
	p.Lock()
	defer p.Unlock()

	if m, ok := p.table[id]; ok {
		m.cancel()
		delete(p.table, id)
	}
}

func (p *filesProgress) updatePath(id, path string) {
	p.Lock()
	defer p.Unlock()

	if m, ok := p.table[id]; ok {
		m.path.Set(path)
	}
}

func (p *filesProgress) updateOnPublish(id string, lastPublished, lastPublishedEventTs time.Time, offset int64) {
	p.Lock()
	defer p.Unlock()

	if m, ok := p.table[id]; ok {
		m.updateOnPublish(lastPublished, lastPublishedEventTs, offset)
	}
}

// progressMetrics contians detailed metrics about every opened file.
type progressMetrics struct {
	all *filesProgress
	id  string

	reg                  *monitoring.Registry
	path                 *monitoring.String
	stateID              *monitoring.String
	started              *monitoring.String
	lastPublished        *monitoring.Timestamp
	lastPublishedEventTs *monitoring.Timestamp
	currentSize          *monitoring.Int
	readOffset           *monitoring.Int
	status               *monitoring.String
	cancel               context.CancelFunc
}

func newProgressMetrics(inputID, path, stateID string, cancel context.CancelFunc, started bool) *progressMetrics {
	if inputID == loginp.GlobalInputID {
		inputID = loginp.GlobalInputID[1:]
	}
	registryID := pluginName + ".files." + inputID + "." + stateID
	r := monitoring.GetNamespace("dataset").GetRegistry().NewRegistry(registryID)
	reg := &progressMetrics{
		id:          stateID,
		reg:         r,
		path:        monitoring.NewString(r, "path"),
		stateID:     monitoring.NewString(r, "state_id"),
		currentSize: monitoring.NewInt(r, "size"),
		readOffset:  monitoring.NewInt(r, "read_offset"),
		status:      monitoring.NewString(r, "status"),
		cancel:      cancel,
	}

	reg.path.Set(path)
	reg.stateID.Set(stateID)

	state := "INACTIVE"
	if started {
		state = "ACTIVE"

		reg.started = monitoring.NewString(r, "start_time")
		reg.lastPublished = monitoring.NewTimestamp(r, "last_event_published_time")
		reg.lastPublishedEventTs = monitoring.NewTimestamp(r, "last_event_timestamp")
		reg.started.Set(time.Now().String())
	}
	reg.status.Set(state)

	return reg
}

func (p *progressMetrics) updateOnPublish(lastPublished, lastPublishedEventTs time.Time, offset int64) {
	p.lastPublished.Set(lastPublished)
	p.lastPublishedEventTs.Set(lastPublishedEventTs)
	p.readOffset.Set(offset)
}

func (p *progressMetrics) updateCurrentSize(size int64) {
	p.currentSize.Set(size)
}

func (p *progressMetrics) stop() {
	p.all.remove(p.id)
}
