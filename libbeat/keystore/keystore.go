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

package keystore

import (
	"errors"

	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/common/bus"
	"github.com/menderesk/go-ucfg"
	"github.com/menderesk/go-ucfg/parse"
)

var (
	// ErrAlreadyExists is returned when the file already exist at the location.
	ErrAlreadyExists = errors.New("cannot create a new keystore a valid keystore already exist at the location")

	// ErrKeyDoesntExists is returned when the key doesn't exist in the store
	ErrKeyDoesntExists = errors.New("cannot retrieve the key")

	// ErrNotWritable is returned when the keystore is not writable
	ErrNotWritable = errors.New("the configured keystore is not writable")

	// ErrNotWritable is returned when the keystore is not writable
	ErrNotListing = errors.New("the configured keystore is not listing")
)

// Keystore implement a way to securely saves and retrieves secrets to be used in the configuration
// Currently all credentials are loaded upfront and are not lazy retrieved, we will eventually move
// to that concept, so we can deal with tokens that has a limited duration or can be revoked by a
// remote keystore.
type Keystore interface {
	// Retrieve returns a SecureString instance of the searched key or an error.
	Retrieve(key string) (*SecureString, error)

	// GetConfig returns the key value pair in the config format to be merged with other configuration.
	GetConfig() (*common.Config, error)

	// IsPersisted check if the current keystore is persisted.
	IsPersisted() bool
}

type WritableKeystore interface {
	// Store add keys to the keystore, wont be persisted until we save.
	Store(key string, secret []byte) error

	// Delete removes a specific key from the keystore.
	Delete(key string) error

	// Create Allow to create an empty keystore.
	Create(override bool) error

	// Save persist the changes to the keystore.
	Save() error
}

type ListingKeystore interface {
	// List returns the list of keys in the keystore, return an empty list if none is found.
	List() ([]string, error)
}

// Provider for keystore
type Provider interface {
	GetKeystore(event bus.Event) Keystore
}

// ResolverWrap wrap a config resolver around an existing keystore.
func ResolverWrap(keystore Keystore) func(string) (string, parse.Config, error) {
	return func(keyName string) (string, parse.Config, error) {
		key, err := keystore.Retrieve(keyName)

		if err != nil {
			// If we cannot find the key, its a non fatal error
			// and we pass to other resolver.
			if err == ErrKeyDoesntExists {
				return "", parse.DefaultConfig, ucfg.ErrMissing
			}
			return "", parse.DefaultConfig, err
		}

		v, err := key.Get()
		if err != nil {
			return "", parse.DefaultConfig, err
		}

		return string(v), parse.DefaultConfig, nil
	}
}

// AsWritableKeystore casts a keystore to WritableKeystore, returning an ErrNotWritable error if the given keystore does not implement
// WritableKeystore interface
func AsWritableKeystore(store Keystore) (WritableKeystore, error) {
	w, ok := store.(WritableKeystore)
	if !ok {
		return nil, ErrNotWritable
	}
	return w, nil
}

// AsListingKeystore casts a keystore to ListingKeystore, returning an ErrNotListing error if the given keystore does not implement
// ListingKeystore interface
func AsListingKeystore(store Keystore) (ListingKeystore, error) {
	w, ok := store.(ListingKeystore)
	if !ok {
		return nil, ErrNotListing
	}
	return w, nil
}
