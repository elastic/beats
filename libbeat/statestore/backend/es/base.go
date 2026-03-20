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

	"github.com/elastic/beats/v7/libbeat/esleg/eslegclient"
	"github.com/elastic/beats/v7/libbeat/statestore/backend"
	"github.com/elastic/elastic-agent-libs/logp"
)

type baseStore struct {
	ctx   context.Context
	cli   *eslegclient.Connection
	name  string
	index string
	log   *logp.Logger
}

const docType = "_doc"

func renderIndexName(name string) string {
	return "agentless-state-" + name
}

func NewStore(ctx context.Context, log *logp.Logger, cli *eslegclient.Connection, name string) *baseStore {
	return &baseStore{
		cli:   cli,
		name:  name,
		index: renderIndexName(name),
		log:   log,
		ctx:   ctx,
	}
}

func (b *baseStore) Get(key string, to interface{}) error {
	if b == nil {
		return nil
	}

	return b.get(key, to)
}

func (b *baseStore) get(key string, to interface{}) error {
	status, data, err := b.cli.Request("GET", fmt.Sprintf("/%s/%s/%s", b.index, docType, url.QueryEscape(key)), "", nil, nil)

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

func (b *baseStore) Has(key string) (bool, error) {
	// Do nothing for now if the store was not initialized
	if b == nil {
		return false, nil
	}

	var v interface{}
	err := b.get(key, &v)
	if err != nil {
		if errors.Is(err, ErrKeyUnknown) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (b *baseStore) Each(fn func(string, backend.ValueDecoder) (bool, error)) error {
	// Do nothing for now if the store was not initialized
	if b == nil {
		return nil
	}

	status, result, err := b.cli.SearchURIWithBody(b.index, "", nil, map[string]any{
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

func (b *baseStore) Set(key string, value interface{}) error {
	if b == nil {
		return nil
	}

	doc := renderRequest(value)
	_, _, err := b.cli.Request("PUT", fmt.Sprintf("/%s/%s/%s", b.index, docType, url.QueryEscape(key)), "", nil, doc)
	if err != nil {
		return err
	}
	return nil
}

func (b *baseStore) Remove(key string) error {
	if b == nil {
		return nil
	}

	_, _, err := b.cli.Delete(b.index, docType, url.QueryEscape(key), nil)
	if err != nil {
		return err
	}
	return nil
}

func (b *baseStore) Close() error {
	return nil
}

func (b *baseStore) SetID(id string) {
	if b == nil {
		return
	}

	if id == "" {
		return
	}
	b.index = renderIndexName(id)
}
