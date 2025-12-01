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
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/go-ldap/ldap/v3"

	"github.com/elastic/elastic-agent-libs/logp"
)

// ldapClient manages a single reusable LDAP connection
type ldapClient struct {
	*ldapConfig

	mu   sync.Mutex
	conn *ldap.Conn

	// Server metadata
	isActiveDirectory bool

	log *logp.Logger
}

type ldapConfig struct {
	address         string
	baseDN          string
	username        string
	password        string
	searchAttr      string
	mappedAttr      string
	searchTimeLimit int
	tlsConfig       *tls.Config
}

// newLDAPClient initializes a new ldapClient with a single connection.
// If baseDN is empty, it will attempt to discover it via rootDSE or domain inference.
// It also detects whether the server is Active Directory.
func newLDAPClient(config *ldapConfig, log *logp.Logger) (*ldapClient, error) {
	client := &ldapClient{ldapConfig: config, log: log}

	// Establish initial connection
	conn, err := client.dial()
	if err != nil {
		return nil, err
	}
	client.conn = conn

	// Discover base DN if not provided and detect AD
	if err := client.initializeMetadata(); err != nil {
		client.close()
		return nil, fmt.Errorf("failed to initialize server metadata: %w", err)
	}

	return client, nil
}

// initializeMetadata discovers base DN (if needed) and detects server type

func (client *ldapClient) initializeMetadata() error {
	client.log.Debug("querying rootDSE for server metadata")

	// Query rootDSE with relevant attributes
	searchRequest := ldap.NewSearchRequest(
		"", // Empty base DN = rootDSE
		ldap.ScopeBaseObject,
		ldap.NeverDerefAliases,
		0, // No size limit
		client.searchTimeLimit,
		false,
		"(objectClass=*)", // Match everything
		[]string{
			"defaultNamingContext",
			"namingContexts",
			"rootDomainNamingContext",    // AD-specific
			"configurationNamingContext", // AD-specific
			"schemaNamingContext",        // AD-specific
			"vendorName",
			"vendorVersion",
		},
		nil,
	)

	var result *ldap.SearchResult
	err := client.withLockedConnection(func(conn *ldap.Conn) error {
		var searchErr error
		result, searchErr = conn.Search(searchRequest)
		return searchErr
	})
	if err != nil {
		client.log.Debugw("rootDSE query failed", "error", err)
		// If baseDN is already set, treat rootDSE failure as non-fatal
		if client.baseDN != "" {
			return nil
		}
		return client.inferBaseDNFromAddress()
	}

	if len(result.Entries) == 0 {
		client.log.Debug("rootDSE query returned no entries")
		if client.baseDN != "" {
			return nil
		}
		return client.inferBaseDNFromAddress()
	}

	entry := result.Entries[0]

	// Detect Active Directory
	// AD has rootDomainNamingContext, configurationNamingContext, and defaultNamingContext
	hasRootDomain := len(entry.GetAttributeValues("rootDomainNamingContext")) > 0
	hasConfigContext := len(entry.GetAttributeValues("configurationNamingContext")) > 0
	hasDefaultContext := len(entry.GetAttributeValues("defaultNamingContext")) > 0

	client.isActiveDirectory = hasRootDomain && hasConfigContext && hasDefaultContext

	if client.isActiveDirectory {
		client.log.Info("detected Active Directory server")
	} else {
		vendorName := ""
		if values := entry.GetAttributeValues("vendorName"); len(values) > 0 {
			vendorName = values[0]
		}
		client.log.Infow("detected LDAP server", "vendor", vendorName)
	}

	if client.baseDN != "" {
		return nil
	}

	// Prefer defaultNamingContext (Active Directory)
	if values := entry.GetAttributeValues("defaultNamingContext"); len(values) > 0 {
		client.baseDN = values[0]
		client.log.Infow("discovered base DN via defaultNamingContext", "base_dn", client.baseDN)
		return nil
	}

	// Fallback to first namingContext
	if values := entry.GetAttributeValues("namingContexts"); len(values) > 0 {
		client.baseDN = values[0]
		client.log.Infow("discovered base DN via namingContexts", "base_dn", client.baseDN)
		return nil
	}

	return client.inferBaseDNFromAddress()
}

