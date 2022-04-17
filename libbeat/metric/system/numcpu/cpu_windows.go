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

package numcpu

import (
	"github.com/pkg/errors"

	"github.com/menderesk/gosigar/sys/windows"
)

// getCPU implements NumCPU on windows
// For now, this is a bit of a hack that just asks for per-CPU performance data, and reports the CPU count
func getCPU() (int, bool, error) {

	// get per-cpu data
	cpus, err := windows.NtQuerySystemProcessorPerformanceInformation()
	if err != nil {
		return -1, false, errors.Wrap(err, "NtQuerySystemProcessorPerformanceInformation failed")
	}

	return len(cpus), true, nil

}
