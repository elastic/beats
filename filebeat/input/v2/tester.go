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

package v2

import (
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/go-concert/chorus"
)

// TestRunner can be used to manage the test run of one or multiple inputs.
// It provides stub implementations for common interfaces and signals test shutdown.
type InputTestRunner struct {
	closer *chorus.Closer
	log    *logp.Logger
}

func NewTestRunner(
	closer *chorus.Closer,
	log *logp.Logger,
) *InputTestRunner {
	return &InputTestRunner{
		closer: closer,
		log:    log,
	}
}

func (tr *InputTestRunner) Run(i Input) error {
	panic("TODO: implement me")
}
