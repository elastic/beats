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
// Based on Go's pkg/hash/fnv.
//
// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package apm // import "go.elastic.co/apm"

const (
	offset64 = 14695981039346656037
	prime64  = 1099511628211
)

type fnv1a uint64

func newFnv1a() fnv1a {
	return offset64
}

func (f *fnv1a) add(s string) {
	for i := 0; i < len(s); i++ {
		*f ^= fnv1a(s[i])
		*f *= prime64
	}
}
