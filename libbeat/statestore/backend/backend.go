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

package backend

// Registry provides access to stores managed by the backend storage.
type Registry interface {
	// Access opens a store. The store will be closed by the frontend, once all
	// accessed stores have been closed.
	//
	// The Store instance returned must be threadsafe.
	Access(name string) (Store, error)

	// Close is called on shutdown after all stores have been closed.
	// An implementation of Registry is not required to check for the stores to be closed.
	Close() error
}

// ValueDecoder is used to decode values into go structs or maps within a transaction.
// A ValueDecoder is supposed to be invalidated by beats after the loop operations has returned.
type ValueDecoder interface {
	Decode(to interface{}) error
}

// Store provides access to key value pairs.
type Store interface {
	// Close should close the store and release all used resources.
	Close() error

	// Has checks if the key exists. No error must be returned if the key does
	// not exist, but the bool return must be false.
	// An error return value must indicate internal errors only. The store is
	// assumed to be in a 'bad' but recoverable state if 'Has' fails.
	Has(key string) (bool, error)

	// Get decodes the value for the given key into value.
	// Besides internal implementation specific errors an error is assumed
	// to be returned if the key does not exist or the type of the value
	// passed is incompatible to the actual value in the store (decoding error).
	Get(key string, value interface{}) error

	// Set inserts or overwrites a key pair in the store.
	// The `value` parameters can be assumed to be a struct or a map.  Besides
	// internal implementation specific errors, an error should be returned if
	// the value given can not be encoded.
	Set(key string, value interface{}) error

	// Remove removes and entry from the store.
	Remove(string) error

	// Each loops over all key value pairs in the store calling fn for each pair.
	// The ValueDecoder is used by fn to optionally decode the value into a
	// custom struct or map. The decoder must be executable multiple times, but
	// is assumed to be invalidated once fn returns
	// The loop shall return if fn returns an error or false.
	Each(fn func(string, ValueDecoder) (bool, error)) error

	// SetID Sets the store ID when the full input configuration is acquired.
	// This is needed in order to support Elasticsearch state store naming convention based on the input ID.
	SetID(id string)
}
