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

package encoding

import "golang.org/x/text/encoding"

// mixed encoder is a copy of encoding.Replacement
// The difference between the two is that for the Decoder the Encoder is used
// The reasons is that during decoding UTF-8 we want to have the behaviour of the encoding,
// means copying all and replacing invalid UTF-8 chars.
type mixed struct{}

func (mixed) NewDecoder() *encoding.Decoder {
	return &encoding.Decoder{Transformer: encoding.Replacement.NewEncoder().Transformer}
}

func (mixed) NewEncoder() *encoding.Encoder {
	return &encoding.Encoder{Transformer: encoding.Replacement.NewEncoder().Transformer}
}
