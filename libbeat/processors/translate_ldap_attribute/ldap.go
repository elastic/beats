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
	conn, err := client.connection()
	if err != nil {
		return fmt.Errorf("failed to get connection: %w", err)
	}

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

	result, err := conn.Search(searchRequest)
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

	// Remove scheme
	addr := strings.TrimPrefix(client.address, "ldap://")
	addr = strings.TrimPrefix(addr, "ldaps://")

	var hostname string
	// Split host and port using net.SplitHostPort for robust hostname extraction
	h, _, err := net.SplitHostPort(addr)
	if err != nil {
		// If SplitHostPort fails, assume the whole address is the host (i.e., no port specified)
		hostname = addr
		if strings.Contains(hostname, ":") {
			// If we still have a colon, it was likely a malformed or IPv6 address without brackets
			return fmt.Errorf("unable to parse hostname from address: %s", client.address)
		}
	} else {
		hostname = h
	}

	if hostname == "" {
		return fmt.Errorf("unable to extract hostname from address: %s", client.address)
	}

	// Check if hostname is an IP address
	if net.ParseIP(hostname) != nil {
		return fmt.Errorf("cannot infer base DN from IP address: %s", hostname)
	}

	// Extract domain from hostname
	parts := strings.Split(hostname, ".")
	if len(parts) < 2 {
		return fmt.Errorf("hostname does not contain a domain: %s", hostname)
	}

	// Skip first part if we have 3+ parts (likely hostname like dc1.example.com)
	var domainParts []string
	if len(parts) >= 3 {
		domainParts = parts[1:]
	} else {
		domainParts = parts
	}

	// Convert to DN format: example.com -> dc=example,dc=com
	var dnParts []string
	for _, part := range domainParts {
		dnParts = append(dnParts, fmt.Sprintf("dc=%s", part))
	}

	client.baseDN = strings.Join(dnParts, ",")
	client.log.Infow("inferred base DN from server domain", "base_dn", client.baseDN)
	return nil
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
func (client *ldapClient) connection() (*ldap.Conn, error) {
	client.mu.Lock()
	defer client.mu.Unlock()

	// Check if the connection is still alive
	if client.conn == nil || client.conn.IsClosing() {
		conn, err := client.dial()
		if err != nil {
			return nil, err
		}
		client.conn = conn
	}
	return client.conn, nil
}

// findObjectBy searches for an object and returns its mapped values.
func (client *ldapClient) findObjectBy(searchBy string) ([]string, error) {
	// Ensure the connection is alive or reconnect if necessary
	conn, err := client.connection()
	if err != nil {
		return nil, fmt.Errorf("failed to reconnect: %w", err)
	}

	// Format the filter and perform the search
	filter := fmt.Sprintf("(%s=%s)", client.searchAttr, searchBy)
	searchRequest := ldap.NewSearchRequest(
		client.baseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 1, client.searchTimeLimit, false,
		filter, []string{client.mappedAttr}, nil,
	)

	// Execute search
	result, err := conn.Search(searchRequest)
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
