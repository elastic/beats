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

// Package dns implements a processor that can perform DNS lookups by sending
// a DNS request over UDP to a recursive nameserver. Each instance of the
// processor is independent (no shared cache) so it's best to only define one
// instance of the processor.
//
// It caches DNS results in memory and honors the record's TTL. It also caches
// failures for the configured failure TTL.
//
// This filter, like all filters, only processes 1 event at a time, so the use
// of this plugin can significantly slow down your pipeline’s throughput if you
// have a high latency network. By way of example, if each DNS lookup takes 2
// milliseconds, the maximum throughput you can achieve with a single filter
// worker is 500 events per second (1000 milliseconds / 2 milliseconds).
package dns
