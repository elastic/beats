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

package otelstorage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/extension/xextension/storage"

	"github.com/elastic/beats/v7/libbeat/common/transform/typeconv"
	"github.com/elastic/beats/v7/libbeat/statestore/backend"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

var errKeyUnknown = errors.New("key unknown")

// storeFromClient implements [backend.Store] on top of an OpenTelemetry [storage.Client].
// Values are JSON objects (maps) matching memlog statestore semantics.
type storeFromClient struct {
	client storage.Client
}

// NewStoreFromClient adapts an OpenTelemetry [storage.Client] to [backend.Store].
// If the client implements ordered iteration (Each), [backend.Store.Each] is supported; otherwise Each returns an error.
func NewStoreFromClient(client storage.Client) backend.Store {
	if client == nil {
		return nil
	}
	return &storeFromClient{client: client}
}

func (s *storeFromClient) Close() error {
	return s.client.Close(context.Background())
}

func (s *storeFromClient) Has(key string) (bool, error) {
	ctx := context.Background()
	b, err := s.client.Get(ctx, key)
	if err != nil {
		return false, err
	}
	return b != nil, nil
}

func (s *storeFromClient) Get(key string, to any) error {
	ctx := context.Background()
	b, err := s.client.Get(ctx, key)
	if err != nil {
		return err
	}
	if b == nil {
		return errKeyUnknown
	}
	var dec map[string]any
	if err := json.Unmarshal(b, &dec); err != nil {
		return fmt.Errorf("failed to unmarshal stored value for key %q: %w", key, err)
	}
	return typeconv.Convert(to, dec)
}

func (s *storeFromClient) Set(key string, value any) error {
	var tmp mapstr.M
	if err := typeconv.Convert(&tmp, value); err != nil {
		return err
	}
	b, err := json.Marshal(tmp)
	if err != nil {
		return err
	}
	return s.client.Set(context.Background(), key, b)
}

func (s *storeFromClient) Remove(key string) error {
	return s.client.Delete(context.Background(), key)
}

func (s *storeFromClient) Each(fn func(string, backend.ValueDecoder) (bool, error)) error {
	walker, ok := s.client.(storage.Walker)
	if !ok {
		return errors.New("otelstorage: storage client does not support Walk")
	}
	return walker.Walk(context.Background(), func(key string, value []byte) ([]*storage.Operation, error) {
		dec := &jsonValueDecoder{raw: value}
		cont, err := fn(key, dec)
		if err != nil {
			return nil, err
		}
		if !cont {
			return nil, storage.SkipAll
		}
		return nil, nil
	})
}

func (s *storeFromClient) SetID(string) {}

type jsonValueDecoder struct {
	raw json.RawMessage
}

func (d *jsonValueDecoder) Decode(to any) error {
	var dec map[string]any
	if err := json.Unmarshal(d.raw, &dec); err != nil {
		return err
	}
	return typeconv.Convert(to, dec)
}
