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

	input "github.com/elastic/beats/v7/filebeat/input/v2"
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/statestore"
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
// are allowed to add a custome per input configuration ID using the `id`
// setting, to collect the same source multiple times, but with different
// state. The key name in the persistent store becomes <Type>-[<ID>]-<Source Name>
type InputManager struct {
	Logger *logp.Logger

	// StateStore gives the InputManager access to the persitent key value store.
	StateStore StateStore

	// Type must contain the name of the input type. It is used to create the key name
	// for all sources the inputs collect from.
	Type string

	// DefaultCleanTimeout configures the key/value garbage collection interval.
	// The InputManager will only collect keys for the configured 'Type'
	DefaultCleanTimeout time.Duration

	// Configure returns an array of Sources, and a configured Input instances
	// that will be used to collect events from each source.
	Configure func(cfg *common.Config) (Prospector, Harvester, error)

	initOnce   sync.Once
	initErr    error
	store      *store
	ackUpdater *updateWriter
	ackCH      *updateChan
}

// Source describe a source the input can collect data from.
// The `Name` method must return an unique name, that will be used to identify
// the source in the persistent state store.
type Source interface {
	Name() string
}

var (
	errNoSourceConfigured = errors.New("no source has been configured")
	errNoInputRunner      = errors.New("no input runner available")
)

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
	})

	return cim.initErr
}

// Init starts background processes for deleting old entries from the
// persistent store if mode is ModeRun.
func (cim *InputManager) Init(group unison.Group, mode v2.Mode) error {
	if mode != v2.ModeRun {
		return nil
	}

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
func (cim *InputManager) Create(config *common.Config) (input.Input, error) {
	if err := cim.init(); err != nil {
		return nil, err
	}

	settings := struct {
		ID             string        `config:"id"`
		CleanTimeout   time.Duration `config:"clean_timeout"`
		HarvesterLimit uint64        `config:"harvester_limit"`
	}{ID: "", CleanTimeout: cim.DefaultCleanTimeout, HarvesterLimit: 0}
	if err := config.Unpack(&settings); err != nil {
		return nil, err
	}

	prospector, harvester, err := cim.Configure(config)
	if err != nil {
		return nil, err
	}
	if harvester == nil {
		return nil, errNoInputRunner
	}

	sourceIdentifier, err := newSourceIdentifier(cim.Type, settings.ID)
	if err != nil {
		return nil, fmt.Errorf("error while creating source identifier for input: %v", err)
	}

	pStore := cim.getRetainedStore()
	defer pStore.Release()

	prospectorStore := newSourceStore(pStore, sourceIdentifier)
	err = prospector.Init(prospectorStore)
	if err != nil {
		return nil, err
	}

	return &managedInput{
		manager:          cim,
		ackCH:            cim.ackCH,
		userID:           settings.ID,
		prospector:       prospector,
		harvester:        harvester,
		sourceIdentifier: sourceIdentifier,
		cleanTimeout:     settings.CleanTimeout,
		harvesterLimit:   settings.HarvesterLimit,
	}, nil
}

func (cim *InputManager) getRetainedStore() *store {
	store := cim.store
	store.Retain()
	return store
}

type sourceIdentifier struct {
	prefix           string
	configuredUserID bool
}

func newSourceIdentifier(pluginName, userID string) (*sourceIdentifier, error) {
	if userID == globalInputID {
		return nil, fmt.Errorf("invalid user ID: .global")
	}

	configuredUserID := true
	if userID == "" {
		configuredUserID = false
		userID = globalInputID
	}
	return &sourceIdentifier{
		prefix:           pluginName + "::" + userID + "::",
		configuredUserID: configuredUserID,
	}, nil
}

func (i *sourceIdentifier) ID(s Source) string {
	return i.prefix + s.Name()
}

func (i *sourceIdentifier) MatchesInput(id string) bool {
	return strings.HasPrefix(id, i.prefix)
}
