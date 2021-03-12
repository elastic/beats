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

package shard

// Information about previously known shards
type knownShards struct {
	ids map[string]bool
}

// Information about the shards tracked in the current iteration of the metricset
type currentShards struct {
	parent   *knownShards
	previous map[string]bool
	current  map[string]bool
}

// Creates a new instance of the MetricSet
func newKnownShards() *knownShards {
	return &knownShards{
		ids: make(map[string]bool),
	}
}

// Starts a new round of collecting shards
func (k *knownShards) startRound() *currentShards {
	// Ids we visited in the previous fetch
	var previous = k.ids
	// We forget the previous round so that if the caller does not call completeRound we play safe and start "fresh"
	k.ids = make(map[string]bool)
	// Struct to keep the current iteration
	return &currentShards{
		parent:   k,
		previous: previous,
		current:  make(map[string]bool),
	}
}

// Processes a new id. Returns whether we should create an event
func (c *currentShards) addID(id string) bool {
	c.current[id] = true
	return !c.previous[id]
}

// Completes a round
func (c *currentShards) done() {
	c.parent.ids = c.current
}
