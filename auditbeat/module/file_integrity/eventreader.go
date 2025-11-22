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

package file_integrity

import (
	"errors"
	"fmt"

	"github.com/elastic/elastic-agent-libs/logp"
)

// backendInitializer is a function that initializes a backend
type backendInitializer func(Config, *logp.Logger) (EventProducer, error)

func NewEventReader(c Config, logger *logp.Logger) (EventProducer, error) {
	// Handle auto backend selection with fallback mechanism
	if c.Backend == BackendAuto || c.Backend == "" {
		return tryBackendsInOrder(supportedBackends, autoBackendOrder, c, logger)
	}

	// Handle explicit backend selection
	return initBackend(supportedBackends, c.Backend, c, logger)
}

// tryBackendsInOrder attempts to initialize backends in the specified order
func tryBackendsInOrder(supportedBackends map[Backend]backendInitializer, backends []Backend, c Config, logger *logp.Logger) (EventProducer, error) {
	logger.Infof("backend auto-selection enabled, trying backends in order: %v", backends)
	var reader EventProducer
	var lastErr error
	for i, backend := range backends {
		l := logger.Named(string(backend))
		initializer, found := supportedBackends[backend]
		if !found {
			// this should never happen
			l.Fatalf("backend %s not supported, this is a bug", backend)
		}
		if reader, lastErr = initializer(c, l); lastErr == nil {
			l.Infof("selected backend: %s", backend)
			return reader, nil
		}

		// Log fallback message unless this is the last backend
		if i < len(backends)-1 {
			logger.Warnf("%s backend not available: %v, falling back to %s",
				backend, lastErr, backends[i+1])
		}
	}
	return nil, fmt.Errorf("all backends failed, last error: %w", lastErr)
}

// initBackend initializes a specific backend
func initBackend(supportedBackends map[Backend]backendInitializer, backend Backend, c Config, logger *logp.Logger) (EventProducer, error) {
	l := logger.Named(string(backend))
	initializer, found := supportedBackends[backend]
	if !found {
		return nil, errors.ErrUnsupported
	}

	reader, err := initializer(c, l)
	if err != nil {
		return nil, err
	}
	l.Infof("selected backend: %s", backend)
	return reader, nil
}
