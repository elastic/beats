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

package opt

// The types here work as wrappers for the sake of go-structform
// Due to structform's streaming type, we can't declare nested types
// inside of struct tag directives, so in the numerious cases where we have
// fields like thing.bytes, thing.pct, etc, use these wrappers to keep it somewhat clean

// BytesOpt wraps a uint64 byte value in an option type
type BytesOpt struct {
	Bytes Uint `json:"bytes" struct:"bytes"`
}

// IsZero returns true if the underlying value nil
func (opt BytesOpt) IsZero() bool {
	return opt.Bytes.IsZero()
}

// Bytes wraps a uint64 byte value
type Bytes struct {
	Bytes uint64 `json:"bytes" struct:"bytes"`
}

// Us wraps a uint64 microseconds value
type Us struct {
	Us uint64 `json:"us" struct:"us"`
}

// Pct wraps a float64 percent value
type Pct struct {
	Pct float64 `json:"pct" struct:"pct"`
}

// PctOpt wraps a float64 percent value in an option type
type PctOpt struct {
	Pct Float `json:"pct" struct:"pct"`
}

// IsZero returns true if the underlying value nil
func (opt PctOpt) IsZero() bool {
	return opt.Pct.IsZero()
}
