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

package mage

import (
	"sync"

	"github.com/magefile/mage/mg"
)

var (
	buildMageOnce sync.Once
)

// MageIntegrationTestStep setups mage to be ran.
type MageIntegrationTestStep struct{}

// Name returns the mage name.
func (m *MageIntegrationTestStep) Name() string {
	return "mage"
}

// Use always returns false.
//
// This step should be defined in `StepRequirements` for the tester, for it
// to be used. It cannot be autodiscovered for usage.
func (m *MageIntegrationTestStep) Use(dir string) (bool, error) {
	return false, nil
}

// Setup ensures the mage binary is built.
//
// Multiple uses of this step will only build the mage binary once.
func (m *MageIntegrationTestStep) Setup(_ map[string]string) error {
	// Pre-build a mage binary to execute.
	buildMageOnce.Do(func() { mg.Deps(buildMage) })
	return nil
}

// Teardown does nothing.
func (m *MageIntegrationTestStep) Teardown(_ map[string]string) error {
	return nil
}
