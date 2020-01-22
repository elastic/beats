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

// Fail helper can be used by output factories, to create a failure response when
// loading an output must return an error.
func Fail(err error) (Group, error) { return Group{}, err }

// Success create a valid output Group response for a set of client instances.
func Success(batchSize, retry int, clients ...Client) (Group, error) {
	return Group{
		Clients:   clients,
		BatchSize: batchSize,
		Retry:     retry,
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

func SuccessNet(loadbalance bool, batchSize, retry int, netclients []NetworkClient) (Group, error) {
	if !loadbalance {
		return Success(batchSize, retry, NewFailoverClient(netclients))
	}

	clients := NetworkClients(netclients)
	return Success(batchSize, retry, clients...)
}
