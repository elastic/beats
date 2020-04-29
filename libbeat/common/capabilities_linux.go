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

// +build linux

package common

import (
	"github.com/pkg/errors"

	"github.com/elastic/go-sysinfo"
	"github.com/elastic/go-sysinfo/types"
)

// Capabilities contains the capability sets of a process
type Capabilities types.CapabilityInfo

// Check performs a permission check for a given capabilities set
func (c Capabilities) Check(set []string) bool {
	for _, capability := range set {
		found := false
		for _, effective := range c.Effective {
			if capability == effective {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// GetCapabilities gets the capabilities of this process
func GetCapabilities() (Capabilities, error) {
	p, err := sysinfo.Self()
	if err != nil {
		return Capabilities{}, errors.Wrap(err, "failed to read self process information")
	}

	if c, ok := p.(types.Capabilities); ok {
		capabilities, err := c.Capabilities()
		return Capabilities(*capabilities), errors.Wrap(err, "failed to read process capabilities")
	}

	return Capabilities{}, errors.New("capabilities not available")
}
