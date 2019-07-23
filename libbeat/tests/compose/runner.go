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

package compose

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// TestRunner starts a service with different combinations of options and
// runs tests on each one of these combinations
type TestRunner struct {
	// Name of the service managed by this runner
	Service string

	// Map of options with the list of possible values
	Options RunnerOptions

	// Set to true if this runner can run in parallel with other runners
	Parallel bool

	// Timeout to start the managed service
	Timeout int
}

// Suite is a set of tests to be run with a TestRunner
// Each test must be one of:
// - func(R)
// - func(*testing.T, R)
// - func(*testing.T)
type Suite map[string]interface{}

// RunnerOptions are the possible options of a runner scenario
type RunnerOptions map[string][]string

func (r *TestRunner) scenarios() []map[string]string {
	n := 1
	options := make(map[string][]string)
	for env, values := range r.Options {
		// Allow to override options from environment variables
		value := os.Getenv(env)
		if value != "" {
			values = []string{value}
		}
		options[env] = values
		n *= len(values)
	}

	scenarios := make([]map[string]string, n)
	for variable, values := range options {
		v := 0
		for i, s := range scenarios {
			if s == nil {
				s = make(map[string]string)
				scenarios[i] = s
			}
			s[variable] = values[v]
			v = (v + 1) % len(values)
		}
	}

	return scenarios
}

func (r *TestRunner) runSuite(t *testing.T, tests Suite, ctl R) {
	for name, test := range tests {
		var testFunc func(t *testing.T)
		switch f := test.(type) {
		case func(R):
			testFunc = func(t *testing.T) { f(ctl.WithT(t)) }
		case func(*testing.T, R):
			testFunc = func(t *testing.T) { f(t, ctl.WithT(t)) }
		case func(*testing.T):
			testFunc = func(t *testing.T) { f(t) }
		default:
			t.Fatalf("incorrect test suite function '%s'", name)
		}
		t.Run(name, testFunc)
	}
}

func (r *TestRunner) runHostOverride(t *testing.T, tests Suite) bool {
	env := strings.ToUpper(r.Service) + "_HOST"
	host := os.Getenv(env)
	if host == "" {
		return false
	}

	t.Logf("Test host overriden by %s=%s", env, host)

	ctl := &runnerControl{
		host: host,
	}
	r.runSuite(t, tests, ctl)
	return true
}

// Run runs a tests suite
func (r *TestRunner) Run(t *testing.T, tests Suite) {
	t.Helper()

	if r.runHostOverride(t, tests) {
		return
	}

	timeout := r.Timeout
	if timeout == 0 {
		timeout = 300
	}

	scenarios := r.scenarios()
	if len(scenarios) == 0 {
		t.Fatal("Test runner configuration doesn't produce scenarios")
	}
	for _, s := range scenarios {
		s := s
		var vars []string
		for k, v := range s {
			vars = append(vars, fmt.Sprintf("%s=%s", k, v))
		}
		sort.Strings(vars)
		desc := strings.Join(vars, ",")
		if desc == "" {
			desc = "WithoutOptions"
		}

		seq := make([]byte, 16)
		rand.Read(seq)
		name := fmt.Sprintf("%s_%x", r.Service, seq)

		project, err := getComposeProject(name)
		if err != nil {
			t.Fatal(err)
		}
		project.SetParameters(s)

		t.Run(desc, func(t *testing.T) {
			if r.Parallel {
				t.Parallel()
			}

			err := project.Start(r.Service)
			// Down() is "idempotent", Start() has several points where it can fail,
			// so run Down() even if Start() fails.
			defer project.Down()
			if err != nil {
				t.Fatal(err)
			}

			err = project.Wait(timeout, r.Service)
			if err != nil {
				t.Fatal(errors.Wrapf(err, "waiting for %s/%s", r.Service, desc))
			}

			host, err := project.Host(r.Service)
			if err != nil {
				t.Fatal(errors.Wrapf(err, "getting host for %s/%s", r.Service, desc))
			}

			ctl := &runnerControl{
				host:     host,
				scenario: s,
			}
			r.runSuite(t, tests, ctl)
		})
	}
}

// R extends the testing.T interface with methods that expose information about current scenario
type R interface {
	testing.TB

	WithT(t *testing.T) R

	Host() string
	Hostname() string
	Port() string

	Option(string) string
}

type runnerControl struct {
	*testing.T

	host     string
	scenario map[string]string
}

// WithT creates a copy of R with the given T
func (r *runnerControl) WithT(t *testing.T) R {
	ctl := *r
	ctl.T = t
	return &ctl
}

// Host returns the host:port the test should use to connect to the service
func (r *runnerControl) Host() string {
	return r.host
}

// Hostname is the address of the host
func (r *runnerControl) Hostname() string {
	hostname, _, _ := net.SplitHostPort(r.host)
	return hostname
}

// Port is the port of the host
func (r *runnerControl) Port() string {
	_, port, _ := net.SplitHostPort(r.host)
	return port
}

// Option returns the value of an option for the current scenario
func (r *runnerControl) Option(key string) string {
	return r.scenario[key]
}
