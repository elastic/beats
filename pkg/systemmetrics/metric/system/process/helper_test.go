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

package process

import (
	"errors"
	"fmt"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestErrors(t *testing.T) {
	cases := []struct {
		name  string
		check func(t *testing.T)
	}{
		{
			name: "non fatal error",
			check: func(t *testing.T) {
				err := fmt.Errorf("Faced non-fatal error: %w", NonFatalErr{Err: syscall.EPERM})
				require.True(t, isNonFatal(err), "Should be a non fatal error")
			},
		},
		{
			name: "non fatal error - unwrapped",
			check: func(t *testing.T) {
				err := fmt.Errorf("Faced non-fatal error: %w", syscall.EPERM)
				require.True(t, isNonFatal(err), "Should be a non fatal error")
			},
		},
		{
			name: "non fatal error - hierarchy",
			check: func(t *testing.T) {
				err := fmt.Errorf("Faced non-fatal error: %w", syscall.EPERM)
				err2 := errors.Join(toNonFatal(err))
				require.True(t, isNonFatal(err2), "Should be a non fatal error")
			},
		},
		{
			name: "fatal error",
			check: func(t *testing.T) {
				err := fmt.Errorf("Faced fatal error: %w", errors.New("FATAL"))
				err = toNonFatal(err) // shouldn't have any effect as it's a fatal error
				require.Falsef(t, isNonFatal(err), "Should be a fatal error")
			},
		},
		{
			name: "fatal error - hierarchy",
			check: func(t *testing.T) {
				err := fmt.Errorf("Faced fatal error: %w", errors.New("FATAL"))
				err2 := errors.Join(err)
				require.Falsef(t, isNonFatal(err2), "Should be a fatal error")
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, c.check)
	}
}
