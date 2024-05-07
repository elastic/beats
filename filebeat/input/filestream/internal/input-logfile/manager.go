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

package input_logfile

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/urso/sderr"

	"github.com/elastic/go-concert/unison"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/statestore"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

// InputManager is used to create, manage, and coordinate stateful inputs and
// their persistent state.
// The InputManager ensures that only one input can be active for a unique source.
// If two inputs have overlapping sources, both can still collect data, but
// only one input will collect from the common source.
//
// The InputManager automatically cleans up old entries without an active
// input, and without any pending update operations for the persistent store.
//
// The Type field is used to create the key name in the persistent store. Users
// are allowed to add a custom type per input configuration ID using the `id`
// setting, to collect the same source multiple times, but with different
// state. The key name in the persistent store becomes <Type>-[<ID>]-<Source Name>
type InputManager struct {
	Logger *logp.Logger

	// StateStore gives the InputManager access to the persistent key value store.
	StateStore StateStore

	// Type must contain the name of the input type. It is used to create the key name
	// for all sources the inputs collect from.
	Type string

	// DefaultCleanTimeout configures the key/value garbage collection interval.
	// The InputManager will only collect keys for the configured 'Type'
	DefaultCleanTimeout time.Duration

	// Configure returns an array of Sources, and a configured Input instances
	// that will be used to collect events from each source.
	Configure func(cfg *conf.C) (Prospector, Harvester, error)

	initOnce   sync.Once
	initErr    error
	store      *store
	ackUpdater *updateWriter
	ackCH      *updateChan
	idsMux     sync.Mutex
	ids        map[string]struct{}
}

// Source describe a source the input can collect data from.
// The `Name` method must return an unique name, that will be used to identify
// the source in the persistent state store.
type Source interface {
	Name() string
}

var errNoInputRunner = errors.New("no input runner available")

// globalInputID is a default ID for inputs created without an ID
// Deprecated: Inputs without an ID are not supported anymore.
const globalInputID = ".global"

// StateStore interface and configurations used to give the Manager access to the persistent store.
type StateStore interface {
	Access() (*statestore.Store, error)
	CleanupInterval() time.Duration
}

func (cim *InputManager) init() error {
	cim.initOnce.Do(func() {
		if cim.DefaultCleanTimeout <= 0 {
			cim.DefaultCleanTimeout = 30 * time.Minute
		}

		log := cim.Logger.With("input_type", cim.Type)

		var store *store
		store, cim.initErr = openStore(log, cim.StateStore, cim.Type)
		if cim.initErr != nil {
			return
		}

		cim.store = store
		cim.ackCH = newUpdateChan()
		cim.ackUpdater = newUpdateWriter(store, cim.ackCH)
		cim.ids = map[string]struct{}{}
	})

	return cim.initErr
}

// Init starts background processes for deleting old entries from the
// persistent store if mode is ModeRun.
func (cim *InputManager) Init(group unison.Group) error {
	if err := cim.init(); err != nil {
		return err
	}

	log := cim.Logger.With("input_type", cim.Type)

	store := cim.getRetainedStore()
	cleaner := &cleaner{log: log}
	err := group.Go(func(canceler context.Context) error {
		defer cim.shutdown()
		defer store.Release()
		interval := cim.StateStore.CleanupInterval()
		if interval <= 0 {
			interval = 5 * time.Minute
		}
		cleaner.run(canceler, store, interval)
		return nil
	})
	if err != nil {
		store.Release()
		cim.shutdown()
		return sderr.Wrap(err, "Can not start registry cleanup process")
	}

	return nil
}

func (cim *InputManager) shutdown() {
	cim.ackUpdater.Close()
	cim.store.Release()
}

// Create builds a new v2.Input using the provided Configure function.
// The Input will run a go-routine per source that has been configured.
func (cim *InputManager) Create(config *conf.C) (v2.Input, error) {
	if err := cim.init(); err != nil {
		return nil, err
	}

	settings := struct {
		ID             string        `config:"id"`
		CleanInactive  time.Duration `config:"clean_inactive"`
		HarvesterLimit uint64        `config:"harvester_limit"`
	}{CleanInactive: cim.DefaultCleanTimeout}
	if err := config.Unpack(&settings); err != nil {
		return nil, err
	}

	if settings.ID == "" {
		cim.Logger.Error("filestream input ID without ID might lead to data" +
			" duplication, please add an ID and restart Filebeat")
	}

	metricsID := settings.ID
	cim.idsMux.Lock()
	if _, exists := cim.ids[settings.ID]; exists {
		cim.Logger.Errorf("filestream input with ID '%s' already exists, this "+
			"will lead to data duplication, please use a different ID. Metrics "+
			"collection has been disabled on this input.", settings.ID)
		metricsID = ""
	}

	// TODO: improve how inputs with empty IDs are tracked.
	// https://github.com/elastic/beats/issues/35202
	cim.ids[settings.ID] = struct{}{}
	cim.idsMux.Unlock()

	prospector, harvester, err := cim.Configure(config)
	if err != nil {
		return nil, err
	}
	if harvester == nil {
		return nil, errNoInputRunner
	}

	sourceIdentifier, err := newSourceIdentifier(cim.Type, settings.ID)
	if err != nil {
		return nil, fmt.Errorf("error while creating source identifier for input: %w", err)
	}

	pStore := cim.getRetainedStore()
	defer pStore.Release()

	prospectorStore := newSourceStore(pStore, sourceIdentifier)

	// create a store with the deprecated global ID. This will be used to
	// migrate the entries in the registry to use the new input ID.
	globalIdentifier, err := newSourceIdentifier(cim.Type, "")
	if err != nil {
		return nil, fmt.Errorf("cannot create global identifier for input: %w", err)
	}
	globalStore := newSourceStore(pStore, globalIdentifier)

	err = prospector.Init(prospectorStore, globalStore, sourceIdentifier.ID)
	if err != nil {
		return nil, err
	}

	return &managedInput{
		manager:          cim,
		ackCH:            cim.ackCH,
		userID:           settings.ID,
		metricsID:        metricsID,
		prospector:       prospector,
		harvester:        harvester,
		sourceIdentifier: sourceIdentifier,
		cleanTimeout:     settings.CleanInactive,
		harvesterLimit:   settings.HarvesterLimit,
	}, nil
}

func (cim *InputManager) Delete(cfg *conf.C) error {
	settings := struct {
		ID string `config:"id"`
	}{}
	if err := cfg.Unpack(&settings); err != nil {
		return fmt.Errorf("could not unpack config to get the input ID: %w", err)
	}

	cim.StopInput(settings.ID)
	return nil
}

// StopInput performs all necessary clean up when an input finishes.
func (cim *InputManager) StopInput(id string) {
	cim.idsMux.Lock()
	delete(cim.ids, id)
	cim.idsMux.Unlock()
}

func (cim *InputManager) getRetainedStore() *store {
	store := cim.store
	store.Retain()
	return store
}

type sourceIdentifier struct {
	prefix string
}

func newSourceIdentifier(pluginName, userID string) (*sourceIdentifier, error) {
	if userID == globalInputID {
		return nil, fmt.Errorf("invalid input ID: .global")
	}

	if userID == "" {
		userID = globalInputID
	}

	return &sourceIdentifier{
		prefix: pluginName + "::" + userID + "::",
	}, nil
}

func (i *sourceIdentifier) ID(s Source) string {
	return i.prefix + s.Name()
}

func (i *sourceIdentifier) MatchesInput(id string) bool {
	return strings.HasPrefix(id, i.prefix)
}
