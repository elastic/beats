// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix

package azureeventhub

import (
	"sync"

	"github.com/elastic/beats/v7/libbeat/beat"
)

// ackClient is a fake beat.Client that ACKs the published messages.
type fakeClient struct {
	sync.Mutex
	publishedEvents []beat.Event
}

func (c *fakeClient) Close() error { return nil }

func (c *fakeClient) Publish(event beat.Event) {
	c.Lock()
	defer c.Unlock()
	c.publishedEvents = append(c.publishedEvents, event)
}

func (c *fakeClient) PublishAll(event []beat.Event) {
	for _, e := range event {
		c.Publish(e)
	}
}
