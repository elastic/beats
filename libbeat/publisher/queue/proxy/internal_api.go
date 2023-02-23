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

package proxyqueue

// producer -> broker API

type pushRequest struct {
	event    interface{}
	producer *producer

	// After receiving a request, the broker will respond on this channel
	// with whether the new entry was accepted or not.
	responseChan chan bool

	// If canBlock is true, then the broker will store this request until
	// either the request can be accepted or the queue itself is closed.
	// Otherwise it will immediately reject the requst if there is no
	// space in the pending buffer.
	canBlock bool
}

// consumer -> broker API

type getRequest struct {
	responseChan chan *batch // channel to send response to
}
