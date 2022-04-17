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
	"runtime"

	"github.com/menderesk/beats/v7/libbeat/logp"
)

// NumCPU is a drop-in replacement for runtime.NumCPU for accurate system config reporting.
// runtime.NumCPU doesn't query any kind of hardware or OS state,
// but merely uses affinity APIs to count what CPUs the given go process is available to run on.
// Most of the time this works okay for reporting metrics, but under certain conditions, such as cases where
// affinity masks are being manually set to manage the go process, or certain job controllers/VMs/etc,
// this number will not reflect the system config.
// Because this is drop-in, it will not return an error.
// if it can't fetch the CPU count the "correct" way, it'll fallback to runtime.NumCPU().
func NumCPU() int {
	count, exists, err := getCPU()
	if err != nil {
		logp.L().Debugf("Error fetching CPU count: %s", err)
		return runtime.NumCPU()
	}
	if !exists {
		logp.L().Debugf("Accurate CPU counts not available on platform, falling back to runtime.NumCPU for metrics")
		return runtime.NumCPU()
	}

	return count
}
