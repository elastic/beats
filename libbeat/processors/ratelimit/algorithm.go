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

package ratelimit

import (
	"fmt"

	"github.com/pkg/errors"

	cfg "github.com/elastic/elastic-agent-libs/config"
)

var registry = make(map[string]constructor, 0)

// algoConfig for rate limit algorithm.
type algoConfig struct {
	// limit is the rate limit to be enforced by the algorithm.
	limit rate

	// config is any algorithm-specific additional configuration.
	config cfg.C
}

// algorithm is the interface that all rate limiting algorithms must
// conform to.
type algorithm interface {
	// IsAllowed accepts a key and returns whether that key is allowed
	// (true) or not (false). If a key is allowed, it means it is NOT
	// rate limited. If a key is not allowed, it means it is being rate
	// limited.
	IsAllowed(uint64) bool
}

type constructor func(algoConfig) (algorithm, error)

func register(id string, ctor constructor) {
	registry[id] = ctor
}

// factory returns the requested rate limiting algorithm, if one is found. If not found,
// an error is returned.
func factory(id string, config algoConfig) (algorithm, error) {
	var ctor constructor
	var found bool
	if ctor, found = registry[id]; !found {
		return nil, fmt.Errorf("rate limiting algorithm '%v' not implemented", id)
	}

	algorithm, err := ctor(config)
	if err != nil {
		return nil, errors.Wrap(err, "could not construct algorithm")
	}

	return algorithm, nil
}
