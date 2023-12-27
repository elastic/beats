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

package route

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

// defaultRoute returns the interface name and netlink device index for the
// default route obtained from the /proc/net filesystem.
func defaultRoute(af int) (name string, index int, err error) {
	const gateway = unix.RTF_UP | unix.RTF_GATEWAY

	index = -1
	switch af {
	default:
		return "", -1, fmt.Errorf("unknown family: %d", af)
	case unix.AF_INET:
		r, err := os.ReadFile("/proc/net/route")
		if err != nil {
			return "", -1, err
		}
		sc := bufio.NewScanner(bytes.NewReader(r))
		sc.Scan() // Drop header.
		for sc.Scan() {
			f := strings.Fields(sc.Text())
			if len(f) != 11 {
				return "", -1, fmt.Errorf("unexpected /proc/net/route line: %q", sc.Text())
			}
			flags, err := strconv.ParseInt(f[3], 16, 64)
			if err != nil {
				return "", -1, err
			}
			if flags&gateway != gateway {
				continue
			}
			name = f[0]
		}
		err = sc.Err()
		if err != nil {
			return "", -1, err
		}

		d, err := os.ReadFile("/proc/net/dev_mcast")
		if err != nil {
			return "", -1, err
		}
		sc = bufio.NewScanner(bytes.NewReader(d))
		for sc.Scan() {
			f := strings.Fields(sc.Text())
			if len(f) != 5 {
				return "", -1, fmt.Errorf("unexpected /proc/net/dev_mcast line: %q", sc.Text())
			}
			if f[1] != name {
				continue
			}
			idx, err := strconv.Atoi(f[0])
			if err != nil {
				return "", -1, err
			}
			index = idx
			break
		}
		err = sc.Err()
		if err != nil {
			return "", -1, err
		}
	case unix.AF_INET6:
		r, err := os.ReadFile("/proc/net/ipv6_route")
		if err != nil {
			return "", -1, err
		}
		sc := bufio.NewScanner(bytes.NewReader(r))
		for sc.Scan() {
			f := strings.Fields(sc.Text())
			if len(f) != 10 {
				return "", -1, fmt.Errorf("unexpected /proc/net/ipv6_route line: %q", sc.Text())
			}
			flags, err := strconv.ParseInt(f[8], 16, 64)
			if err != nil {
				return "", -1, err
			}
			if flags&gateway != gateway {
				continue
			}
			name = f[9]
		}
		err = sc.Err()
		if err != nil {
			return "", -1, err
		}

		d, err := os.ReadFile("/proc/net/if_inet6")
		if err != nil {
			return "", -1, err
		}
		sc = bufio.NewScanner(bytes.NewReader(d))
		for sc.Scan() {
			f := strings.Fields(sc.Text())
			if len(f) != 6 {
				return "", -1, fmt.Errorf("unexpected /proc/net/if_inet6 line: %q", sc.Text())
			}
			if f[5] != name {
				continue
			}
			idx, err := strconv.ParseInt(f[1], 16, 8)
			if err != nil {
				return "", -1, err
			}
			index = int(idx)
			break
		}
		err = sc.Err()
		if err != nil {
			return "", -1, err
		}
	}
	if index == -1 {
		err = ErrNotFound
	}
	return name, index, err
}
