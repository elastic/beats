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

package codec

import "github.com/elastic/go-structform"

// NewTypeCodec creates a type visitor that targets go-structform for the given T.
// Note that the generic T in OnVisitFunc is passed as pointer of type *T.
func NewTypeCodec[T any](f OnVisitFunc[*T]) *ExtVisitor[T] {
	return &ExtVisitor[T]{
		f: f,
	}
}

type ExtVisitor[T any] struct {
	f OnVisitFunc[*T]
}

func (v *ExtVisitor[T]) Codec() any {
	return v.f
}

type OnVisitFunc[T any] func(t T, v structform.ExtVisitor) error
