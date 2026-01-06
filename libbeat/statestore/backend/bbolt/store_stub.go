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
//
// This file was contributed to by generative AI

package bbolt

import (
	"errors"

	"github.com/elastic/beats/v7/libbeat/statestore/backend"
	"github.com/elastic/elastic-agent-libs/logp"
)

// TODO (Day 1-2 follow-up): replace this stub with a real bbolt-backed store implementation.
// The stub exists so the new package compiles while work proceeds file-by-file.

var errNotImplemented = errors.New("bbolt backend not implemented yet")

type store struct {
	logger *logp.Logger
}

func openStore(logger *logp.Logger, _ string, _ Settings) (*store, error) {
	return &store{logger: logger}, nil
}

func (s *store) Close() error { return nil }

func (s *store) Has(string) (bool, error) { return false, errNotImplemented }

func (s *store) Get(string, any) error { return errNotImplemented }

func (s *store) Set(string, any) error { return errNotImplemented }

func (s *store) Remove(string) error { return errNotImplemented }

func (s *store) Each(func(string, backend.ValueDecoder) (bool, error)) error {
	return errNotImplemented
}

func (s *store) SetID(string) {}

func (s *store) collectGarbage() error { return nil }
