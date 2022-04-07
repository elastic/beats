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

package cgcommon

import (
	"bufio"
	"fmt"
	"os"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v8/libbeat/opt"
)

// CPUUsage wraps the CPU usage time values for the CPU controller metrics
type CPUUsage struct {
	NS   uint64     `json:"ns" struct:"ns"`
	Pct  opt.Float  `json:"pct,omitempty" struct:"pct,omitempty"`
	Norm opt.PctOpt `json:"norm,omitempty" struct:"norm,omitempty"`
}

// Pressure contains load metrics for a controller,
// Broken apart into 10, 60, and 300 second samples,
// as well as a total time in US
type Pressure struct {
	Ten          opt.Pct  `json:"10,omitempty" struct:"10,omitempty"`
	Sixty        opt.Pct  `json:"60,omitempty" struct:"60,omitempty"`
	ThreeHundred opt.Pct  `json:"300,omitempty" struct:"300,omitempty"`
	Total        opt.Uint `json:"total,omitempty" struct:"total,omitempty"`
}

// IsZero implements the IsZero interface for Pressure
// This is "all or nothing", as pressure stats don't exist on certain systems
// If `total` doesn't exist, that means there's no pressure metrics.
func (p Pressure) IsZero() bool {
	return p.Total.IsZero()
}

// GetPressure takes the path of a *.pressure file and returns a
// map of the pressure (IO contension) stats for the cgroup
// on CPU controllers, the only key will be "some"
// on IO controllers, the keys will be "some" and "full"
// See https://github.com/torvalds/linux/blob/master/Documentation/accounting/psi.rst
func GetPressure(path string) (map[string]Pressure, error) {
	pressureData := make(map[string]Pressure)
	f, err := os.Open(path)
	// pass along any OS open errors directly
	if err != nil {
		return pressureData, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var stallTime string
		data := Pressure{}
		var total uint64
		matched, err := fmt.Sscanf(sc.Text(), "%s avg10=%f avg60=%f avg300=%f total=%d", &stallTime, &data.Ten.Pct, &data.Sixty.Pct, &data.ThreeHundred.Pct, &total)
		if err != nil {
			return pressureData, errors.Wrapf(err, "error scanning file: %s", path)
		}
		// Assume that if we didn't match at least three numbers, something has gone wrong
		if matched < 3 {
			return pressureData, fmt.Errorf("Error: only matched %d fields from file %s", matched, path)
		}
		data.Total = opt.UintWith(total)
		pressureData[stallTime] = data

	}

	return pressureData, nil
}
