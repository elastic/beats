// Copyright 2018 Elasticsearch Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package linux

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/elastic/go-sysinfo/types"
)

type SeccompMode uint8

const (
	SeccompModeDisabled SeccompMode = iota
	SeccompModeStrict
	SeccompModeFilter
)

func (m SeccompMode) String() string {
	switch m {
	case SeccompModeDisabled:
		return "disabled"
	case SeccompModeStrict:
		return "strict"
	case SeccompModeFilter:
		return "filter"
	default:
		return strconv.Itoa(int(m))
	}
}

func Seccomp() (*types.SeccompInfo, error) {
	mode, err := seccompMode()
	if err != nil {
		return nil, err
	}

	info := &types.SeccompInfo{
		Mode: mode.String(),
	}

	noNewPrivs, err := noNewPrivs()
	if err == nil {
		info.NoNewPrivs = &noNewPrivs
	}

	return info, nil
}

func seccompMode() (SeccompMode, error) {
	v, err := findValue(statusFile(), ":", "Seccomp")
	if err != nil {
		return 0, err
	}

	mode, err := strconv.ParseUint(v, 10, 8)
	if err != nil {
		return 0, err
	}

	return SeccompMode(mode), nil
}

func noNewPrivs() (bool, error) {
	v, err := findValue(statusFile(), ":", "NoNewPrivs")
	if err != nil {
		return false, err
	}

	return strconv.ParseBool(v)
}

func statusFile() string {
	return filepath.Join("/proc", strconv.Itoa(os.Getpid()), "status")
}
