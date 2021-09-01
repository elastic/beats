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

package cpu

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/opt"
)

func scanStatFile(scanner *bufio.Scanner) (CPUMetrics, error) {
	cpuData, err := statScanner(scanner, parseCPULine)
	if err != nil {
		return CPUMetrics{}, errors.Wrap(err, "error scanning stat file")
	}
	return cpuData, nil
}

func parseCPULine(line string) (CPU, error) {
	cpuData := CPU{}
	fields := strings.Fields(line)
	var errs multierror.Errors

	tryParseUint := func(name, field string) (v opt.Uint) {
		u, err := touint(field)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to parse %v: %s", name, field))
		} else {
			v = opt.UintWith(u)
		}
		return v
	}

	cpuData.User = tryParseUint("user", fields[1])
	cpuData.Nice = tryParseUint("nice", fields[2])
	cpuData.Sys = tryParseUint("sys", fields[3])
	cpuData.Idle = tryParseUint("idle", fields[4])

	return cpuData, errs.Err()
}
