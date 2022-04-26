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

package beat

import (
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
)

// Pipeline provides access to libbeat event publishing by creating a Client
// instance.
type Pipeline interface {
	ConnectWith(ClientConfig) (Client, error)
	Connect() (Client, error)
}

type PipelineConnector = Pipeline

// Client holds a connection to the beats publisher pipeline
type Client interface {
	Publish(Event)
	PublishAll([]Event)
	Close() error
}

// ClientConfig defines common configuration options one can pass to
// Pipeline.ConnectWith to control the clients behavior and provide ACK support.
type ClientConfig struct {
	PublishMode PublishMode

	Processing ProcessingConfig

	CloseRef CloseRef

	// WaitClose sets the maximum duration to wait on ACK, if client still has events
	// active non-acknowledged events in the publisher pipeline.
	// WaitClose is only effective if one of ACKCount, ACKEvents and ACKLastEvents
	// is configured
	WaitClose time.Duration

	// Configure ACK callback.
	ACKHandler ACKer

	// Events configures callbacks for common client callbacks
	Events ClientEventer
}

// ACKer can be registered with a Client when connecting to the pipeline.
// The ACKer will be informed when events are added or dropped by the processors,
// and when an event has been ACKed by the outputs.
//
// Due to event publishing and ACKing are asynchronous operations, the
// operations on ACKer are normally executed in different go routines. ACKers
// are required to be multi-threading safe.
type ACKer interface {
	// AddEvent informs the ACKer that a new event has been send to the client.
	// AddEvent is called after the processors have handled the event. If the
	// event has been dropped by the processor `published` will be set to true.
	// This allows the ACKer to do some bookeeping for dropped events.
	AddEvent(event Event, published bool)

	// ACK Events from the output and pipeline queue are forwarded to ACKEvents.
	// The number of reported events only matches the known number of events downstream.
	// ACKers might need to keep track of dropped events by themselves.
	ACKEvents(n int)

	// Close informs the ACKer that the Client used to publish to the pipeline has been closed.
	// No new events should be published anymore. The ACKEvents method still will be actively called
	// as long as there are pending events for the client in the pipeline. The Close signal can be used
	// to supress any ACK event propagation if required.
	// Close might be called from another go-routine than AddEvent and ACKEvents.
	Close()
}

// CloseRef allows users to close the client asynchronously.
// A CloseReg implements a subset of function required for context.Context.
type CloseRef interface {
	Done() <-chan struct{}
	Err() error
}

// ProcessingConfig provides additional event processing settings a client can
// pass to the publisher pipeline on Connect.
type ProcessingConfig struct {
	// EventMetadata configures additional fields/tags to be added to published events.
	EventMetadata common.EventMetadata

	// Meta provides additional meta data to be added to the Meta field in the beat.Event
	// structure.
	Meta mapstr.M

	// Fields provides additional 'global' fields to be added to every event
	Fields mapstr.M

	// DynamicFields provides additional fields to be added to every event, supporting live updates
	DynamicFields *mapstr.MPointer

	// Processors passes additional processor to the client, to be executed before
	// the pipeline processors.
	Processor ProcessorList

	// KeepNull determines whether published events will keep null values or omit them.
	KeepNull bool

	// Disables the addition of host.name if it was enabled for the publisher.
	DisableHost bool

	// Private contains additional information to be passed to the processing
	// pipeline builder.
	Private interface{}
}

// ClientEventer provides access to internal client events.
type ClientEventer interface {
	Closing() // Closing indicates the client is being shutdown next
	Closed()  // Closed indicates the client being fully shutdown

	Published()             // event has been successfully forwarded to the publisher pipeline
	FilteredOut(Event)      // event has been filtered out/dropped by processors
	DroppedOnPublish(Event) // event has been dropped, while waiting for the queue
}

type ProcessorList interface {
	Processor
	Close() error
	All() []Processor
}

// Processor defines the minimal required interface for processor, that can be
// registered with the publisher pipeline.
type Processor interface {
	String() string // print full processor description
	Run(in *Event) (event *Event, err error)
}

// PublishMode enum sets some requirements on the client connection to the beats
// publisher pipeline
type PublishMode uint8

const (
	// DefaultGuarantees are up to the pipeline configuration itself.
	DefaultGuarantees PublishMode = iota

	// OutputChooses mode fully depends on the output and its configuration.
	// Events might be dropped based on the users output configuration.
	// In this mode no events are dropped within the pipeline. Events are only removed
	// after the output has ACKed the events to the pipeline, even if the output
	// did drop the events.
	OutputChooses

	// GuaranteedSend ensures events are retried until acknowledged by the output.
	// Normally guaranteed sending should be used with some client ACK-handling
	// to update state keeping track of the sending status.
	GuaranteedSend

	// DropIfFull drops an event to be send if the pipeline is currently full.
	// This ensures a beats internals can continue processing if the pipeline has
	// filled up. Useful if an event stream must be processed to keep internal
	// state up-to-date.
	DropIfFull
)
