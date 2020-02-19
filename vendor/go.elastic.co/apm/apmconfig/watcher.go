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

package apmconfig

import (
	"context"
)

// Watcher provides an interface for watching config changes.
type Watcher interface {
	// WatchConfig subscribes to changes to configuration for the agent,
	// which must match the given ConfigSelector.
	//
	// If the watcher experiences an unexpected error fetching config,
	// it will surface this in a Change with the Err field set.
	//
	// If the provided context is cancelled, or the watcher experiences
	// a fatal condition, the returned channel will be closed.
	WatchConfig(context.Context, WatchParams) <-chan Change
}

// WatchParams holds parameters for watching for config changes.
type WatchParams struct {
	// Service holds the name and optionally environment name used
	// for filtering the config to watch.
	Service struct {
		Name        string
		Environment string
	}
}

// Change holds an agent configuration change: an error or the new config attributes.
type Change struct {
	// Err holds an error that occurred while querying agent config.
	Err error

	// Attrs holds the agent's configuration. May be empty.
	Attrs map[string]string
}
