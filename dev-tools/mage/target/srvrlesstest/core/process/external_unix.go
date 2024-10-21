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

//go:build !windows

package process

import (
	"os"
	"syscall"
	"time"
)

// externalProcess is a watch mechanism used in cases where OS requires  a process to be a child
// for waiting for process. We need to be able to await any process.
func externalProcess(proc *os.Process) {
	if proc == nil {
		return
	}

	for {
		<-time.After(1 * time.Second)
		if proc.Signal(syscall.Signal(0)) != nil {
			// failed to contact process, return
			return
		}
	}
}
