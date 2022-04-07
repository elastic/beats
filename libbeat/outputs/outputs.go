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

// Package outputs defines common types and interfaces to be implemented by
// output plugins.

package outputs

import (
	"context"

	"github.com/elastic/beats/v8/libbeat/publisher"
)

// Client provides the minimal interface an output must implement to be usable
// with the publisher pipeline.
type Client interface {
	Close() error

	// Publish sends events to the clients sink. A client must synchronously or
	// asynchronously ACK the given batch, once all events have been processed.
	// Using Retry/Cancelled a client can return a batch of unprocessed events to
	// the publisher pipeline. The publisher pipeline (if configured by the output
	// factory) will take care of retrying/dropping events.
	// Context is intended for carrying request-scoped values, not for cancellation.
	Publish(context.Context, publisher.Batch) error

	// String identifies the client type and endpoint.
	String() string
}

// NetworkClient defines the required client capabilities for network based
// outputs, that must be reconnectable.
type NetworkClient interface {
	Client
	Connectable
}

// Connectable is optionally implemented by clients that might be able to close
// and reconnect dynamically.
type Connectable interface {
	// Connect establishes a connection to the clients sink.
	// The connection attempt shall report an error if no connection could been
	// established within the given time interval. A timeout value of 0 == wait
	// forever.
	Connect() error
}
