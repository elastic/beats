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

package es

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/elastic/beats/v7/libbeat/common/transform/typeconv"
	"github.com/elastic/beats/v7/libbeat/esleg/eslegclient"
	"github.com/elastic/beats/v7/libbeat/statestore/backend"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

// The current typical usage of the state storage is such that the consumer
// of the storage fetches all the keys and caches them at the start of the beat.
// Then the key value gets updated (Set is called) possibly frequently, so we want these operations to happen fairly fast
// and not block waiting on Elasticsearch refresh, thus the slight trade-off for performance instead of consistency.
// The value is not normally retrieved after a modification, so the inconsistency (potential refresh delay) is acceptable for our use cases.
//
// If consistency becomes a strict requirement, the storage would need to implement possibly some caching mechanism
// that would guarantee the consistency between Set/Remove/Get/Each operations at least for a given "in-memory" instance of the storage.

type store struct {
	ctx      context.Context
	cn       context.CancelFunc
	log      *logp.Logger
	name     string
	notifier *Notifier

	chReady chan struct{}
	once    sync.Once

	mx     sync.Mutex
	cli    *eslegclient.Connection
	cliErr error
	id     string

	base *baseStore
}

func openStore(ctx context.Context, log *logp.Logger, name string, notifier *Notifier) (*store, error) {
	ctx, cn := context.WithCancel(ctx)
	s := &store{
		ctx:      ctx,
		cn:       cn,
		log:      log.With("name", name).With("backend", "elasticsearch"),
		name:     name,
		notifier: notifier,
		chReady:  make(chan struct{}),
	}

	chCfg := make(chan *conf.C)

	unsubFn := s.notifier.Subscribe(func(c *conf.C) {
		select {
		case chCfg <- c:
		case <-ctx.Done():
		}
	})

	go s.loop(ctx, cn, unsubFn, chCfg)

	return s, nil
}

func (s *store) waitReady() error {
	select {
	case <-s.ctx.Done():
		return s.ctx.Err()
	case <-s.chReady:
		return s.cliErr
	}
}

func (s *store) SetID(id string) {
	s.mx.Lock()
	s.id = id
	s.mx.Unlock()

	if err := s.waitReady(); err != nil {
		return
	}
	s.mx.Lock()
	defer s.mx.Unlock()

	s.base.SetID(s.id)
}

func (s *store) Close() error {
	s.mx.Lock()
	defer s.mx.Unlock()

	if s.cn != nil {
		s.cn()
	}
	if s.cli != nil {
		err := s.cli.Close()
		s.cli = nil
		return err
	}
	return nil
}

func (s *store) Has(key string) (bool, error) {
	if err := s.waitReady(); err != nil {
		return false, err
	}
	s.mx.Lock()
	defer s.mx.Unlock()

	return s.base.Has(key)
}

func (s *store) Get(key string, to interface{}) error {
	if err := s.waitReady(); err != nil {
		return err
	}
	s.mx.Lock()
	defer s.mx.Unlock()

	return s.base.Get(key, to)
}

type queryResult struct {
	Found  bool `json:"found"`
	Source struct {
		Value json.RawMessage `json:"v"`
	} `json:"_source"`
}

type doc struct {
	Value     any `struct:"v"`
	UpdatedAt any `struct:"updated_at"`
}

type entry struct {
	value interface{}
}

func (e entry) Decode(to interface{}) error {
	return typeconv.Convert(to, e.value)
}

func renderRequest(val interface{}) doc {
	return doc{
		Value:     val,
		UpdatedAt: time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
	}
}

func (s *store) Set(key string, value interface{}) error {
	if err := s.waitReady(); err != nil {
		return err
	}
	s.mx.Lock()
	defer s.mx.Unlock()

	return s.base.Set(key, value)
}

func (s *store) Remove(key string) error {
	if err := s.waitReady(); err != nil {
		return err
	}
	s.mx.Lock()
	defer s.mx.Unlock()

	return s.base.Remove(key)
}

type searchResult struct {
	ID     string `json:"_id"`
	Source struct {
		Value json.RawMessage `json:"v"`
	} `json:"_source"`
}

func (s *store) Each(fn func(string, backend.ValueDecoder) (bool, error)) error {
	if err := s.waitReady(); err != nil {
		return err
	}

	s.mx.Lock()
	defer s.mx.Unlock()

	return s.base.Each(fn)
}

func (s *store) configure(ctx context.Context, c *conf.C) {
	s.log.Info("Configuring ES store")
	s.mx.Lock()
	defer s.mx.Unlock()

	if s.cli != nil {
		_ = s.cli.Close()
		s.cli = nil
	}
	s.cliErr = nil

	cli, err := eslegclient.NewConnectedClient(ctx, c, s.name, s.log)
	if err != nil {
		s.log.Errorf("ES store, failed to create elasticsearch client: %v", err)
		s.cliErr = err
	} else {
		s.base = NewStore(ctx, s.log, cli, s.name)
		if s.id != "" {
			s.base.SetID(s.id)
		}
		s.cli = cli
	}

	// Signal store is ready
	s.once.Do(func() {
		close(s.chReady)
	})

}

func (s *store) loop(ctx context.Context, cn context.CancelFunc, unsubFn UnsubscribeFunc, chCfg chan *conf.C) {
	defer cn()

	// Unsubscribe on exit
	defer unsubFn()

	defer s.log.Debug("ES store exit main loop")

	for {
		select {
		case <-ctx.Done():
			return
		case cu := <-chCfg:
			s.configure(ctx, cu)
		}
	}
}
