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

//nolint:errorlint // Bad linter!
package route

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/sys/windows"
)

// defaultRoute returns the interface name and netlink device index for the
// default route obtained from netsh and getmac.
func defaultRoute(af int) (name string, index int, err error) {
	index = -1
	var family string
	switch af {
	default:
		return "", -1, fmt.Errorf("unknown family: %d", af)
	case windows.AF_INET:
		family = "ipv4"
	case windows.AF_INET6:
		family = "ipv6"
	}

	r, err := run("netsh", "interface", family, "show", "route")
	if err != nil {
		return "", -1, err
	}
	sc := bufio.NewScanner(bytes.NewReader(r))
	for inTable := false; sc.Scan(); {
		f := strings.Fields(sc.Text())
		if len(f) == 0 {
			if inTable {
				break
			}
			continue
		}
		if !inTable {
			inTable = f[0] == "-------"
			continue
		}
		if len(f) < 5 {
			return "", -1, fmt.Errorf("unexpected netsh %s line: %q\n\n%s", name, sc.Text(), r)
		}
		if strings.Contains(f[3], "/") {
			ip, _, err := net.ParseCIDR(f[3])
			if err != nil || !ip.IsUnspecified() {
				continue
			}
		} else {
			ip := net.ParseIP(f[3])
			if ip == nil || !ip.IsUnspecified() {
				continue
			}
		}
		idx, err := strconv.Atoi(f[4])
		if err != nil {
			return "", -1, fmt.Errorf("failed to parse index from %s: %v", f[4], err)
		}
		index = idx
		break
	}
	err = sc.Err()
	if err != nil {
		return "", -1, err
	}

	d, err := run("netsh", "interface", family, "show", "interfaces")
	if err != nil {
		return "", -1, err
	}
	sc = bufio.NewScanner(bytes.NewReader(d))
	for inTable := false; sc.Scan(); {
		f := fieldsN(sc.Text(), 5)
		if len(f) == 0 {
			if inTable {
				break
			}
			continue
		}
		if !inTable {
			inTable = f[0] == "---"
			continue
		}
		if len(f) < 5 {
			return "", -1, fmt.Errorf("unexpected netsh %s line: %q\n\n%s", name, sc.Text(), d)
		}
		idx, err := strconv.Atoi(f[0])
		if err != nil {
			return "", -1, fmt.Errorf("failed to parse index from %s: %v", f[0], err)
		}
		if idx == index {
			name = f[4]
			break
		}
	}
	err = sc.Err()
	if err != nil {
		return "", -1, err
	}

	m, err := run("getmac", "/fo", "csv", "/v", "/nh")
	if err != nil {
		return "", -1, err
	}
	cr := csv.NewReader(bytes.NewReader(m))
	cr.FieldsPerRecord = 4
	for {
		row, err := cr.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", -1, err
		}
		if row[0] == name {
			name = row[3]
			break
		}
	}

	if index == -1 {
		err = ErrNotFound
	}
	return name, index, err
}

// fieldsN is an equivalent of strings.Fields that returns only n fields.
// If n < 1, s is returned with space trimmed as the only element.
func fieldsN(s string, n int) []string {
	s = strings.TrimSpace(s)
	if n < 1 {
		return []string{s}
	}
	var f []string
	for s != "" {
		l := len(s)
		for i, r := range s {
			if unicode.IsSpace(r) {
				f = append(f, s[:i])
				s = s[i:]
				break
			}
		}
		for i, r := range s {
			if !unicode.IsSpace(r) {
				s = s[i:]
				break
			}
		}
		if len(f) == n-1 || len(s) == l {
			break
		}
	}
	if s != "" {
		f = append(f, s)
	}
	return f
}
