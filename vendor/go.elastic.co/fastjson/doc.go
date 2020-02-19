// Copyright 2018 Elasticsearch BV
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package fastjson provides a library for fast JSON encoding,
// optimised for static code generation.
//
// Fastjson functions and interfaces are structured such that
// all encoding appends to a buffer, enabling buffer reuse
// without forcing specific mechanisms such as sync.Pool. This
// enables zero-allocation encoding without incurring any
// concurrency overhead in certain applications.
package fastjson // import "go.elastic.co/fastjson"
