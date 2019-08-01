// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package clientvault

import (
	"sync"

	"github.com/pkg/errors"
)

var (
	// ErrClientNotFound signals that client is not present in the vault.
	ErrClientNotFound = errors.New("client not present")
)

// Client is a client stored in client vault.
type Client interface{}

// ConnectionCreator describes a creator of connections.
// ConnectionCreator should be used in client vault to generate new connections.
type ConnectionCreator interface {
	NewConnection(provider ConnectionProvider) (Client, error)
}

// ConnectionProvider is a basic provider everybody needs to implement
// in order to provide a valid connection.
// Minimal set of properties is: address
type ConnectionProvider interface {
	Address() string
}

// closer close the connection
type closer interface {
	Close() error
}

// ClientVault is a common storage for GRPC clients
type ClientVault struct {
	sync.Mutex
	vault   map[string]Client
	factory ConnectionCreator
}

// NewClientVault creates a new instance of client vault
func NewClientVault(connectionFactory ConnectionCreator) (*ClientVault, error) {
	cv := ClientVault{
		vault:   make(map[string]Client),
		factory: connectionFactory,
	}
	return &cv, nil
}

// UpdateClient updates a client, if client is nil it will get removed
func (cv *ClientVault) UpdateClient(id string, provider ConnectionProvider) error {
	cv.Lock()
	defer cv.Unlock()

	if rc, found := cv.vault[id]; found {
		if closeClient, ok := rc.(closer); ok {
			closeClient.Close()
		}
	}

	if provider != nil && provider.Address() != "" {
		c, err := cv.factory.NewConnection(provider)
		if err != nil {
			return errors.Wrap(err, "creating connection")
		}
		cv.vault[id] = c
	} else if _, found := cv.vault[id]; found {
		delete(cv.vault, id)
	}

	return nil
}

// GetClient retrieves a client, id.
// If client is not found ErrClientNotFound is returned.
// id is a hash computed of tags for sidecar identification.
func (cv *ClientVault) GetClient(id string) (Client, error) {
	cv.Lock()
	defer cv.Unlock()

	c, found := cv.vault[id]
	if !found {
		return nil, ErrClientNotFound
	}

	return c, nil
}
