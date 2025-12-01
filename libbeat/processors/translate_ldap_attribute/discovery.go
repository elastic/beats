// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

//go:build !requirefips

package translate_ldap_attribute

import (
	"errors"
	"fmt"
	"math/rand"
	"net"
	"os"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
)

var (
	// errNoLDAPServerFound is returned when no LDAP server can be discovered
	errNoLDAPServerFound = errors.New("no LDAP server found via DNS SRV or system configuration")

	// resolveTCPAddr allows tests to stub DNS resolution
	resolveTCPAddr = net.ResolveTCPAddr

	// lookupSRV allows tests to stub DNS SRV lookups
	lookupSRV = net.LookupSRV

	// newSRVRandomizer returns a random source for SRV weighting (overridden in tests)
	newSRVRandomizer = func() intnRandom {
		return rand.New(rand.NewSource(time.Now().UnixNano()))
	}
)

type intnRandom interface {
	Intn(n int) int
}

// discoverLDAPAddress attempts to auto-discover the LDAP server address.
// It returns a list of candidate addresses sorted by preference (LDAPS over LDAP, SRV over LOGONSERVER).
// The caller should attempt to connect to each address in order until one succeeds.
func discoverLDAPAddress(log *logp.Logger) ([]string, error) {
	log.Debugw("attempting LDAP server auto-discovery")

	var candidates []string

	// 1. Primary: DNS SRV Lookup (LDAPS, then LDAP)
	candidates = append(candidates, findServers(true, log)...)
	candidates = append(candidates, findServers(false, log)...)

	// 2. Windows Fallback: LOGONSERVER environment variable
	if runtime.GOOS == "windows" {
		// Log the attempt and try both secure and non-secure
		log.Debug("attempting discovery via LOGONSERVER environment variable")
		candidates = append(candidates, findLogonServer(true, log)...)
		candidates = append(candidates, findLogonServer(false, log)...)
	}

	if len(candidates) == 0 {
		return nil, errNoLDAPServerFound
	}

	return candidates, nil
}

// findServers is a simplified wrapper for DNS SRV lookup.
func findServers(useTLS bool, log *logp.Logger) []string {
	// Note: net.LookupSRV with empty domain "" relies on the system resolver's default domain.
	service := "ldap"
	proto := "tcp"
	scheme := "ldap"
	if useTLS {
		service = "ldaps"
		scheme = "ldaps"
	}

	log.Debugw("looking up DNS SRV record", "service", service, "proto", proto)

	_, addrs, err := lookupSRV(service, proto, "")
	if err != nil {
		log.Debugw("DNS SRV lookup failed", "query", fmt.Sprintf("_%s._%s", service, proto), "error", err)
		return nil
	}

	if len(addrs) == 0 {
		log.Debugw("No SRV records found", "query", fmt.Sprintf("_%s._%s", service, proto))
		return nil
	}

	ordered := orderSRVRecords(addrs, newSRVRandomizer())
	var addresses []string
	for _, addr := range ordered {
		// Remove trailing dot if present (FQDN format in DNS often includes it)
		target := strings.TrimSuffix(addr.Target, ".")
		address := fmt.Sprintf("%s://%s:%d", scheme, target, addr.Port)
		addresses = append(addresses, address)
	}

	log.Infow("discovered servers via DNS SRV", "scheme", scheme, "count", len(addresses), "addresses", addresses)

	return addresses
}

// findLogonServer is a simplified wrapper for LOGONSERVER lookup.
// It attempts to resolve the NetBIOS name to an IP address before returning the address.
func findLogonServer(useTLS bool, log *logp.Logger) []string {
	logonServer := os.Getenv("LOGONSERVER")
	if logonServer == "" {
		log.Debug("LOGONSERVER environment variable not set")
		return nil
	}

	// Remove leading backslashes (Windows format: \\SERVERNAME)
	serverName := strings.TrimPrefix(logonServer, "\\\\")
	serverName = strings.TrimPrefix(serverName, "\\")

	if serverName == "" {
		log.Debugw("invalid LOGONSERVER format", "value", logonServer)
		return nil
	}

	scheme := "ldap"
	port := 389
	if useTLS {
		scheme = "ldaps"
		port = 636
	}

	// Attempt to resolve the NetBIOS name to a resolvable IP/Hostname.
	addressToResolve := net.JoinHostPort(serverName, fmt.Sprintf("%d", port))

	var resolvedIP string
	resolvedAddr, err := resolveTCPAddr("tcp", addressToResolve)
	if err != nil {
		log.Debugw("failed to resolve LOGONSERVER address", "server_name", serverName, "port", port, "error", err)
	} else if resolvedAddr.IP != nil {
		resolvedIP = resolvedAddr.IP.String()
	}

	var addresses []string
	hostURL := fmt.Sprintf("%s://%s:%d", scheme, serverName, port)
	addresses = append(addresses, hostURL)

	if resolvedIP != "" && !strings.EqualFold(resolvedIP, serverName) {
		addresses = append(addresses, fmt.Sprintf("%s://%s:%d", scheme, resolvedIP, port))
	}

	log.Infow("discovered server via LOGONSERVER", "addresses", addresses, "original_name", serverName)

	return addresses
}

// orderSRVRecords sorts SRV answers according to RFC 2782 so we do not
// overload a single controller. The algorithm groups records by priority
// (lower numeric value means a more preferred server) and, within each
// priority, uses selectByWeight to shuffle according to the advertised
// weight field. A deterministic sorter would break load balancing, so
// we optionally accept a custom random source for tests.
func orderSRVRecords(addrs []*net.SRV, r intnRandom) []*net.SRV {
	if len(addrs) == 0 {
		return nil
	}
	if r == nil {
		r = rand.New(rand.NewSource(time.Now().UnixNano()))
	}

	// Build buckets keyed by priority so we can process high priority servers first.
	priorityGroups := make(map[uint16][]*net.SRV)
	for _, addr := range addrs {
		priorityGroups[addr.Priority] = append(priorityGroups[addr.Priority], addr)
	}
	var priorities []uint16
	for priority := range priorityGroups {
		priorities = append(priorities, priority)
	}
	slices.Sort(priorities)

	var ordered []*net.SRV
	for _, priority := range priorities {
		// Within each priority, pick servers according to relative weight.
		group := priorityGroups[priority]
		ordered = append(ordered, selectByWeight(group, r)...)
	}
	return ordered
}

// selectByWeight performs the weighted-shuffle portion of RFC 2782: each
// iteration chooses one server proportionally to its Weight value, removes
// it from the candidate list, and repeats. This ensures a server with a
// higher advertised weight is more likely to be attempted earlier but every
// server eventually appears in the ordered slice. Tests inject a deterministic
// random source so the selection becomes predictable.
func selectByWeight(group []*net.SRV, r intnRandom) []*net.SRV {
	remaining := slices.Clone(group)
	var ordered []*net.SRV
	for len(remaining) > 0 {
		totalWeight := 0
		for _, addr := range remaining {
			totalWeight += int(addr.Weight)
		}

		var idx int
		if totalWeight == 0 {
			idx = r.Intn(len(remaining))
		} else {
			pick := r.Intn(totalWeight)
			sum := 0
			for i, addr := range remaining {
				sum += int(addr.Weight)
				if pick < sum {
					idx = i
					break
				}
			}
		}

		ordered = append(ordered, remaining[idx])
		remaining = slices.Delete(remaining, idx, idx+1)
	}
	return ordered
}
