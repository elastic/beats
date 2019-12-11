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
	"fmt"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	ucfg "github.com/elastic/go-ucfg"
)

var (
	// ErrAlreadyExists is returned when the file already exist at the location.
	ErrAlreadyExists = errors.New("cannot create a new keystore a valid keystore already exist at the location")

	// ErrKeyDoesntExists is returned when the key doesn't exist in the store
	ErrKeyDoesntExists = errors.New("cannot retrieve the key")
)

// Keystore implement a way to securely saves and retrieves secrets to be used in the configuration
// Currently all credentials are loaded upfront and are not lazy retrieved, we will eventually move
// to that concept, so we can deal with tokens that has a limited duration or can be revoked by a
// remote keystore.
type Keystore interface {
	// Store add keys to the keystore, wont be persisted until we save.
	Store(key string, secret []byte) error

	// Retrieve returns a SecureString instance of the searched key or an error.
	Retrieve(key string) (*SecureString, error)

	// Delete removes a specific key from the keystore.
	Delete(key string) error

	// List returns the list of keys in the keystore, return an empty list if none is found.
	List() ([]string, error)

	// GetConfig returns the key value pair in the config format to be merged with other configuration.
	GetConfig() (*common.Config, error)

	// Create Allow to create an empty keystore.
	Create(override bool) error

	// IsPersisted check if the current keystore is persisted.
	IsPersisted() bool

	// Save persist the changes to the keystore.
	Save() error
}

// Packager defines a keystore that we can read the raw bytes and be packaged in an artifact.
type Packager interface {
	Package() ([]byte, error)
	ConfiguredPath() string
}

// Factory Create the right keystore with the configured options.
func Factory(cfg *common.Config, defaultPath string) (Keystore, error) {
	config := defaultConfig

	if cfg == nil {
		cfg = common.NewConfig()
	}
	err := cfg.Unpack(&config)

	if err != nil {
		return nil, fmt.Errorf("could not read keystore configuration, err: %v", err)
	}

	if config.Path == "" {
		config.Path = defaultPath
	}

	logp.Debug("keystore", "Loading file keystore from %s", config.Path)
	keystore, err := NewFileKeystore(config.Path)
	return keystore, err
}

// ResolverFromConfig create a resolver from a configuration.
func ResolverFromConfig(cfg *common.Config, dataPath string) (func(string) (string, error), error) {
	keystore, err := Factory(cfg, dataPath)

	if err != nil {
		return nil, err
	}

	return ResolverWrap(keystore), nil
}

// ResolverWrap wrap a config resolver around an existing keystore.
func ResolverWrap(keystore Keystore) func(string) (string, error) {
	return func(keyName string) (string, error) {
		key, err := keystore.Retrieve(keyName)

		if err != nil {
			// If we cannot find the key, its a non fatal error
			// and we pass to other resolver.
			if err == ErrKeyDoesntExists {
				return "", ucfg.ErrMissing
			}
			return "", err
		}

		v, err := key.Get()
		if err != nil {
			return "", err
		}

		logp.Debug("keystore", "accessing key '%s' from the keystore", keyName)
		return string(v), nil
	}
}
