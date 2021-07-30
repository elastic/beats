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

// Package cursor provides an InputManager for use with the v2 API, that is
// capable of storing an internal cursor state between restarts.
//
// The InputManager requires authors to Implement a configuration function and
// the cursor.Input interface. The configuration function returns a slice of
// sources ([]Source) that it has read from the configuration object, and the
// actual Input that will be used to collect events from each configured
// source.
// When Run a go-routine will be started per configured source. If two inputs have
// configured the same source, only one will be active, while the other waits
// for the resource to become free.
// The manager keeps track of the state per source. When publishing an event a
// new cursor value can be passed as well. Future instance of the input can
// read the last published cursor state.
//
// For each source an in-memory and a persitent state are tracked. Internal
// meta updates by the input manager can not be read by Inputs, and will be
// written to the persistent store immediately. Cursor state updates are read
// and update by the input. Cursor updates are written to the persistent store
// only after the events have been ACKed by the output. Internally the input
// manager keeps track of already ACKed updates and pending ACKs.
// In order to guarantee progress even if the pbulishing is slow or blocked, all cursor
// updates are written to the in-memory state immediately. Source without any
// pending updates are in-sync (in-memory state == persistet state). All
// updates are ordered, but we allow the in-memory state to be ahead of the
// persistent state.
// When an input is started, the cursor state is read from the in-memory state.
// This way a new input instance can continue where other inputs have been
// stopped, even if we still have in-flight events from older input instances.
// The coordination between inputs guarantees that all updates are always in
// order.
//
// When a shutdown signal is received, the publisher is directly disconnected
// from the outputs. As all coordination is directly handled by the
// InputManager, shutdown will be immediate (once the input itself has
// returned), and can not be blocked by the outputs.
//
// An input that is about to collect a source that is already collected by
// another input will wait until the other input has returned or the current
// input did receive a shutdown signal.
package input_logfile
