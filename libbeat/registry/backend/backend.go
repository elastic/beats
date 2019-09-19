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
	Access(name string) (Store, error)

	// Close must release any resource held by the backend.
	Close() error
}

// Key represents the internal key.
type Key string

// ValueDecoder is used to decode values into go structs or maps within a transaction.
// A ValueDecoder must be invalidated once the owning transaction has been closed.
type ValueDecoder interface {
	Decode(to interface{}) error
}

// Store is used to create a transaction in a target storage (e.g.
// index/table/directory).
type Store interface {
	Close() error
	Begin(readonly bool) (Tx, error)
}

// Tx implements the actual access to a store.
// Transactions must be isolated and updates atomic. Transaction must guarantee
// consistent and valid state when re-opening a registry/store after restarts.
type Tx interface {
	Close() error
	Rollback() error
	Commit() error

	Has(key Key) (bool, error)
	Get(key Key) (ValueDecoder, error)
	Set(key Key, from interface{}) error
	Update(key Key, fields interface{}) error
	Remove(Key) error

	EachKey(internal bool, fn func(Key) (bool, error)) error
	Each(internal bool, fn func(Key, ValueDecoder) (bool, error)) error
}
