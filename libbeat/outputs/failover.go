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

package outputs

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"

	"github.com/elastic/beats/libbeat/publisher"
	"github.com/elastic/beats/libbeat/testing"
)

type failoverClient struct {
	clients []NetworkClient
	active  int
}

var (
	// ErrNoConnectionConfigured indicates no configured connections for publishing.
	ErrNoConnectionConfigured = errors.New("No connection configured")

	errNoActiveConnection = errors.New("No active connection")
)

// NewFailoverClient combines a set of NetworkClients into one NetworkClient instances,
// with at most one active client. If the active client fails, another client
// will be used.
func NewFailoverClient(clients []NetworkClient) NetworkClient {
	if len(clients) == 1 {
		return clients[0]
	}

	return &failoverClient{
		clients: clients,
		active:  -1,
	}
}

func (f *failoverClient) Connect() error {
	var (
		next   int
		active = f.active
		l      = len(f.clients)
	)

	switch {
	case l == 0:
		return ErrNoConnectionConfigured
	case l == 1:
		next = 0
	case l == 2 && 0 <= active && active <= 1:
		next = 1 - active
	default:
		for {
			// Connect to random server to potentially spread the
			// load when large number of beats with same set of sinks
			// are started up at about the same time.
			next = rand.Int() % l
			if next != active {
				break
			}
		}
	}

	client := f.clients[next]
	f.active = next
	return client.Connect()
}

func (f *failoverClient) Close() error {
	if f.active < 0 {
		return errNoActiveConnection
	}
	return f.clients[f.active].Close()
}

func (f *failoverClient) Publish(batch publisher.Batch) error {
	if f.active < 0 {
		batch.Retry()
		return errNoActiveConnection
	}
	return f.clients[f.active].Publish(batch)
}

func (f *failoverClient) Test(d testing.Driver) {
	for i, client := range f.clients {
		c, ok := client.(testing.Testable)
		d.Run(fmt.Sprintf("Client %d", i), func(d testing.Driver) {
			if !ok {
				d.Fatal("output", errors.New("client doesn't support testing"))
			}
			c.Test(d)
		})
	}
}

func (f *failoverClient) String() string {
	names := make([]string, len(f.clients))

	for i, client := range f.clients {
		names[i] = client.String()
	}

	return "failover(" + strings.Join(names, ",") + ")"
}
