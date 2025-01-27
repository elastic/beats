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
	"sync"

	"github.com/elastic/beats/v7/libbeat/statestore/backend"
	"github.com/elastic/elastic-agent-libs/logp"
)

type Registry struct {
	ctx context.Context

	log *logp.Logger
	mx  sync.Mutex

	notifier *Notifier
}

func New(ctx context.Context, log *logp.Logger, notifier *Notifier) *Registry {
	return &Registry{
		ctx:      ctx,
		log:      log,
		notifier: notifier,
	}
}

func (r *Registry) Access(name string) (backend.Store, error) {
	r.mx.Lock()
	defer r.mx.Unlock()
	return openStore(r.ctx, r.log, name, r.notifier)
}

func (r *Registry) Close() error {
	return nil
}
