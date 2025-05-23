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

//go:build darwin || dragonfly || freebsd || netbsd || openbsd

//nolint:errorlint // Bad linter! All the advice given by this linter in this file harms readability.
package route

import (
	"bufio"
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

// defaultRoute returns the interface name and netlink device index for the
// default route obtained from netstat.
func defaultRoute(af int) (name string, index int, err error) {
	index = -1
	var family string
	switch af {
	default:
		return "", -1, fmt.Errorf("unknown family: %d", af)
	case unix.AF_INET:
		family = "inet"
	case unix.AF_INET6:
		family = "inet6"
	}

	r, err := run("netstat", "-rnf", family)
	if err != nil {
		return "", -1, err
	}
	sc := bufio.NewScanner(bytes.NewReader(r))
	for inTable := false; sc.Scan(); {
		f := strings.Fields(sc.Text())
		if len(f) == 0 {
			continue
		}
		if !inTable {
			inTable = f[0] == "Destination"
			continue
		}
		if f[0] == "default" {
			name = f[3]
			break
		}
	}
	err = sc.Err()
	if err != nil {
		return "", -1, err
	}

	d, err := run("netstat", "-I", name)
	if err != nil {
		return "", -1, err
	}
	sc = bufio.NewScanner(bytes.NewReader(d))
	sc.Scan() // Drop header.
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) < 3 {
			return "", -1, fmt.Errorf("unexpected netstat -I %s line: %q", name, sc.Text())
		}
		if strings.HasPrefix(f[2], "<Link#") {
			idx, err := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(f[2], "<Link#"), ">"))
			if err != nil {
				return "", -1, fmt.Errorf("failed to parse index from %s: %v", f[2], err)
			}
			index = idx
			break
		}
	}
	err = sc.Err()
	if err != nil {
		return "", -1, err
	}

	if index == -1 {
		err = ErrNotFound
	}
	return name, index, err
}
