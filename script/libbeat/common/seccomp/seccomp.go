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
	"runtime"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/go-seccomp-bpf"
)

// PolicyChangeType specifies the type of change to make to a seccomp policy.
type PolicyChangeType uint8

const (
	// AddSyscall changes a policy by adding a syscall.
	AddSyscall PolicyChangeType = iota
)

var (
	defaultPolicy    *seccomp.Policy
	registeredPolicy *seccomp.Policy
)

// MustRegisterPolicy registers a seccomp policy to use instead of the default
// policy. This can be used to register an application specific seccomp policy
// that is tailored to the specific system calls that the application requires.
// It panics if a policy has already been registered or if the given policy
// is invalid.
func MustRegisterPolicy(p *seccomp.Policy) {
	if p == nil {
		panic(errors.New("seccomp policy cannot be nil"))
	}

	if registeredPolicy != nil {
		panic(errors.New("a seccomp policy is already registered"))
	}

	// Ensure that the policy is valid and usable.
	if _, err := p.Assemble(); err != nil {
		panic(errors.Wrap(err, "failed to register seccomp policy"))
	}
	registeredPolicy = p
}

// LoadFilter loads a seccomp system call filter into the kernel for this
// process. This feature is only available on Linux 3.17+. If c is nil or does
// not contain a seccomp policy then a default policy will be used.
//
// An error is returned if there is a config validation problem. Otherwise any
// errors interfacing with the kernel are logged (i.e. it is non-fatal if
// seccomp cannot be setup).
//
// Policy precedence order (highest to lowest):
// - Policy values from config
// - Application registered policy
// - Default policy (a simple blacklist)
func LoadFilter(c *common.Config) error {
	// Bail out if seccomp.enabled=false.
	if c != nil && !c.Enabled() {
		return nil
	}

	p, err := getPolicy(c)
	if err != nil {
		return err
	}

	loadFilter(p)
	return nil
}

// loadFilter loads a system call filter.
func loadFilter(p *seccomp.Policy) {
	log := logp.NewLogger("seccomp")

	if runtime.GOOS != "linux" {
		log.Debug("Syscall filtering is only supported on Linux")
		return
	}

	if !seccomp.Supported() {
		log.Info("Syscall filter could not be installed because the kernel " +
			"does not support seccomp")
		return
	}

	if p == nil {
		log.Debug("No seccomp policy is defined")
		return
	}

	filter := seccomp.Filter{
		NoNewPrivs: true,
		Flag:       seccomp.FilterFlagTSync,
		Policy:     *p,
	}

	log.Debugw("Loading syscall filter", "seccomp_filter", filter)
	if err := seccomp.LoadFilter(filter); err != nil {
		log.Warn("Syscall filter could not be installed", "error", err,
			"seccomp_filter", filter)
		return
	}

	log.Infow("Syscall filter successfully installed")
}

func getPolicy(c *common.Config) (*seccomp.Policy, error) {
	policy := defaultPolicy
	if registeredPolicy != nil {
		policy = registeredPolicy
	}

	if c != nil && (c.HasField("default_action") || c.HasField("syscalls")) {
		if policy == nil {
			policy = &seccomp.Policy{}
		}

		if err := c.Unpack(policy); err != nil {
			return nil, err
		}
	}

	return policy, nil
}

// ModifyDefaultPolicy modifies the syscalls in the default policy. Any callers
// of this function must first check the architecture because policies are
// architecture specific.
func ModifyDefaultPolicy(changeType PolicyChangeType, syscalls ...string) error {
	if defaultPolicy == nil {
		return errors.New("no default policy exists (check the architecture)")
	}

	switch changeType {
	case AddSyscall:
		list := defaultPolicy.Syscalls[0].Names
		for _, newSyscall := range syscalls {
			found := false
			for _, existingSyscall := range list {
				if found = newSyscall == existingSyscall; found {
					break
				}
			}
			if !found {
				list = append(list, newSyscall)
			}
		}
		defaultPolicy.Syscalls[0].Names = list

	default:
		return errors.New("unsupported PolicyChangeType value")
	}

	return nil
}
