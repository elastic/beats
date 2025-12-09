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

//go:build !windows && !requirefips

package translate_ldap_attribute

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// discoverDomainInPlatform Chain: Resolv.conf -> Hostname -> Krb5.conf
func discoverDomainInPlatform() (string, error) {
	if d, err := getDomainResolv(); err == nil && d != "" {
		return d, nil
	}
	if d, err := getDomainKrbConf(); err == nil && d != "" {
		return d, nil
	}
	if d, err := getDomainHostname(); err == nil && d != "" {
		return d, nil
	}
	return "", fmt.Errorf("domain discovery failed")
}

func getDomainResolv() (string, error) {
	f, err := os.Open("/etc/resolv.conf")
	if err != nil {
		return "", err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		fields := strings.Fields(s.Text())
		if len(fields) > 1 && (fields[0] == "search" || fields[0] == "domain") {
			return fields[1], nil
		}
	}
	return "", nil
}

func getDomainKrbConf() (string, error) {
	f, err := os.Open("/etc/krb5.conf")
	if err != nil {
		return "", err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if strings.HasPrefix(line, "default_realm") {
			if _, rhs, ok := strings.Cut(line, "="); ok {
				return strings.ToLower(strings.TrimSpace(rhs)), nil
			}
		}
	}
	return "", nil
}
