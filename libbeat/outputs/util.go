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
	"fmt"

	"github.com/elastic/beats/v7/libbeat/management"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
	"github.com/elastic/beats/v7/libbeat/publisher/queue/diskqueue"
	"github.com/elastic/beats/v7/libbeat/publisher/queue/memqueue"
	"github.com/elastic/elastic-agent-libs/config"
)

// Fail helper can be used by output factories, to create a failure response when
// loading an output must return an error.
func Fail(err error) (Group, error) { return Group{}, err }

// Success create a valid output Group response for a set of client
// instances.  The first argument is expected to contain a queue
// config.Namespace.  The queue config is passed to assign the queue
// factory when elastic-agent reloads the output.
func Success(cfg config.Namespace, batchSize, retry int, clients ...Client) (Group, error) {
	var q queue.QueueFactory
	if cfg.IsSet() && cfg.Config().Enabled() {
		switch cfg.Name() {
		case memqueue.QueueType:
			settings, err := memqueue.SettingsForUserConfig(cfg.Config())
			if err != nil {
				return Group{}, fmt.Errorf("unable to get memory queue settings: %w", err)
			}
			q = memqueue.FactoryForSettings(settings)
		case diskqueue.QueueType:
			if management.UnderAgent() {
				return Group{}, fmt.Errorf("disk queue not supported under agent")
			}
			settings, err := diskqueue.SettingsForUserConfig(cfg.Config())
			if err != nil {
				return Group{}, fmt.Errorf("unable to get disk queue settings: %w", err)
			}
			q = diskqueue.FactoryForSettings(settings)
		default:
			return Group{}, fmt.Errorf("unknown queue type: %s", cfg.Name())
		}
	}
	return Group{
		Clients:      clients,
		BatchSize:    batchSize,
		Retry:        retry,
		QueueFactory: q,
	}, nil
}

// NetworkClients converts a list of NetworkClient instances into []Client.
func NetworkClients(netclients []NetworkClient) []Client {
	clients := make([]Client, len(netclients))
	for i, n := range netclients {
		clients[i] = n
	}
	return clients
}

// SuccessNet create a valid output Group and creates client instances
// The first argument is expected to contain a queue config.Namespace.
// The queue config is passed to assign the queue factory when
// elastic-agent reloads the output.
func SuccessNet(cfg config.Namespace, loadbalance bool, batchSize, retry int, netclients []NetworkClient) (Group, error) {

	if !loadbalance {
		return Success(cfg, batchSize, retry, NewFailoverClient(netclients))
	}

	clients := NetworkClients(netclients)
	return Success(cfg, batchSize, retry, clients...)
}
