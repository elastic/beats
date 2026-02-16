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

package bbolt

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/elastic/beats/v7/libbeat/statestore/backend"
	"github.com/elastic/elastic-agent-libs/logp"
)

var errRegClosed = errors.New("registry has been closed")

// Registry provides access to bbolt-based stores.
type Registry struct {
	log *logp.Logger

	mu     sync.Mutex
	active bool

	settings Settings
}

// Settings configures a new bbolt Registry.
type Settings struct {
	// Root is the directory where bbolt database files are stored.
	Root string

	// FileMode is the file mode for new database files.
	// Defaults to 0600 if not set.
	FileMode os.FileMode

	// Config holds bbolt-specific configuration.
	Config Config
}

const defaultFileMode os.FileMode = 0600

// New creates a new bbolt Registry.
func New(log *logp.Logger, settings Settings) (*Registry, error) {
	if settings.FileMode == 0 {
		settings.FileMode = defaultFileMode
	}

	root, err := filepath.Abs(settings.Root)
	if err != nil {
		return nil, err
	}
	settings.Root = root

	if err := os.MkdirAll(root, os.ModeDir|0770); err != nil {
		return nil, fmt.Errorf("failed to create registry directory %s: %w", root, err)
	}

	log.Debugf("Created bbolt registry: root=%s file_mode=%04o", root, settings.FileMode)

	return &Registry{
		log:      log,
		active:   true,
		settings: settings,
	}, nil
}

// Access opens a store. Each store is backed by its own bbolt database
// file named <name>.db inside the registry root directory.
func (r *Registry) Access(name string) (backend.Store, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.active {
		return nil, errRegClosed
	}

	logger := r.log.With("store", name)
	dbPath := filepath.Join(r.settings.Root, name+".db")

	logger.Debugf("Opening bbolt store: path=%s", dbPath)

	store, err := openStore(logger, dbPath, r.settings.FileMode, r.settings.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to open store %q: %w", name, err)
	}

	logger.Debugf("Opened bbolt store: path=%s", dbPath)

	return store, nil
}

// Close closes the registry. No new stores can be opened after Close.
func (r *Registry) Close() error {
	r.mu.Lock()
	r.active = false
	r.mu.Unlock()
	r.log.Debug("Closed bbolt registry")
	return nil
}
