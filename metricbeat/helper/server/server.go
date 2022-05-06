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

package server

import "github.com/elastic/elastic-agent-libs/mapstr"

type Meta mapstr.M

const (
	EventDataKey = "data"
)

// Server is an interface that can be used to implement servers which can accept data.
type Server interface {
	// Start is used to start the server at a well defined port.
	Start() error
	// Stop the server.
	Stop()
	// Get a channel of events.
	GetEvents() chan Event
}

// Event is an interface that can be used to get the event and event source related information.
type Event interface {
	// Get the raw bytes of the event.
	GetEvent() mapstr.M
	// Get any metadata associated with the data that was received. Ex: client IP for udp message,
	// request/response headers for HTTP call.
	GetMeta() Meta
}
