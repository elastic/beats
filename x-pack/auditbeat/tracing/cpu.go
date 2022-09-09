// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux
// +build linux

package tracing

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
)

const (
	// OnlineCPUsPath is the path to the system file listing the online CPUs.
	OnlineCPUsPath = "/sys/devices/system/cpu/online"

	// OfflineCPUsPath is the path to the system file listing the offline CPUs.
	OfflineCPUsPath = "/sys/devices/system/cpu/offline"

	// PossibleCPUsPath is the path to the system file listing the CPUs that can be brought online.
	PossibleCPUsPath = "/sys/devices/system/cpu/possible"

	// PresentCPUsPath is the path to the system file listing the CPUs that are identified as present.
	PresentCPUsPath = "/sys/devices/system/cpu/present"

	// See `Documentation/admin-guide/cputopology.rst` in the Linux kernel docs for more information
	// on the above files.

	// IsolatedCPUsPath is only present when CPU isolation is active, for example using the `isolcpus`
	// kernel argument.
	IsolatedCPUsPath = "/sys/devices/system/cpu/isolated"
)

// CPUSet represents a group of CPUs.
type CPUSet struct {
	mask  []bool
	count int
}

// Mask returns a bitmask where each bit is set if the given CPU is present in the set.
func (s CPUSet) Mask() []bool {
	return s.mask
}

// NumCPU returns the number of CPUs in the set.
func (s CPUSet) NumCPU() int {
	return s.count
}

// Contains allows to query if a given CPU exists in the set.
func (s CPUSet) Contains(cpu int) bool {
	if cpu < 0 || cpu >= len(s.mask) {
		return false
	}
	return s.mask[cpu]
}

// AsList returns the list of CPUs in the set.
func (s CPUSet) AsList() []int {
	list := make([]int, 0, s.count)
	for num, bit := range s.mask {
		if bit {
			list = append(list, num)
		}
	}
	return list
}

// NewCPUSetFromFile creates a new CPUSet from the contents of a file.
func NewCPUSetFromFile(path string) (cpus CPUSet, err error) {
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return cpus, err
	}
	return NewCPUSetFromExpression(string(bytes.TrimRight(contents, "\n\r")))
}

// NewCPUSetFromExpression creates a new CPUSet from a range expression.
// Expression: RANGE ( ',' RANGE )*
// Where:
// RANGE := <NUMBER> | <NUMBER>-<NUMBER>
func NewCPUSetFromExpression(contents string) (CPUSet, error) {
	var ranges [][]int
	var max, count int
	for _, expr := range strings.Split(contents, ",") {
		if len(expr) == 0 {
			continue
		}
		parts := strings.Split(expr, "-")
		r := make([]int, 0, len(parts))
		for _, numStr := range parts {
			num16, err := strconv.ParseInt(numStr, 10, 16)
			if err != nil || num16 < 0 {
				return CPUSet{}, fmt.Errorf("failed to parse integer '%s' from range '%s' at '%s'", numStr, expr, contents)
			}
			num := int(num16)
			r = append(r, num)
			if num+1 > max {
				max = num + 1
			}
		}
		ranges = append(ranges, r)
	}
	if max == 0 {
		return CPUSet{}, nil
	}
	mask := make([]bool, max)
	for _, r := range ranges {
		from, to := -1, -1
		switch len(r) {
		case 0:
			continue // Ignore empty range.
		case 1:
			from = r[0]
			to = r[0]
		case 2:
			from = r[0]
			to = r[1]
		}
		if from == -1 || to < from {
			return CPUSet{}, fmt.Errorf("invalid cpu range %v in '%s'", r, contents)
		}
		for i := from; i <= to; i++ {
			if !mask[i] {
				count++
				mask[i] = true
			}
		}
	}
	return CPUSet{
		mask:  mask,
		count: count,
	}, nil
}
