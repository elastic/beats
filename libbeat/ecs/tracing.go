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

// Distributed tracing makes it possible to analyze performance throughout a
// microservice architecture all in one view. This is accomplished by tracing
// all of the requests - from the initial web request in the front-end service
// - to queries made through multiple back-end services.
// Unlike most field sets in ECS, the tracing fields are *not* nested under the
// field set name. In other words, the correct field name is `trace.id`, not
// `tracing.trace.id`, and so on.
type Tracing struct {
	// Unique identifier of the trace.
	// A trace groups multiple events like transactions that belong together.
	// For example, a user request handled by multiple inter-connected
	// services.
	TraceID string `ecs:"trace.id"`

	// Unique identifier of the transaction within the scope of its trace.
	// A transaction is the highest level of work measured within a service,
	// such as a request to a server.
	TransactionID string `ecs:"transaction.id"`

	// Unique identifier of the span within the scope of its trace.
	// A span represents an operation within a transaction, such as a request
	// to another service, or a database query.
	SpanID string `ecs:"span.id"`
}
