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

package auditd

import (
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/auditbeat/ab"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/metricbeat/mb"
	conf "github.com/elastic/beats/v7/pkg/config"
	"github.com/elastic/beats/v7/pkg/logp/logptest"
	"github.com/elastic/beats/v7/pkg/paths"
)

// New creates the control client before the data client. If the data client
// cannot be created, the control client's socket must not be left open.
//
// The data client is made to fail by leaving the process one free file
// descriptor, which the control client takes. Other threads in the process
// open and close files too, so the control client can lose the descriptor and
// fail, or the data client can be given another and succeed. Neither attempt
// says anything about the cleanup, so both are repeated.
func TestNew_DataClientFailureClosesControlClient(t *testing.T) {
	c, err := conf.NewConfigFrom(getConfig())
	if err != nil {
		t.Fatal(err)
	}
	info := beat.Info{Paths: paths.New(), UserAgent: "test", Logger: logptest.NewTestingLogger(t, "")}

	const attempts = 20
	for range attempts {
		before := socketCount(t)
		release := useUpFileDescriptors(t, 1)
		_, sets, err := mb.NewModule(c, ab.Registry, info)
		release()
		closeClients(sets)
		if err == nil || strings.Contains(err.Error(), "failed to create audit control client") {
			continue
		}

		assert.ErrorContains(t, err, "failed to create audit data client", "error from creating the module")
		assert.Equal(t, before, socketCount(t), "open sockets after a failed New")
		return
	}
	t.Errorf("no attempt in %d reached a failure to create the data client", attempts)
}

// closeClients closes the audit clients of the metric sets, so that sockets
// opened by an attempt that succeeded do not count against the next one.
func closeClients(sets []mb.MetricSet) {
	for _, set := range sets {
		ms, ok := set.(*MetricSet)
		if !ok {
			continue
		}
		ms.control.Close()
		ms.client.Close()
	}
}

// socketCount returns the number of open sockets in the process.
func socketCount(t *testing.T) int {
	t.Helper()
	ents, err := os.ReadDir("/proc/self/fd")
	if err != nil {
		t.Skipf("cannot list open file descriptors: %v", err)
	}
	var n int
	for _, e := range ents {
		link, err := os.Readlink("/proc/self/fd/" + e.Name())
		if err == nil && strings.HasPrefix(link, "socket:") {
			n++
		}
	}
	return n
}

// useUpFileDescriptors lowers the process's file descriptor limit to just
// above the highest descriptor in use, then opens files until none are left
// and closes n of them. It returns a function that undoes this, which is also
// run when the test ends. The limit is for the whole process, so the test must
// not run in parallel with others.
func useUpFileDescriptors(t *testing.T, n int) (release func()) {
	t.Helper()
	ents, err := os.ReadDir("/proc/self/fd")
	if err != nil {
		t.Skipf("cannot list open file descriptors: %v", err)
	}
	var highest int
	for _, e := range ents {
		fd, err := strconv.Atoi(e.Name())
		if err == nil && fd > highest {
			highest = fd
		}
	}

	var orig syscall.Rlimit
	err = syscall.Getrlimit(syscall.RLIMIT_NOFILE, &orig)
	if err != nil {
		t.Skipf("cannot read the file descriptor limit: %v", err)
	}
	lowered := orig
	lowered.Cur = uint64(highest + 1)
	err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &lowered)
	if err != nil {
		t.Skipf("cannot lower the file descriptor limit: %v", err)
	}

	var fillers []*os.File
	for {
		f, err := os.Open(os.DevNull)
		if err != nil {
			break
		}
		fillers = append(fillers, f)
	}
	for range n {
		if len(fillers) == 0 {
			break
		}
		last := len(fillers) - 1
		fillers[last].Close()
		fillers = fillers[:last]
	}

	var once sync.Once
	release = func() {
		once.Do(func() {
			for _, f := range fillers {
				f.Close()
			}
			err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &orig)
			if err != nil {
				t.Errorf("failed to restore the file descriptor limit: %v", err)
			}
		})
	}
	t.Cleanup(release)
	return release
}
