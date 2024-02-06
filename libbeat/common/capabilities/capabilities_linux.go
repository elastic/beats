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

//go:build linux

package capabilities

import (
	"errors"
	"math/bits"
	"strconv"
	"strings"

	"kernel.org/pub/linux/libs/security/libcap/cap"
)

var (
	// errInvalidCapability expresses an invalid capability ID: x < 0 || x >= 64.
	errInvalidCapability = errors.New("invalid capability")
)

// The capability set flag/vector, re-exported from
// libcap(3). Inherit, Bound & Ambient not exported since we have no
// use for it yet.
type Flag = cap.Flag

const (
	// aka CapEff
	Effective = cap.Effective
	// aka CapPrm
	Permitted = cap.Permitted
)

// Fetch the capabilities of pid for a given flag/vector and convert
// it to the representation used in ECS. cap.GetPID() fetches it with
// SYS_CAPGET.
// Returns errors.ErrUnsupported on "not linux".
func FromPid(flag Flag, pid int) ([]string, error) {
	set, err := cap.GetPID(pid)
	if err != nil {
		return nil, err
	}
	empty, err := isEmpty(flag, set)
	if err != nil {
		return nil, err
	}
	if empty {
		return []string{}, nil
	}

	sl := make([]string, 0, cap.MaxBits())
	for i := 0; i < int(cap.MaxBits()); i++ {
		c := cap.Value(i)
		enabled, err := set.GetFlag(flag, c)
		if err != nil {
			return nil, err
		}
		if !enabled {
			continue
		}
		s, err := toECS(i)
		// impossible since MaxBits <= 64
		if err != nil {
			return nil, err
		}
		sl = append(sl, s)
	}

	return sl, err
}

// Convert a uint64 to the capabilities representation used in ECS.
// Returns errors.ErrUnsupported on "not linux".
func FromUint64(w uint64) ([]string, error) {
	sl := make([]string, 0, bits.OnesCount64(w))
	for i := 0; w != 0; i++ {
		if w&1 != 0 {
			s, err := toECS(i)
			// impossible since MaxBits <= 64
			if err != nil {
				return nil, err
			}
			sl = append(sl, s)
		}
		w >>= 1
	}

	return sl, nil
}

// Convert a string to the capabilities representation used in
// ECS. Example input: "1ffffffffff", 16.
// Returns errors.ErrUnsupported on "not linux".
func FromString(s string, base int) ([]string, error) {
	w, err := strconv.ParseUint(s, 16, 64)
	if err != nil {
		return nil, err
	}

	return FromUint64(w)
}

// True if sets are equal for the given flag/vector, errors out in
// case any of the sets is malformed.
func isEqual(flag Flag, a *cap.Set, b *cap.Set) (bool, error) {
	d, err := a.Cf(b)
	if err != nil {
		return false, err
	}

	return !d.Has(flag), nil
}

// Convert the capability ID to a string suitable to be used in
// ECS.
// If capabiliy ID X is unknown, but valid (0 <= X < 64), "CAP_X"
// will be returned instead. Fetches from an internal table built at
// startup.
var toECS = makeToECS()

// Make toECS() which creates a map of every possible valid capability
// ID on startup. Returns errInvalidCapabilty for an invalid ID.
func makeToECS() func(int) (string, error) {
	ecsNames := make(map[int]string)

	for i := 0; i < 64; i++ {
		c := cap.Value(i)
		if i < int(cap.MaxBits()) {
			ecsNames[i] = strings.ToUpper(c.String())
		} else {
			ecsNames[i] = strings.ToUpper("CAP_" + c.String())
		}
	}

	return func(b int) (string, error) {
		s, ok := ecsNames[b]
		if !ok {
			return "", errInvalidCapability
		}
		return s, nil
	}
}

// Like isAll(), but for the empty set, here for symmetry.
func isEmpty(flag Flag, set *cap.Set) (bool, error) {
	return isEqual(flag, set, cap.NewSet())
}
