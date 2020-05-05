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

package model

// StringMap is a slice-representation of map[string]string,
// optimized for fast JSON encoding.
//
// Slice items are expected to be ordered by key.
type StringMap []StringMapItem

// StringMapItem holds a string key and value.
type StringMapItem struct {
	// Key is the map item's key.
	Key string

	// Value is the map item's value.
	Value string
}

// IfaceMap is a slice-representation of map[string]interface{},
// optimized for fast JSON encoding.
//
// Slice items are expected to be ordered by key.
type IfaceMap []IfaceMapItem

// IfaceMapItem holds a string key and value.
type IfaceMapItem struct {
	// Key is the map item's key.
	Key string

	// Value is the map item's value.
	Value interface{}
}
