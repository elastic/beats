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

package ecs

// This field set is meant to facilitate pivoting around a piece of data.
// Some pieces of information can be seen in many places in an ECS event. To
// facilitate searching for them, store an array of all seen values to their
// corresponding field in `related.`.
// A concrete example is IP addresses, which can be under host, observer,
// source, destination, client, server, and network.forwarded_ip. If you append
// all IPs to `related.ip`, you can then search for a given IP trivially, no
// matter where it appeared, by querying `related.ip:192.0.2.15`.
type Related struct {
	// All of the IPs seen on your event.
	IP []string `ecs:"ip"`

	// All the user names or other user identifiers seen on the event.
	User []string `ecs:"user"`

	// All the hashes seen on your event. Populating this field, then using it
	// to search for hashes can help in situations where you're unsure what the
	// hash algorithm is (and therefore which key name to search).
	Hash []string `ecs:"hash"`

	// All hostnames or other host identifiers seen on your event. Example
	// identifiers include FQDNs, domain names, workstation names, or aliases.
	Hosts []string `ecs:"hosts"`
}
