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

package channel

import (
	"testing"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

func TestThing(t *testing.T) {
	done := make(chan struct{})
	beatInfo := beat.Info{Beat: "TestBeat", Version: "3.9.27"}
	outletFactory := NewOutletFactory(done, emptyCounter{}, beatInfo)
	pipeline := newChannelPipeline()
	connector := outletFactory.Create(pipeline)
	config, err := common.NewConfigFrom("index: 'test'")
	if err != nil {
		t.Error(err)
	}
	//field, _ := config.String("index", -1)
	//fmt.Printf("config index: %v\n", field)
	//config := common.NewConfig()
	outleter, err := connector.ConnectWith(
		config,
		beat.ClientConfig{},
	)
	outleter.OnEvent(beat.Event{})
	if err != nil {
		t.Error(err)
	}
	processedEvent := <-pipeline.events
	if processedEvent.Meta == nil {
		//t.Error("Event Meta shouldn't be empty")
	}
}

type emptyCounter struct{}

func (c emptyCounter) Add(n int) {}
func (c emptyCounter) Done()     {}

// channelPipeline is a Pipeline (and Client) whose connections just echo their
// events to a shared events channel for testing.
type channelPipeline struct {
	events chan beat.Event
}

func newChannelPipeline() *channelPipeline {
	return &channelPipeline{make(chan beat.Event, 100)}
}

func (cp *channelPipeline) SetACKHandler(h beat.PipelineACKHandler) error {
	return nil
}

func (cp *channelPipeline) ConnectWith(conf beat.ClientConfig) (beat.Client, error) {
	return cp, nil
}

func (cp *channelPipeline) Connect() (beat.Client, error) {
	return cp, nil
}

func (cp *channelPipeline) Publish(event beat.Event) {
	cp.events <- event
}

func (cp *channelPipeline) PublishAll(events []beat.Event) {
	for _, event := range events {
		cp.Publish(event)
	}
}

func (cp *channelPipeline) Close() error {
	return nil
}
