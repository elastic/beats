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
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"

	"github.com/elastic/beats/v7/libbeat/common/transform/typeconv"
	"github.com/elastic/beats/v7/libbeat/esleg/eslegclient"
	"github.com/elastic/beats/v7/libbeat/statestore/backend"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

// TODO: Possibly add in-memory cache, since the operations could have delays
// for example when the key is deleted, it's still could be searchable until the next refresh
// the refresh delay is even worse for serverless
type store struct {
	ctx      context.Context
	cn       context.CancelFunc
	log      *logp.Logger
	name     string
	index    string
	notifier *Notifier

	chReady chan struct{}
	once    sync.Once

	mx     sync.Mutex
	cli    *eslegclient.Connection
	cliErr error
}

const docType = "_doc"

func openStore(ctx context.Context, log *logp.Logger, name string, notifier *Notifier) (*store, error) {
	ctx, cn := context.WithCancel(ctx)
	s := &store{
		ctx:      ctx,
		cn:       cn,
		log:      log.With("name", name).With("backend", "elasticsearch"),
		name:     name,
		index:    renderIndexName(name),
		notifier: notifier,
		chReady:  make(chan struct{}),
	}

	chCfg := make(chan *conf.C)

	id := s.notifier.Subscribe(func(c *conf.C) {
		select {
		case chCfg <- c:
		case <-ctx.Done():
		}
	})

	go s.loop(ctx, cn, id, chCfg)

	return s, nil
}

func renderIndexName(name string) string {
	return "agentless-state-" + name
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
	defer s.mx.Unlock()

	if id == "" {
		return
	}
	s.index = renderIndexName(id)
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

	var v interface{}
	err := s.get(key, v)
	if err != nil {
		if errors.Is(err, ErrKeyUnknown) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s *store) Get(key string, to interface{}) error {
	if err := s.waitReady(); err != nil {
		return err
	}
	s.mx.Lock()
	defer s.mx.Unlock()

	return s.get(key, to)
}

func (s *store) get(key string, to interface{}) error {
	status, data, err := s.cli.Request("GET", fmt.Sprintf("/%s/%s/%s", s.index, docType, url.QueryEscape(key)), "", nil, nil)

	if err != nil {
		if status == http.StatusNotFound {
			return ErrKeyUnknown
		}
		return err
	}

	var qr queryResult
	err = json.Unmarshal(data, &qr)
	if err != nil {
		return err
	}

	err = json.Unmarshal(qr.Source.Value, to)
	if err != nil {
		return err
	}
	return nil
}

type queryResult struct {
	Found  bool `json:"found"`
	Source struct {
		Value json.RawMessage `json:"v"`
	} `json:"_source"`
}

type doc struct {
	Value any `struct:"v"`
}

type upsertRequest struct {
	Doc    doc `struct:"doc"`
	Upsert doc `struct:"upsert"`
}

type entry struct {
	value interface{}
}

func (e entry) Decode(to interface{}) error {
	return typeconv.Convert(to, e.value)
}

func renderUpsertRequest(val interface{}) upsertRequest {
	d := doc{
		Value: val,
	}
	return upsertRequest{
		Doc:    d,
		Upsert: d,
	}
}

func (s *store) Set(key string, value interface{}) error {
	if err := s.waitReady(); err != nil {
		return err
	}
	s.mx.Lock()
	defer s.mx.Unlock()

	// The advantage of using upsert here is that the seqno doesn't increase if the document is the same
	upsert := renderUpsertRequest(value)
	_, _, err := s.cli.Request("POST", fmt.Sprintf("/%s/%s/%s", s.index, "_update", url.QueryEscape(key)), "", nil, upsert)
	if err != nil {
		return err
	}
	return nil
}

func (s *store) Remove(key string) error {
	if err := s.waitReady(); err != nil {
		return err
	}
	s.mx.Lock()
	defer s.mx.Unlock()

	_, _, err := s.cli.Delete(s.index, docType, url.QueryEscape(key), nil)
	if err != nil {
		return err
	}
	return nil
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

	// Do nothing for now if the store was not initialized
	if s.cli == nil {
		return nil
	}

	status, result, err := s.cli.SearchURIWithBody(s.index, "", nil, map[string]any{
		"query": map[string]any{
			"match_all": map[string]any{},
		},
		"size": 1000, // TODO: we might have to do scroll if there are more than 1000 keys
	})

	if err != nil && status != http.StatusNotFound {
		return err
	}

	if result == nil || len(result.Hits.Hits) == 0 {
		return nil
	}

	for _, hit := range result.Hits.Hits {
		var sres searchResult
		err = json.Unmarshal(hit, &sres)
		if err != nil {
			return err
		}

		var e entry
		err = json.Unmarshal(sres.Source.Value, &e.value)
		if err != nil {
			return err
		}

		key, err := url.QueryUnescape(sres.ID)
		if err != nil {
			return err
		}

		cont, err := fn(key, e)
		if !cont || err != nil {
			return err
		}
	}

	return nil
}

func (s *store) configure(ctx context.Context, c *conf.C) {
	s.mx.Lock()
	defer s.mx.Unlock()

	if s.cli != nil {
		_ = s.cli.Close()
		s.cli = nil
	}
	s.cliErr = nil

	cli, err := eslegclient.NewConnectedClient(ctx, c, s.name)
	if err != nil {
		s.log.Errorf("ES store, failed to create elasticsearch client: %v", err)
		s.cliErr = err
	} else {
		s.cli = cli
	}

	// Signal store is ready
	s.once.Do(func() {
		close(s.chReady)
	})

}

func (s *store) loop(ctx context.Context, cn context.CancelFunc, subId int, chCfg chan *conf.C) {
	defer cn()

	defer s.notifier.Unsubscribe(subId)

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
