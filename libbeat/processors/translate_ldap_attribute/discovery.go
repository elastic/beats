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

//go:build !requirefips

package translate_ldap_attribute

import (
	"context"
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
	"github.com/miekg/dns"
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
func discoverLDAPAddress(configDomain string, log *logp.Logger) ([]string, error) {
	log.Debug("attempting LDAP server auto-discovery")

	var candidates []string

	// 1. Primary: DNS SRV Lookup (LDAPS, then LDAP)
	candidates = append(candidates, findServers(configDomain, true, log)...)
	candidates = append(candidates, findServers(configDomain, false, log)...)

	// 2. Windows Fallback: LOGONSERVER environment variable
	log.Debug("attempting discovery via LOGONSERVER environment variable")
	candidates = append(candidates, findLogonServer(true, log)...)
	candidates = append(candidates, findLogonServer(false, log)...)

	if len(candidates) == 0 {
		log.Warnw("no LDAP servers discovered", "dns_srv_attempted", true, "logonserver_attempted", runtime.GOOS == "windows")
		return nil, errNoLDAPServerFound
	}

	log.Infow("LDAP server auto-discovery completed", "total_candidates", len(candidates), "candidates", candidates)
	return candidates, nil
}

// discoverDomainName attempts to discover the DNS domain name from various sources.
// Priority order: USERDNSDOMAIN (Windows AD), hostname parsing.
func discoverDomainName(log *logp.Logger) string {
	// 1. Windows AD domain from USERDNSDOMAIN
	domain := os.Getenv("USERDNSDOMAIN")
	if domain != "" {
		log.Infow("discovered domain name from USERDNSDOMAIN environment variable", "domain", domain)
		return strings.ToLower(domain)
	}

	// 2. Try to extract domain from hostname (works on domain-joined Unix systems)
	hostname, err := os.Hostname()
	if err == nil && strings.Contains(hostname, ".") {
		parts := strings.SplitN(hostname, ".", 2)
		if len(parts) == 2 {
			domain = parts[1]
			log.Infow("discovered domain name from hostname", "hostname", hostname, "domain", domain)
			return strings.ToLower(domain)
		}
	}

	log.Debugw("no domain name discovered", "checked", []string{"USERDNSDOMAIN", "hostname"})
	return ""
}

// findServers performs DNS SRV lookup using miekg/dns for better control.
func findServers(configDomain string, useTLS bool, log *logp.Logger) []string {
	// Use configured domain if provided, otherwise attempt auto-discovery
	var domain string
	if configDomain != "" {
		log.Infow("using configured domain for DNS SRV lookup", "domain", configDomain)
		domain = strings.ToLower(configDomain)
	} else {
		domain = discoverDomainName(log)
	}

	service := "ldap"
	scheme := "ldap"
	if useTLS {
		service = "ldaps"
		scheme = "ldaps"
	}

	// Build query patterns to try
	var queries []string
	if domain != "" {
		// Pattern 1: Active Directory DC-specific: _ldap._tcp.dc._msdcs.{domain}
		// Pattern 2: Standard domain: _ldap._tcp.{domain}
		queries = []string{
			fmt.Sprintf("_%s._tcp.dc._msdcs.%s.", service, domain),
			fmt.Sprintf("_%s._tcp.%s.", service, domain),
		}
	} else {
		// Fallback: bare query (let DNS resolver apply search suffixes)
		queries = []string{
			fmt.Sprintf("_%s._tcp.dc._msdcs", service),
			fmt.Sprintf("_%s._tcp", service),
		}
		log.Infow("no domain available, attempting bare DNS SRV lookup with search suffix")
	}

	var netSRVs []*net.SRV
	var successQuery string

	for _, query := range queries {
		log.Infow("executing DNS SRV lookup", "query", query, "service", service)

		records, err := lookupSRVWithMiekgDNS(query, log)
		if err == nil {
			log.Infow("DNS SRV lookup succeeded", "query", query, "record_count", len(records))
			for _, srv := range records {
				netSRVs = append(netSRVs, &net.SRV{
					Target:   srv.Target,
					Port:     srv.Port,
					Priority: srv.Priority,
					Weight:   srv.Weight,
				})
			}
			successQuery = query
			break
		}
		log.Debugw("DNS SRV lookup failed", "query", query, "error", err)
	}

	if len(netSRVs) == 0 {
		log.Warnw("all DNS SRV lookup attempts failed", "domain", domain, "queries_tried", queries)
		return nil
	}

	ordered := orderSRVRecords(netSRVs, newSRVRandomizer())
	var addresses []string
	for _, addr := range ordered {
		// Remove trailing dot if present (FQDN format in DNS often includes it)
		target := strings.TrimSuffix(addr.Target, ".")
		address := fmt.Sprintf("%s://%s:%d", scheme, target, addr.Port)
		addresses = append(addresses, address)
	}

	log.Infow("discovered servers via DNS SRV", "scheme", scheme, "query", successQuery, "count", len(addresses), "addresses", addresses)

	return addresses
}

// lookupSRVWithMiekgDNS performs DNS SRV lookup using miekg/dns client.
// This gives us more control over the query and allows bare queries that use DNS search suffixes.
func lookupSRVWithMiekgDNS(query string, log *logp.Logger) ([]*dns.SRV, error) {
	// Get system DNS config
	dnsConfig, err := dns.ClientConfigFromFile("/etc/resolv.conf")
	if err != nil {
		// On Windows or if file doesn't exist, use system defaults
		dnsConfig = &dns.ClientConfig{
			Servers: []string{"127.0.0.1"},
			Port:    "53",
			Timeout: 5,
		}
	}

	client := &dns.Client{
		Timeout: time.Duration(dnsConfig.Timeout) * time.Second,
	}

	msg := &dns.Msg{}
	msg.SetQuestion(query, dns.TypeSRV)
	msg.RecursionDesired = true

	// Try each DNS server
	var lastErr error
	for _, server := range dnsConfig.Servers {
		target := net.JoinHostPort(server, dnsConfig.Port)
		log.Debugw("querying DNS server", "server", target, "query", query)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		response, _, err := client.ExchangeContext(ctx, msg, target)
		cancel()

		if err != nil {
			log.Debugw("DNS query failed", "server", target, "error", err)
			lastErr = err
			continue
		}

		if response.Rcode != dns.RcodeSuccess {
			log.Debugw("DNS query returned error code", "server", target, "rcode", dns.RcodeToString[response.Rcode])
			lastErr = fmt.Errorf("DNS query failed with rcode: %s", dns.RcodeToString[response.Rcode])
			continue
		}

		// Extract SRV records from answer section
		var srvRecords []*dns.SRV
		for _, answer := range response.Answer {
			if srv, ok := answer.(*dns.SRV); ok {
				srvRecords = append(srvRecords, srv)
			}
		}

		if len(srvRecords) > 0 {
			log.Debugw("DNS server returned SRV records", "server", target, "count", len(srvRecords))
			return srvRecords, nil
		}

		log.Debugw("DNS server returned no SRV records", "server", target)
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("no SRV records found")
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
