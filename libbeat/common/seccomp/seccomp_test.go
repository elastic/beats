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

// registerStep is a single registerPolicy call within a test case.
type registerStep struct {
	policy    *seccomp.Policy
	wantPanic string // empty means the registration must succeed
}

// TestRegisterPolicy exercises the registration logic through a local pointer,
// so it never touches the registeredPolicy global.
func TestRegisterPolicy(t *testing.T) {
	t.Parallel()

	// Two independently constructed policies that install the same filter.
	policyA := namesPolicy(seccomp.ActionErrno, seccomp.ActionAllow, "read", "write")
	policyAIdentical := namesPolicy(seccomp.ActionErrno, seccomp.ActionAllow, "read", "write")

	// Variations on policyA that each install a different filter.
	policyDifferentDefault := namesPolicy(seccomp.ActionAllow, seccomp.ActionAllow, "read", "write")
	policyDifferentNames := namesPolicy(seccomp.ActionErrno, seccomp.ActionAllow, "read")
	policyDifferentGroupAction := namesPolicy(seccomp.ActionErrno, seccomp.ActionTrap, "read", "write")

	// Conditional policies exercising NamesWithCondtions: two identical, one
	// differing only in the argument value it matches on.
	policyConditions := conditionPolicy(1)
	policyConditionsIdentical := conditionPolicy(1)
	policyConditionsDifferent := conditionPolicy(2)

	const (
		alreadyRegistered = "a different seccomp policy is already registered"
		cannotBeNil       = "seccomp policy cannot be nil"
	)

	tests := []struct {
		name  string
		steps []registerStep
	}{
		{
			name:  "re-registering an identical policy is a no-op",
			steps: []registerStep{{policy: policyA}, {policy: policyAIdentical}},
		},
		{
			name:  "re-registering an identical conditional policy is a no-op",
			steps: []registerStep{{policy: policyConditions}, {policy: policyConditionsIdentical}},
		},
		{
			name:  "panics on a different default action",
			steps: []registerStep{{policy: policyA}, {policy: policyDifferentDefault, wantPanic: alreadyRegistered}},
		},
		{
			name:  "panics on different names",
			steps: []registerStep{{policy: policyA}, {policy: policyDifferentNames, wantPanic: alreadyRegistered}},
		},
		{
			name:  "panics on the same names with a different action",
			steps: []registerStep{{policy: policyA}, {policy: policyDifferentGroupAction, wantPanic: alreadyRegistered}},
		},
		{
			name:  "panics on different syscall conditions",
			steps: []registerStep{{policy: policyConditions}, {policy: policyConditionsDifferent, wantPanic: alreadyRegistered}},
		},
		{
			name:  "panics on a nil policy",
			steps: []registerStep{{policy: nil, wantPanic: cannotBeNil}},
		},
		{
			name: "panics on an invalid policy",
			steps: []registerStep{{
				policy:    &seccomp.Policy{DefaultAction: seccomp.ActionErrno},
				wantPanic: "failed to register seccomp policy: syscalls must not be empty",
			}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var current *seccomp.Policy
			for i, step := range tc.steps {
				if step.wantPanic != "" {
					assert.PanicsWithError(t, step.wantPanic,
						func() { registerPolicy(&current, step.policy) }, "step %d", i)
					continue
				}
				registerPolicy(&current, step.policy)
				assert.Same(t, step.policy, current, "step %d: stored policy pointer", i)
			}
		})
	}

	// Call MustRegisterPolicy directly
	assert.PanicsWithError(t, cannotBeNil, func() { MustRegisterPolicy(nil) })
}

// namesPolicy builds a policy with a single syscall group matching names.
func namesPolicy(defaultAction, groupAction seccomp.Action, names ...string) *seccomp.Policy {
	return &seccomp.Policy{
		DefaultAction: defaultAction,
		Syscalls: []seccomp.SyscallGroup{{
			Action: groupAction,
			Names:  names,
		}},
	}
}

// conditionPolicy builds a policy whose single group matches "write" only when
// its first argument equals value, exercising the NamesWithCondtions field.
func conditionPolicy(value uint64) *seccomp.Policy {
	return &seccomp.Policy{
		DefaultAction: seccomp.ActionErrno,
		Syscalls: []seccomp.SyscallGroup{{
			Action: seccomp.ActionAllow,
			NamesWithCondtions: []seccomp.NameWithConditions{{
				Name: "write",
				Conditions: seccomp.ArgumentConditions{{
					Argument:  0,
					Operation: seccomp.Equal,
					Value:     value,
				}},
			}},
		}},
	}
}
