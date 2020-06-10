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
	// Access opens a store. The store will be closed by the registry, once all
	// accessors have closed the store.
	//
	// The Store instance returned must be threadsafe.
	Access(name string) (Store, error)

	Close() error
}

// ValueDecoder is used to decode values into go structs or maps within a transaction.
// A ValueDecoder must be invalidated once the owning transaction has been closed.
type ValueDecoder interface {
	Decode(to interface{}) error
}

// Store provides access to key value pairs.
type Store interface {
	// Close should close the store and release all used resources.
	Close() error

	Has(key string) (bool, error)

	Get(key string, into interface{}) error

	Set(key string, from interface{}) error

	Remove(string) error

	Each(fn func(string, ValueDecoder) (bool, error)) error
}
