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

package elasticsearch

import (
	"sync"

	"github.com/menderesk/beats/v7/libbeat/esleg/eslegclient"

	"github.com/gofrs/uuid"
)

// ConnectCallback defines the type for the function to be called when the Elasticsearch client successfully connects to the cluster
type ConnectCallback func(*eslegclient.Connection) error

// Callbacks must not depend on the result of a previous one,
// because the ordering is not fixed.
type callbacksRegistry struct {
	callbacks map[uuid.UUID]ConnectCallback
	mutex     sync.Mutex
}

// XXX: it would be fantastic to do this without a package global
var connectCallbackRegistry = newCallbacksRegistry()

// NOTE(ph): We need to refactor this, right now this is the only way to ensure that every calls
// to an ES cluster executes a callback.
var globalCallbackRegistry = newCallbacksRegistry()

func newCallbacksRegistry() callbacksRegistry {
	return callbacksRegistry{
		callbacks: make(map[uuid.UUID]ConnectCallback),
	}
}

// RegisterGlobalCallback register a global callbacks.
func RegisterGlobalCallback(callback ConnectCallback) (uuid.UUID, error) {
	globalCallbackRegistry.mutex.Lock()
	defer globalCallbackRegistry.mutex.Unlock()

	// find the next unique key
	var key uuid.UUID
	var err error
	exists := true
	for exists {
		key, err = uuid.NewV4()
		if err != nil {
			return uuid.Nil, err
		}
		_, exists = globalCallbackRegistry.callbacks[key]
	}

	globalCallbackRegistry.callbacks[key] = callback
	return key, nil
}

// RegisterConnectCallback registers a callback for the elasticsearch output
// The callback is called each time the client connects to elasticsearch.
// It returns the key of the newly added callback, so it can be deregistered later.
func RegisterConnectCallback(callback ConnectCallback) (uuid.UUID, error) {
	connectCallbackRegistry.mutex.Lock()
	defer connectCallbackRegistry.mutex.Unlock()

	// find the next unique key
	var key uuid.UUID
	var err error
	exists := true
	for exists {
		key, err = uuid.NewV4()
		if err != nil {
			return uuid.Nil, err
		}
		_, exists = connectCallbackRegistry.callbacks[key]
	}

	connectCallbackRegistry.callbacks[key] = callback
	return key, nil
}

// DeregisterGlobalCallback deregisters a callback for the elasticsearch output
// specified by its key. If a callback does not exist, nothing happens.
func DeregisterGlobalCallback(key uuid.UUID) {
	globalCallbackRegistry.mutex.Lock()
	defer globalCallbackRegistry.mutex.Unlock()

	delete(globalCallbackRegistry.callbacks, key)
}

// DeregisterConnectCallback deregisters a callback for the elasticsearch output
// specified by its key. If a callback does not exist, nothing happens.
func DeregisterConnectCallback(key uuid.UUID) {
	connectCallbackRegistry.mutex.Lock()
	defer connectCallbackRegistry.mutex.Unlock()

	delete(connectCallbackRegistry.callbacks, key)
}
