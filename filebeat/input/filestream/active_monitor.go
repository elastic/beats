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
	"sync"
	"time"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/go-concert/unison"
)

// activeFileMonitor checks the state of every opened file.
// If a file is renamed or removed, it cancels the harvester.
// This has to run separately from reading the file
// to avoid blocking when the output cannot accept events.
type activeFileMonitor struct {
	sync.RWMutex
	log   *logp.Logger
	files map[string]fileSource

	interval     time.Duration
	closeRenamed bool
	closeRemoved bool
}

func newActiveFileMonitor(cfg stateChangeCloserConfig) *activeFileMonitor {
	return &activeFileMonitor{
		log:          logp.NewLogger("active_file_monitor"),
		files:        make(map[string]fileSource, 0),
		interval:     cfg.CheckInterval,
		closeRenamed: cfg.Renamed,
		closeRemoved: cfg.Removed,
	}
}

func (m *activeFileMonitor) addFile(path string, src fileSource) bool {
	m.Lock()
	defer m.Unlock()

	m.log.Debug("Adding new file to monitor %s", path)
	if _, ok := m.files[path]; ok {
		return false
	}
	m.files[path] = src
	return true
}

func (m *activeFileMonitor) run(ctx unison.Canceler, hg *loginp.HarvesterGroup) {
	ticker := time.NewTicker(m.interval)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.Lock()
			for path, src := range m.files {
				isSame, err := isSameFile(path, src.info)
				if m.closeBecauseRemoved(path, err) || m.closeBecauseRenamed(path, isSame) {
					hg.Cancel(src)
					delete(m.files, path)
				}
			}

			m.Unlock()
		}
	}
}

func (m *activeFileMonitor) closeBecauseRemoved(path string, err error) bool {
	mustClose := m.closeRemoved && err != nil
	if mustClose {
		m.log.Debugf("File %s must be closed as it has been removed and close.removed is enabled", path)
	}
	return mustClose
}

func (m *activeFileMonitor) closeBecauseRenamed(path string, isSame bool) bool {
	mustClose := m.closeRenamed && !isSame
	if mustClose {
		m.log.Debugf("File %s must be closed as it has been renamed and close.renamed is enabled", path)
	}
	return mustClose
}
