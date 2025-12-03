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

	domain := normalizeDomain(configDomain)
	if domain == "" {
		domain = discoverDomainName(log)
	}

	var candidates []string

	// 1. Primary: DNS SRV Lookup (LDAPS, then LDAP)
	candidates = append(candidates, lookupSRVServers(domain, true, log)...)
	candidates = append(candidates, lookupSRVServers(domain, false, log)...)

	if len(candidates) > 0 {
		return candidates, nil
	}

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

type domainSource struct {
	name   string
	getter func() string
}

// discoverDomainName attempts to discover the DNS domain name from various sources in priority order.
func discoverDomainName(log *logp.Logger) string {

	sources := []domainSource{
		{
			name:   "USERDNSDOMAIN",
			getter: func() string { return os.Getenv("USERDNSDOMAIN") },
		},
	}

	if h, err := os.Hostname(); err == nil && h != "" {
		sources = append(sources,
			domainSource{
				name: "hostname",
				getter: func() string {
					if !strings.Contains(h, ".") {
						return ""
					}
					parts := strings.SplitN(h, ".", 2)
					if len(parts) == 2 {
						return parts[1]
					}
					return ""
				},
			},
			domainSource{
				name:   "reverse_dns",
				getter: func() string { return discoverDomainViaReverseDNS(h, log) },
			},
		)
	} else {
		log.Debugw("failed to read hostname", "error", err)
	}

	if runtime.GOOS == "windows" {
		sources = append(sources, domainSource{
			name:   "windows_registry",
			getter: func() string { return discoverDomainFromRegistry(log) },
		})
	}

	triedSources := make([]string, 0, len(sources))
	for _, source := range sources {
		triedSources = append(triedSources, source.name)
		domain := normalizeDomain(source.getter())
		if domain != "" {
			log.Infow("discovered domain name", "source", source.name, "domain", domain)
			return domain
		}
	}

	log.Debugw("no domain name discovered", "sources_tried", triedSources)
	return ""
}

// normalizeDomain trims and lower-cases a domain value.
func normalizeDomain(domain string) string {
	return strings.ToLower(strings.TrimSpace(domain))
}

// discoverDomainViaReverseDNS performs reverse DNS lookup to discover the domain.
// It resolves the hostname to IP addresses, then does reverse lookup to get FQDN.
func discoverDomainViaReverseDNS(hostname string, log *logp.Logger) string {
	addrs, err := net.LookupHost(hostname)
	if err != nil {
		log.Debugw("failed to resolve hostname for reverse DNS lookup", "hostname", hostname, "error", err)
		return ""
	}

	for _, addr := range addrs {
		if strings.Contains(addr, "%") {
			continue
		}

		names, err := net.LookupAddr(addr)
		if err != nil {
			log.Debugw("reverse DNS lookup failed", "ip", addr, "error", err)
			continue
		}

		for _, name := range names {
			name = strings.TrimSuffix(name, ".")
			if !strings.Contains(name, ".") {
				continue
			}
			parts := strings.SplitN(name, ".", 2)
			if len(parts) != 2 {
				continue
			}
			log.Infow("discovered domain name via reverse DNS", "fqdn", name, "domain", parts[1], "ip", addr)
			return parts[1]
		}
	}

	log.Debugw("reverse DNS lookup did not yield domain", "hostname", hostname, "addresses_tried", len(addrs))
	return ""
}

// lookupSRVServers performs DNS SRV lookups for the provided domain using the Go resolver.
func lookupSRVServers(domain string, useTLS bool, log *logp.Logger) []string {
	service := "ldap"
	scheme := "ldap"
	if useTLS {
		service = "ldaps"
		scheme = "ldaps"
	}

	queries := buildSRVQueries(service, domain, log)
	var netSRVs []*net.SRV
	var successQuery string

	for _, query := range queries {
		log.Infow("executing DNS SRV lookup", "query", query, "service", service)
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

	var addresses []string
	for _, addr := range netSRVs {
		target := strings.TrimSuffix(addr.Target, ".")
		addresses = append(addresses, fmt.Sprintf("%s://%s:%d", scheme, target, addr.Port))
	}

	log.Infow("discovered servers via DNS SRV", "scheme", scheme, "query", successQuery, "count", len(addresses), "addresses", addresses)
	return addresses
}

func buildSRVQueries(service, domain string, log *logp.Logger) []string {
	if domain != "" {
		domain = fmt.Sprintf(".%s", domain)
	}
	return []string{
		fmt.Sprintf("_%s._tcp.dc._msdcs%s", service, domain),
		fmt.Sprintf("_%s._tcp%s", service, domain),
	}
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
