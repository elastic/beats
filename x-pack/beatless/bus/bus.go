// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package bus

import "github.com/elastic/beats/libbeat/beat"

// Bus is take a source or multiple sources and wait for events, when new events are available,
// the events are send to the publisher pipeline.
type Bus struct {
	client beat.Client
}

// New return a new bus.
func New(client beat.Client) *Bus {
	return &Bus{client: client}
}

// Listen start listening for events from the source.
func (b *Bus) Listen() {}
