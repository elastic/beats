// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package elasticsearchstorage

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/elastic/beats/v7/libbeat/esleg/eslegclient"
	"github.com/elastic/beats/v7/libbeat/statestore/backend"
	"github.com/elastic/elastic-agent-libs/transform/typeconv"
)

type store struct {
	client *eslegclient.Connection
	index  string
}

type queryResult struct {
	Source struct {
		Value json.RawMessage `json:"v"`
	} `json:"_source"`
}

type doc struct {
	Value     any `struct:"v"`
	UpdatedAt any `struct:"updated_at"`
}

var ErrKeyUnknown = errors.New("key unknown")

const docType = "_doc"

var _ backend.Store = (*store)(nil)

func openStore(client *eslegclient.Connection, name string) (*store, error) {
	return &store{
		client: client,
		index:  renderIndexName(name),
	}, nil
}

func (s *store) Close() error {
	return nil
}

func (s *store) Has(key string) (bool, error) {
	var v interface{}
	err := s.get(key, &v)
	if err != nil {
		if errors.Is(err, ErrKeyUnknown) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s *store) Get(key string, to interface{}) error {
	return s.get(key, to)
}

func (s *store) get(key string, to interface{}) error {
	status, data, err := s.client.Request("GET", fmt.Sprintf("/%s/%s/%s", s.index, docType, url.QueryEscape(key)), "", nil, nil)

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

func (s *store) Set(key string, value interface{}) error {
	doc := renderRequest(value)
	_, _, err := s.client.Request("PUT", fmt.Sprintf("/%s/%s/%s", s.index, docType, url.QueryEscape(key)), "", nil, doc)
	if err != nil {
		return err
	}
	return nil
}

func (s *store) Remove(key string) error {
	_, _, err := s.client.Delete(s.index, docType, url.QueryEscape(key), nil)
	if err != nil {
		return err
	}
	return nil
}

type entry struct {
	value interface{}
}

func (e entry) Decode(to interface{}) error {
	return typeconv.Convert(to, e.value)
}

type searchResult struct {
	ID     string `json:"_id"`
	Source struct {
		Value json.RawMessage `json:"v"`
	} `json:"_source"`
}

func (s *store) Each(fn func(string, backend.ValueDecoder) (bool, error)) error {
	status, result, err := s.client.SearchURIWithBody(s.index, "", nil, map[string]any{
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

func (s *store) SetID(id string) {
	if id == "" {
		return
	}
	s.index = renderIndexName(id)
}

func renderIndexName(name string) string {
	return "agentless-state-" + name
}

func renderRequest(val interface{}) doc {
	return doc{
		Value:     val,
		UpdatedAt: time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
	}
}
