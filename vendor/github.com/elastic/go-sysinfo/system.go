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

package sysinfo

import (
	"runtime"

	"github.com/elastic/go-sysinfo/internal/registry"
	"github.com/elastic/go-sysinfo/types"

	// Register host and process providers.
	_ "github.com/elastic/go-sysinfo/providers/aix"
	_ "github.com/elastic/go-sysinfo/providers/darwin"
	_ "github.com/elastic/go-sysinfo/providers/linux"
	_ "github.com/elastic/go-sysinfo/providers/windows"
)

// Go returns information about the Go runtime.
func Go() types.GoInfo {
	return types.GoInfo{
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
		MaxProcs: runtime.GOMAXPROCS(0),
		Version:  runtime.Version(),
	}
}

// Host returns information about host on which this process is running. If
// host information collection is not implemented for this platform then
// types.ErrNotImplemented is returned.
func Host() (types.Host, error) {
	provider := registry.GetHostProvider()
	if provider == nil {
		return nil, types.ErrNotImplemented
	}
	return provider.Host()
}

// Process returns a types.Process object representing the process associated
// with the given PID. The types.Process object can be used to query information
// about the process.  If process information collection is not implemented for
// this platform then types.ErrNotImplemented is returned.
func Process(pid int) (types.Process, error) {
	provider := registry.GetProcessProvider()
	if provider == nil {
		return nil, types.ErrNotImplemented
	}
	return provider.Process(pid)
}

// Processes return a list of all processes. If process information collection
// is not implemented for this platform then types.ErrNotImplemented is
// returned.
func Processes() ([]types.Process, error) {
	provider := registry.GetProcessProvider()
	if provider == nil {
		return nil, types.ErrNotImplemented
	}
	return provider.Processes()
}

// Self return a types.Process object representing this process. If process
// information collection is not implemented for this platform then
// types.ErrNotImplemented is returned.
func Self() (types.Process, error) {
	provider := registry.GetProcessProvider()
	if provider == nil {
		return nil, types.ErrNotImplemented
	}
	return provider.Self()
}
