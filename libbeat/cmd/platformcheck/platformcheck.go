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

//go:build linux || windows
// +build linux windows

package platformcheck

import (
	"fmt"
	"math/bits"
	"strings"

	"github.com/shirou/gopsutil/host"
)

func CheckNativePlatformCompat() error {
	const compiledArchBits = bits.UintSize // 32 if the binary was compiled for 32 bit architecture.

	if compiledArchBits > 32 {
		// We assume that 64bit binaries can only be run on 64bit systems
		return nil
	}

	arch, err := host.KernelArch()
	if err != nil {
		return err
	}

	if strings.Contains(arch, "64") {
		return fmt.Errorf("trying to run %vBit binary on 64Bit system", compiledArchBits)
	}

	return nil
}