// inferBaseDNFromAddress infers base DN from the server address
func (client *ldapClient) inferBaseDNFromAddress() error {
	client.log.Debugw("attempting to infer base DN from server address", "address", client.address)

	addr := client.address
	lowerAddr := strings.ToLower(addr)
	switch {
	case strings.HasPrefix(lowerAddr, "ldap://"):
		addr = addr[len("ldap://"):]
	case strings.HasPrefix(lowerAddr, "ldaps://"):
		addr = addr[len("ldaps://"):]
	}

	var hostname string
	h, _, err := net.SplitHostPort(addr)
	if err != nil {
		hostname = addr
		if strings.Contains(hostname, ":") {
			return fmt.Errorf("unable to parse hostname from address: %s", client.address)
		}
	} else {
		hostname = h
	}

	hostname = strings.TrimSuffix(hostname, ".")
	hostname = strings.ToLower(hostname)

	if hostname == "" {
		return fmt.Errorf("unable to extract hostname from address: %s", client.address)
	}

	if net.ParseIP(hostname) != nil {
		return fmt.Errorf("cannot infer base DN from IP address: %s", hostname)
	}

	parts := strings.Split(hostname, ".")
	if len(parts) < 2 {
		return fmt.Errorf("hostname does not contain a domain: %s", hostname)
	}

	var candidates [][]string
	seen := make(map[string]struct{})
	addCandidate := func(parts []string) {
		if len(parts) < 2 {
			return
		}
		domain := strings.Join(parts, ".")
		if _, ok := seen[domain]; ok {
			return
		}
		seen[domain] = struct{}{}
		candidates = append(candidates, append([]string(nil), parts...))
	}

	for i := 1; i < len(parts); i++ {
		addCandidate(parts[i:])
	}
	addCandidate(parts[len(parts)-2:])

	if len(candidates) == 0 {
		addCandidate(parts)
	}

	validate := client.conn != nil
	var fallbackDN string
	for _, candidate := range candidates {
		dn := domainPartsToDN(candidate)
		if fallbackDN == "" {
			fallbackDN = dn
		}
		if !validate {
			continue
		}
		if err := client.validateBaseDN(dn); err != nil {
			client.log.Debugw("base DN candidate validation failed", "candidate", dn, "error", err)
			continue
		}
		client.baseDN = dn
		client.log.Infow("inferred base DN from server domain", "base_dn", client.baseDN, "validated", true)
		return nil
	}

	if fallbackDN != "" {
		client.baseDN = fallbackDN
		client.log.Infow("inferred base DN from server domain", "base_dn", client.baseDN, "validated", false)
		return nil
	}

	return fmt.Errorf("unable to infer base DN from address: %s", client.address)
}

func domainPartsToDN(parts []string) string {
	dnParts := make([]string, 0, len(parts))
	for _, part := range parts {
		dnParts = append(dnParts, fmt.Sprintf("dc=%s", part))
	}
	return strings.Join(dnParts, ",")
}

func (client *ldapClient) validateBaseDN(baseDN string) error {
	return client.withLockedConnection(func(conn *ldap.Conn) error {
		req := ldap.NewSearchRequest(
			baseDN,
			ldap.ScopeBaseObject,
			ldap.NeverDerefAliases,
			1,
			client.searchTimeLimit,
			false,
			"(objectClass=*)",
			[]string{"distinguishedName"},
			nil,
		)
		_, err := conn.Search(req)
		return err
	})
}

// dial establishes a new connection to the LDAP server
func (client *ldapClient) dial() (*ldap.Conn, error) {
	client.log.Debugw("ldap client connecting")

	// Connect with or without TLS based on configuration
	var opts []ldap.DialOpt
	if client.tlsConfig != nil {
		opts = append(opts, ldap.DialWithTLSConfig(client.tlsConfig))
	}
	conn, err := ldap.DialURL(client.address, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to dial LDAP server: %w", err)
	}

	// Bind with appropriate method
	switch {
	case client.password != "":
		// Explicit credentials provided
		client.log.Debugw("ldap client bind with provided credentials")
		err = conn.Bind(client.username, client.password)
	case client.username == "" && client.password == "":
		// No credentials: try Windows SSPI auth, fall back to unauthenticated
		err = client.bindWithCurrentUser(conn)
		if err != nil {
			client.log.Debugw("Windows auth not available, falling back to unauthenticated bind", "error", err)
			err = conn.UnauthenticatedBind("")
		}
	default:
		// Username provided but no password: unauthenticated bind
		client.log.Debugw("ldap client unauthenticated bind")
		err = conn.UnauthenticatedBind(client.username)
	}

	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to bind to LDAP server: %w", err)
	}

	return conn, nil
}

// connection checks the connection's health and reconnects if necessary
// withLockedConnection runs fn while holding the client mutex and ensuring the
// underlying LDAP connection is healthy before invoking the callback.
func (client *ldapClient) withLockedConnection(fn func(*ldap.Conn) error) error {
	client.mu.Lock()
	defer client.mu.Unlock()

	if err := client.ensureConnectedLocked(); err != nil {
		return err
	}
	return fn(client.conn)
}

func (client *ldapClient) ensureConnectedLocked() error {
	if client.conn == nil || client.conn.IsClosing() {
		conn, err := client.dial()
		if err != nil {
			return err
		}
		client.conn = conn
	}
	return nil
}

// findObjectBy searches for an object and returns its mapped values.

func (client *ldapClient) findObjectBy(searchBy string) ([]string, error) {
	var result *ldap.SearchResult
	// Format the filter and perform the search
	filter := fmt.Sprintf("(%s=%s)", client.searchAttr, searchBy)
	searchRequest := ldap.NewSearchRequest(
		client.baseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 1, client.searchTimeLimit, false,
		filter, []string{client.mappedAttr}, nil,
	)

	// Execute search while holding the connection lock to avoid concurrent usage of *ldap.Conn
	err := client.withLockedConnection(func(conn *ldap.Conn) error {
		var searchErr error
		result, searchErr = conn.Search(searchRequest)
		return searchErr
	})
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}
	if len(result.Entries) == 0 {
		return nil, fmt.Errorf("no entries found for search attribute %s", searchBy)
	}

	// Retrieve the mapped attribute values
	values := result.Entries[0].GetAttributeValues(client.mappedAttr)
	return values, nil
}

// close closes the LDAP connection
func (client *ldapClient) close() {
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.conn != nil {
		client.conn.Close()
		client.conn = nil
	}
}
