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

//+build !linux !cgo

package reader

import (
	"errors"

	"github.com/elastic/beats/journalbeat/checkpoint"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/logp"
)

// Reader stub for non linux builds.
type Reader struct{}

// New creates a new journal reader and moves the FP to the configured position.
//
// Note: New fails if journalbeat is not compiled for linux
func New(c Config, done chan struct{}, state checkpoint.JournalState, logger *logp.Logger) (*Reader, error) {
	return nil, errors.New("journald reader only supported on linux")
}

// NewLocal creates a reader to read form the local journal and moves the FP
// to the configured position.
//
// Note: NewLocal fails if journalbeat is not compiled for linux
func NewLocal(c Config, done chan struct{}, state checkpoint.JournalState, logger *logp.Logger) (*Reader, error) {
	return nil, errors.New("journald reader only supported on linux")
}

// Next waits until a new event shows up and returns it.
// It blocks until an event is returned or an error occurs.
func (r *Reader) Next() (*beat.Event, error) {
	return nil, nil
}

// Close closes the underlying journal reader.
func (r *Reader) Close() {}
