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

package feature

// Bundle defines a list of features available in the current beat.
type Bundle struct {
	features []Featurable
}

// NewBundle creates a new Bundle of feature to be registered.
func NewBundle(features ...Featurable) *Bundle {
	return &Bundle{features: features}
}

// Features returns the interface features slice so
func (b *Bundle) Features() []Featurable {
	return b.features
}
