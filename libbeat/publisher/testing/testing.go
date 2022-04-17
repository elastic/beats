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

package testing

// ChanClient implements Client interface, forwarding published events to some
import (
	"github.com/menderesk/beats/v7/libbeat/beat"
)

type TestPublisher struct {
	client beat.Client
}

// given channel only.
type ChanClient struct {
	done            chan struct{}
	Channel         chan beat.Event
	publishCallback func(event beat.Event)
}

func PublisherWithClient(client beat.Client) beat.Pipeline {
	return &TestPublisher{client}
}

func (pub *TestPublisher) Connect() (beat.Client, error) {
	return pub.client, nil
}

func (pub *TestPublisher) ConnectWith(_ beat.ClientConfig) (beat.Client, error) {
	return pub.client, nil
}

func NewChanClientWithCallback(bufSize int, callback func(event beat.Event)) *ChanClient {
	chanClient := NewChanClientWith(make(chan beat.Event, bufSize))
	chanClient.publishCallback = callback

	return chanClient
}

func NewChanClient(bufSize int) *ChanClient {
	return NewChanClientWith(make(chan beat.Event, bufSize))
}

func NewChanClientWith(ch chan beat.Event) *ChanClient {
	if ch == nil {
		ch = make(chan beat.Event, 1)
	}
	c := &ChanClient{
		done:    make(chan struct{}),
		Channel: ch,
	}
	return c
}

func (c *ChanClient) Close() error {
	close(c.done)
	return nil
}

// PublishEvent will publish the event on the channel. Options will be ignored.
// Always returns true.
func (c *ChanClient) Publish(event beat.Event) {
	select {
	case <-c.done:
	case c.Channel <- event:
		if c.publishCallback != nil {
			c.publishCallback(event)
			<-c.Channel
		}
	}
}

func (c *ChanClient) PublishAll(event []beat.Event) {
	for _, e := range event {
		c.Publish(e)
	}
}

func (c *ChanClient) ReceiveEvent() beat.Event {
	return <-c.Channel
}
