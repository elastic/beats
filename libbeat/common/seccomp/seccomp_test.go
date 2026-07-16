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

package seccomp

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/go-seccomp-bpf"
)

func TestMustRegisterPolicy(t *testing.T) {
	policyA := testPolicy(seccomp.ActionErrno, []string{"read", "write"})
	policyASame := testPolicy(seccomp.ActionErrno, []string{"read", "write"})
	policyB := testPolicy(seccomp.ActionAllow, []string{"read"})

	t.Run("registers first policy", func(t *testing.T) {
		resetRegisteredPolicyForTest(t)
		MustRegisterPolicy(policyA)
		assert.Same(t, policyA, registeredPolicy, "registered policy pointer")
	})

	t.Run("re-registering an identical policy is a no-op", func(t *testing.T) {
		resetRegisteredPolicyForTest(t)
		MustRegisterPolicy(policyA)
		assert.NotPanics(t, func() { MustRegisterPolicy(policyASame) })
	})

	t.Run("panics on a different policy", func(t *testing.T) {
		resetRegisteredPolicyForTest(t)
		MustRegisterPolicy(policyA)
		assert.PanicsWithError(t, "a different seccomp policy is already registered",
			func() { MustRegisterPolicy(policyB) })
	})

	t.Run("panics on a nil policy", func(t *testing.T) {
		resetRegisteredPolicyForTest(t)
		assert.PanicsWithError(t, "seccomp policy cannot be nil",
			func() { MustRegisterPolicy(nil) })
	})
}

func resetRegisteredPolicyForTest(t *testing.T) {
	t.Helper()
	registeredPolicy = nil
}

func testPolicy(defaultAction seccomp.Action, names []string) *seccomp.Policy {
	return &seccomp.Policy{
		DefaultAction: defaultAction,
		Syscalls: []seccomp.SyscallGroup{
			{
				Action: seccomp.ActionAllow,
				Names:  names,
			},
		},
	}
}
