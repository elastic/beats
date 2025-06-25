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

//go:build requirefips

package beater

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/cfgfile"
)

type fipsUnawareInput struct{}

func newFIPSUnawareInput() *fipsUnawareInput { return &fipsUnawareInput{} }
func (f *fipsUnawareInput) String() string   { return "fips_unaware_input" }
func (f *fipsUnawareInput) Start()           {}
func (f *fipsUnawareInput) Stop()            {}

type fipsAwareInput struct{ isFIPSCapable bool }

func newFIPSAwareInput(isFIPSCapable bool) *fipsAwareInput {
	return &fipsAwareInput{isFIPSCapable: isFIPSCapable}
}
func (f *fipsAwareInput) String() string      { return "fips_aware_input" }
func (f *fipsAwareInput) Start()              {}
func (f *fipsAwareInput) Stop()               {}
func (f *fipsAwareInput) IsFIPSCapable() bool { return f.isFIPSCapable }

func TestCheckFIPSCapability(t *testing.T) {
	tests := map[string]struct {
		runner      cfgfile.Runner
		expectedErr string
	}{
		"input_is_not_fips_aware": {
			runner:      newFIPSUnawareInput(),
			expectedErr: "",
		},
		"input_is_fips_aware_but_not_fips_capable": {
			runner:      newFIPSAwareInput(false),
			expectedErr: "running a FIPS-capable distribution but input [fips_aware_input] is not FIPS capable",
		},
		"input_is_fips_aware_and_fips_capable": {
			runner:      newFIPSAwareInput(true),
			expectedErr: "",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			err := checkFIPSCapability(test.runner)
			if test.expectedErr == "" {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, test.expectedErr)
			}
		})
	}
}
