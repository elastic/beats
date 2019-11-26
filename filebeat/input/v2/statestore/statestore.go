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

// Package statestore provides coordinated access to entries in the registry
// via resources.
//
// A Store supports selective update of fields using go structures.
// Data to be stored in the registry should be split into resource identifying
// meta-data and read state data. This allows inputs to have separate
// go-routines for updating resource tracking meta-data (like file path upon
// file renames) and for read state updates (like file offset).
//
// The registry is only eventually consistent to the current state of the
// store. When using (*Resource).Update, both the in-memory state and the
// registry state will be updated immediately. But when using
// (*Resource).UpdateOp, only the in memory state will be updated. The
// registry state must be updated using the ResourceUpdateOp, after the
// associated events have been ACKed by the outputs. Once all pending update operations have been applied
// the in-memory state and the persistent state are assumed to be in-sync, and
// the in-memory state is dropped so to free some memory.
//
// The eventual consistency allows resources to be Unlocked and Locked by another go-routine
// immediately, as the final read state from the former go-routine is available
// right away. The lock guarantees exclusive access. In the meantime older
// updates might still be applied to the registry file, while the new
// go-routine can start creating new update operations concurrently to be
// applied to after already pending updates.
package statestore

// ResourceKey is used to describe an unique resource to be stored in the registry.
type ResourceKey string
