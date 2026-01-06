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
//
// This file was contributed to by generative AI

// Package bbolt provides a statestore backend implementation backed by bbolt.
//
// The backend is designed for Filebeat registry storage. It uses one bbolt
// database file per store (e.g. "<root>/<store>.db") and maintains two buckets:
//   - "data":     the JSON-encoded values for each key
//   - "metadata": per-key timestamps used for TTL-based garbage collection
//
// TTL and GC:
// The backend supports disk TTL via a background registry-level goroutine that
// periodically scans each open store and deletes entries that have been inactive
// longer than Settings.DiskTTL. "Inactivity" is tracked via the metadata bucket.
//
// Current scope:
// This package currently implements Phase 1 ("cold storage") plus full-scan GC.
// The in-memory cache layer and incremental GC are planned but not implemented
// yet.
package bbolt
