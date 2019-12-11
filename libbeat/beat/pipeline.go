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

	"github.com/elastic/beats/libbeat/common"
)

type Pipeline interface {
	PipelineConnector
	SetACKHandler(PipelineACKHandler) error
}

// PipelineConnector creates a publishing Client. This is typically backed by a Pipeline.
type PipelineConnector interface {
	ConnectWith(ClientConfig) (Client, error)
	Connect() (Client, error)
}

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

	// Events configures callbacks for common client callbacks
	Events ClientEventer

	// ACK handler strategies.
	// Note: ack handlers are run in another go-routine owned by the publisher pipeline.
	//       They should not block for to long, to not block the internal buffers for
	//       too long (buffers can only be freed after ACK has been processed).
	// Note: It's not supported to configure multiple ack handler types. Use at
	//       most one.

	// ACKCount reports the number of published events recently acknowledged
	// by the pipeline.
	ACKCount func(int)

	// ACKEvents reports the events private data of recently acknowledged events.
	// Note: The slice passed must be copied if the events are to be processed
	//       after the handler returns.
	ACKEvents func([]interface{})

	// ACKLastEvent reports the last ACKed event out of a batch of ACKed events only.
	// Only the events 'Private' field will be reported.
	ACKLastEvent func(interface{})
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
	Meta common.MapStr

	// Fields provides additional 'global' fields to be added to every event
	Fields common.MapStr

	// DynamicFields provides additional fields to be added to every event, supporting live updates
	DynamicFields *common.MapStrPointer

	// Processors passes additional processor to the client, to be executed before
	// the pipeline processors.
	Processor ProcessorList

	// KeepNull determines whether published events will keep null values or omit them.
	KeepNull bool

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

// PipelineACKHandler configures some pipeline-wide event ACK handler.
type PipelineACKHandler struct {
	// ACKCount reports the number of published events recently acknowledged
	// by the pipeline.
	ACKCount func(int)

	// ACKEvents reports the events recently acknowledged by the pipeline.
	// Only the events 'Private' field will be reported.
	ACKEvents func([]interface{})

	// ACKLastEvent reports the last ACKed event per pipeline client.
	// Only the events 'Private' field will be reported.
	ACKLastEvents func([]interface{})
}

type ProcessorList interface {
	Processor
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
