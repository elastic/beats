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

package v2

import (
	"sync"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/go-concert/chorus"
)

// managedPipeline is used by the internal runners to wrap the pipeline of an Input.
// The runners take care of closing all open beat.Client objects after the input has
// stopped. The runners close context is installed into a beat.Client automatically,
// if no custom CloseRef is configured. This ensure close signal propagation
// when the input is closed, potentially unblocking inputs waiting on events to
// be published.
type managedPipeline struct {
	pipeline beat.Pipeline

	closer *chorus.Closer

	mu      sync.Mutex
	clients []managedClient
}

type managedClient struct {
	pipeline *managedPipeline
	id       int
	beat.Client
}

func (p *managedPipeline) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// TODO: multierr
	var err error
	for _, client := range p.clients {
		closeErr := client.Client.Close()
		if err != nil {
			err = closeErr
		}
	}
	p.clients = nil
	return err
}

func (p *managedPipeline) Connect() (beat.Client, error) {
	return p.ConnectWith(beat.ClientConfig{})
}

func (p *managedPipeline) ConnectWith(cfg beat.ClientConfig) (beat.Client, error) {
	if cfg.CloseRef == nil {
		cfg.CloseRef = p.closer
	}

	client, err := p.pipeline.ConnectWith(cfg)
	if err != nil {
		return nil, err
	}

	managedClient := &managedClient{
		pipeline: p,
		id:       len(p.clients),
		Client:   client,
	}
	p.clients = append(p.clients, managedClient)
	return managedClient, nil
}

func (p *managedPipeline) unregister(client *managedClient) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.clients) == 0 {
		// pipeline was closed concurrently to the unregister call. We are done here
		return false
	}

	end := len(p.clients) - 1
	p.clients[client.id] = p.clients[end]
	p.clients[client.id].id = client.id
	p.clients[end] = nil
	p.clients = p.clients[:end]
	return true
}

func (c *managedClient) Close() error {
	if !c.pipeline.unregister(c) {
		return nil
	}
	return c.Client.Close()
}
