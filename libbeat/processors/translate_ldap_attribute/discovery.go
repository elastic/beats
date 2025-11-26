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
	"net"
	"os"
	"runtime"
	"strings"

	"github.com/elastic/elastic-agent-libs/logp"
)

var (
	// ErrNoLDAPServerFound is returned when no LDAP server can be discovered
	ErrNoLDAPServerFound = errors.New("no LDAP server found via DNS SRV or system configuration")
)

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
		return nil, ErrNoLDAPServerFound
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

	_, addrs, err := net.LookupSRV(service, proto, "")
	if err != nil {
		log.Debugw("DNS SRV lookup failed", "query", fmt.Sprintf("_%s._%s", service, proto), "error", err)
		return nil
	}

	if len(addrs) == 0 {
		log.Debugw("No SRV records found", "query", fmt.Sprintf("_%s._%s", service, proto))
		return nil
	}

	var addresses []string
	for _, addr := range addrs {
		// Remove trailing dot if present (FQDN format in DNS often includes it)
		target := strings.TrimSuffix(addr.Target, ".")
		address := fmt.Sprintf("%s://%s:%d", scheme, target, addr.Port)
		addresses = append(addresses, address)
	}

	log.Infow("discovered servers via DNS SRV", "scheme", scheme, "count", len(addresses), "addresses", addresses)

	return addresses
}

// findLogonServer is a simplified wrapper for LOGONSERVER lookup.
// It resolves the NetBIOS name to an IP or FQDN before returning the address.
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

	// Use net.ResolveTCPAddr to resolve the NetBIOS name to a specific IP address.
	resolvedAddr, err := net.ResolveTCPAddr("tcp", addressToResolve)
	if err != nil {
		log.Debugw("failed to resolve LOGONSERVER address", "server_name", serverName, "port", port, "error", err)
		return nil
	}

	// The primary result we want is the IP address, which is reliable.
	resolvedHost := resolvedAddr.IP.String()

	// If the IP is unavailable or invalid, fall back to the original NetBIOS name.
	// This is a calculated risk, but better than returning an empty string.
	if resolvedHost == "" || resolvedHost == "<nil>" {
		resolvedHost = serverName
		log.Debug("Resolved IP was invalid or empty, falling back to original server name.")
	}

	// Construct the final LDAP URL using the resolved IP address and the specific port/scheme
	ldapAddress := fmt.Sprintf("%s://%s:%d", scheme, resolvedHost, port)

	log.Infow("discovered server via LOGONSERVER", "address", ldapAddress, "original_name", serverName)

	return []string{ldapAddress}
}
