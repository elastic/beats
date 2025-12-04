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
	"sort"
	"strings"

	"github.com/elastic/elastic-agent-libs/logp"
)

var (
	// errNoLDAPServerFound is returned when no LDAP server can be discovered
	errNoLDAPServerFound = errors.New("no LDAP server found via DNS SRV or system configuration")
)

// discoverLDAPAddress attempts to auto-discover the LDAP server address.
// It returns a list of candidate addresses sorted by preference (LDAPS over LDAP, SRV over LOGONSERVER).
// The caller should attempt to connect to each address in order until one succeeds.
func discoverLDAPAddress(configDomain string, log *logp.Logger) ([]string, error) {
	log.Debug("attempting LDAP server auto-discovery")

	domain := discoverDomain(configDomain, log)

	var candidates []string

	if domain != "" {
		// 1. Primary: DNS SRV Lookup (LDAPS, then LDAP)
		candidates = append(candidates, lookupSRVServers(domain, true, log)...)
		candidates = append(candidates, lookupSRVServers(domain, false, log)...)
	}

	if len(candidates) > 0 {
		return candidates, nil
	}

	// 2. Fallback: LOGONSERVER environment variable,
	// typically only available on Windows interactive sessions
	log.Debug("attempting discovery via LOGONSERVER environment variable")
	candidates = append(candidates, findLogonServer(domain, true, log)...)
	candidates = append(candidates, findLogonServer(domain, false, log)...)

	if len(candidates) == 0 {
		log.Warnw("no LDAP servers discovered", "dns_srv_attempted", true, "logonserver_attempted", runtime.GOOS == "windows")
		return nil, errNoLDAPServerFound
	}

	log.Infow("LDAP server auto-discovery completed", "total_candidates", len(candidates), "candidates", candidates)
	return candidates, nil
}

func discoverDomain(configDomain string, log *logp.Logger) string {
	if configDomain != "" {
		return normalizeDomain(configDomain)
	}
	d, err := discoverDomainInPlatform()
	if err != nil {
		log.Warnw("failed to discover domain in platform", "error", err)
		return ""
	}
	log.Infow("discovered domain in platform", "domain", d)
	return normalizeDomain(d)
}

func normalizeDomain(domain string) string {
	return strings.ToLower(strings.TrimSpace(domain))
}

func getDomainHostname() (string, error) {
	h, err := os.Hostname()
	if err != nil {
		return "", err
	}
	parts := strings.Split(h, ".")
	if len(parts) > 1 {
		return strings.Join(parts[1:], "."), nil
	}
	return "", fmt.Errorf("not FQDN")
}

func lookupSRVServers(domain string, useTLS bool, log *logp.Logger) []string {
	service := "ldap"
	scheme := "ldap"
	if useTLS {
		service = "ldaps"
		scheme = "ldaps"
	}

	log.Infow("executing DNS SRV lookup", "domain", domain, "service", service)
	_, records, err := net.LookupSRV(service, "tcp", domain)
	if err != nil || len(records) == 0 {
		log.Debugw("DNS SRV lookup failed", "domain", domain, "error", err)
		return nil
	}
	log.Infow("DNS SRV lookup succeeded", "domain", domain, "record_count", len(records))

	// Even if the DNS server *usually* sorts them, we enforce it here
	// to ensure we don't accidentally hit a DR site first.
	sort.Slice(records, func(i, j int) bool {
		// 1. Lower Priority is better (RFC 2782)
		if records[i].Priority != records[j].Priority {
			return records[i].Priority < records[j].Priority
		}
		// 2. Higher Weight is better (RFC 2782)
		return records[i].Weight > records[j].Weight
	})

	var addresses []string
	for _, addr := range records {
		target := strings.TrimSuffix(addr.Target, ".")
		addresses = append(addresses, fmt.Sprintf("%s://%s:%d", scheme, target, addr.Port))
	}
	log.Infow("discovered servers via DNS SRV", "scheme", scheme, "domain", domain, "count", len(addresses), "addresses", addresses)
	return addresses
}

// findLogonServer attempts to construct a valid FQDN from the LOGONSERVER env var.
// It requires the previously discovered domain to ensure TLS validation works.
func findLogonServer(domain string, useTLS bool, log *logp.Logger) []string {
	logonServer := os.Getenv("LOGONSERVER")
	if logonServer == "" {
		log.Debug("LOGONSERVER environment variable not set")
		return nil
	}

	// 1. Sanitize: Remove leading backslashes (Windows format: \\SERVERNAME)
	serverName := strings.TrimPrefix(logonServer, `\\`)
	if serverName == "" {
		return nil
	}

	scheme := "ldap"
	port := 389
	if useTLS {
		scheme = "ldaps"
		port = 636
	}

	var addresses []string

	// 2. Option A: The FQDN (Best for TLS)
	// If we have a domain, and the serverName isn't already fully qualified, join them.
	if domain != "" && !strings.Contains(serverName, ".") {
		fqdn := fmt.Sprintf("%s.%s", serverName, domain)
		log.Debugw("constructed FQDN from LOGONSERVER", "original", serverName, "fqdn", fqdn)
		// Return FQDN first - this has the highest chance of passing TLS checks
		addresses = append(addresses, fmt.Sprintf("%s://%s:%d", scheme, fqdn, port))
	}

	// 3. Option B: The NetBIOS Name (Fallback)
	// We add this just in case the FQDN construction was wrong,
	// though this will likely fail TLS validation unless InsecureSkipVerify is used.
	addresses = append(addresses, fmt.Sprintf("%s://%s:%d", scheme, serverName, port))

	log.Infow("discovered server via LOGONSERVER", "addresses", addresses)
	return addresses
}
