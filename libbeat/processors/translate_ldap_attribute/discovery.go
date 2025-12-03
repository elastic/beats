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
	"errors"
	"fmt"
	"net"
	"os"
	"runtime"
	"strings"

	"github.com/elastic/elastic-agent-libs/logp"
)

var (
	// errNoLDAPServerFound is returned when no LDAP server can be discovered
	errNoLDAPServerFound = errors.New("no LDAP server found via DNS SRV or system configuration")

	// resolveTCPAddr allows tests to stub DNS resolution
	resolveTCPAddr = net.ResolveTCPAddr
)

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
// Priority order: USERDNSDOMAIN (Windows AD), hostname parsing, reverse DNS lookup.
func discoverDomainName(log *logp.Logger) string {
	// 1. Windows AD domain from USERDNSDOMAIN (only available in interactive Windows sessions)
	domain := os.Getenv("USERDNSDOMAIN")
	if domain != "" {
		log.Infow("discovered domain name from USERDNSDOMAIN environment variable", "domain", domain)
		return strings.ToLower(domain)
	}

	// 2. Try to extract domain from hostname (works on domain-joined Unix systems with FQDN hostnames)
	hostname, err := os.Hostname()
	if err == nil && strings.Contains(hostname, ".") {
		parts := strings.SplitN(hostname, ".", 2)
		if len(parts) == 2 {
			domain = parts[1]
			log.Infow("discovered domain name from hostname", "hostname", hostname, "domain", domain)
			return strings.ToLower(domain)
		}
	}

	// 3. Try reverse DNS lookup on local machine IP to get FQDN
	// This works in service contexts where environment variables are unavailable
	if hostname != "" {
		domain = discoverDomainViaReverseDNS(hostname, log)
		if domain != "" {
			return strings.ToLower(domain)
		}
	}

	log.Debugw("no domain name discovered", "checked", []string{"USERDNSDOMAIN", "hostname", "reverse_dns"})
	return ""
}

// discoverDomainViaReverseDNS performs reverse DNS lookup to discover the domain.
// It resolves the hostname to IP addresses, then does reverse lookup to get FQDN.
func discoverDomainViaReverseDNS(hostname string, log *logp.Logger) string {
	// Resolve hostname to IP addresses
	addrs, err := net.LookupHost(hostname)
	if err != nil {
		log.Debugw("failed to resolve hostname for reverse DNS lookup", "hostname", hostname, "error", err)
		return ""
	}

	// Try reverse DNS lookup on each IP address (prefer IPv4)
	for _, addr := range addrs {
		// Skip link-local IPv6 addresses (contain %)
		if strings.Contains(addr, "%") {
			continue
		}

		names, err := net.LookupAddr(addr)
		if err != nil {
			log.Debugw("reverse DNS lookup failed", "ip", addr, "error", err)
			continue
		}

		// Extract domain from first FQDN found
		for _, name := range names {
			name = strings.TrimSuffix(name, ".")
			if strings.Contains(name, ".") {
				parts := strings.SplitN(name, ".", 2)
				if len(parts) == 2 {
					domain := parts[1]
					log.Infow("discovered domain name via reverse DNS", "fqdn", name, "domain", domain, "ip", addr)
					return domain
				}
			}
		}
	}

	log.Debugw("reverse DNS lookup did not yield domain", "hostname", hostname, "addresses_tried", len(addrs))
	return ""
}

// findServers performs DNS SRV lookup using the standard Go net resolver.
// This automatically handles Windows DNS configuration including search suffixes.
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
	// Always use empty service/proto to make net.LookupSRV do direct name lookup
	// which allows full control over the query name and proper search suffix handling
	var queries []string
	if domain != "" {
		// Pattern 1: Active Directory DC-specific: _ldap._tcp.dc._msdcs.{domain}
		// Pattern 2: Standard domain: _ldap._tcp.{domain}
		queries = []string{
			fmt.Sprintf("_%s._tcp.dc._msdcs.%s", service, domain),
			fmt.Sprintf("_%s._tcp.%s", service, domain),
		}
	} else {
		// Fallback: bare query (net.LookupSRV will apply search suffixes)
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

		// Pass empty service/proto to make LookupSRV do direct lookup of the full name
		_, records, err := net.LookupSRV("", "", query)
		if err == nil && len(records) > 0 {
			log.Infow("DNS SRV lookup succeeded", "query", query, "record_count", len(records))
			netSRVs = records
			successQuery = query
			break
		}
		log.Debugw("DNS SRV lookup failed", "query", query, "error", err)
	}

	if len(netSRVs) == 0 {
		log.Warnw("all DNS SRV lookup attempts failed", "domain", domain, "queries_tried", len(queries))
		return nil
	}

	// net.LookupSRV already sorts by priority and randomizes by weight per RFC 2782
	// so we can use the records directly
	var addresses []string
	for _, addr := range netSRVs {
		// Remove trailing dot if present (FQDN format in DNS often includes it)
		target := strings.TrimSuffix(addr.Target, ".")
		address := fmt.Sprintf("%s://%s:%d", scheme, target, addr.Port)
		addresses = append(addresses, address)
	}

	log.Infow("discovered servers via DNS SRV", "scheme", scheme, "query", successQuery, "count", len(addresses), "addresses", addresses)

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
