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
	"net/url"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/go-ldap/ldap/v3"

	"github.com/elastic/elastic-agent-libs/logp"
)

// ldapClient manages a single reusable LDAP connection
type ldapClient struct {
	*ldapConfig

	mu   sync.Mutex
	conn *ldap.Conn

	sspiTimedout atomic.Bool

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

	if client.baseDN == "" {
		// Discover base DN if not provided
		baseDN, err := client.getBaseDN()
		if err != nil {
			client.close()
			return nil, fmt.Errorf("failed to discover base DN: %w", err)
		}
		client.baseDN = baseDN
	}

	return client, nil
}

// dial establishes a new connection to the LDAP server.
// It handles the upgrade to StartTLS if the scheme is ldap:// and a TLS config is present.
func (client *ldapClient) dial() (*ldap.Conn, error) {
	client.log.Debugw("ldap client connecting")

	// Connect with or without TLS based on configuration
	var opts []ldap.DialOpt
	if client.tlsConfig != nil {
		opts = append(opts, ldap.DialWithTLSConfig(client.tlsConfig))
	}

	// ldap.DialURL handles parsing ldap:// vs ldaps://
	conn, err := ldap.DialURL(client.address, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to dial LDAP server: %w", err)
	}

	// Explicitly handle StartTLS upgrade.
	// DialURL connects to 389 for "ldap://" but does not upgrade automatically.
	// We must do this before binding credentials.
	if strings.HasPrefix(client.address, "ldap://") && client.tlsConfig != nil {
		if err := conn.StartTLS(client.tlsConfig); err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to upgrade connection to StartTLS: %w", err)
		}
		client.log.Debug("connection upgraded to StartTLS")
	}

	switch {
	case client.password != "":
		client.log.Debugw("ldap client bind with provided credentials")
		if err = conn.Bind(client.username, client.password); err == nil {
			return conn, nil
		} else {
			client.log.Debugw("ldap client bind with provided credentials failed", "error", err)
		}
	case client.username == "" && client.password == "":
		client.log.Debugw("trying automatic ldap client bind")
		if err = client.bindAuto(conn); err == nil {
			return conn, nil
		} else {
			client.log.Debugw("automatic ldap client bind failed", "error", err)
		}
	}

	client.log.Debugw("trying ldap client unauthenticated bind")
	if err = conn.UnauthenticatedBind(client.username); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to bind to LDAP server: %w", err)
	}
	return conn, nil
}

// bindAuto attempts to authenticate using the best available platform-specific method.
// On Windows: SSPI (Current User)
// On Linux: Kerberos Cache (Current User)
func (client *ldapClient) bindAuto(conn *ldap.Conn) error {
	// Parse hostname for SPN (Service Principal Name)
	// SPN Format: ldap/server.example.com
	parsedURL, err := url.Parse(client.address)
	if err != nil {
		return fmt.Errorf("failed to parse LDAP address: %w", err)
	}
	// Canonicalize the SPN (Active Directory expects this format)
	spn := fmt.Sprintf("ldap/%s", strings.ToLower(parsedURL.Hostname()))
	return client.bindPlatformSpecific(conn, spn)
}

// getBaseDN discovers base DN (if needed) and detects server type
func (client *ldapClient) getBaseDN() (string, error) {
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
		return "", fmt.Errorf("rootDSE query failed: %w", err)
	}

	if len(result.Entries) == 0 {
		return "", fmt.Errorf("no entries returned from rootDSE")
	}

	entry := result.Entries[0]

	// 1. Prefer defaultNamingContext (Active Directory standard)
	if values := entry.GetAttributeValues("defaultNamingContext"); len(values) > 0 {
		client.log.Infow("discovered base DN via defaultNamingContext", "base_dn", values[0])
		return values[0], nil
	}

	// 2. Fallback to namingContexts (OpenLDAP / Standard)
	// We must filter out system contexts like cn=config, cn=schema, etc.
	if values := entry.GetAttributeValues("namingContexts"); len(values) > 0 {
		for _, v := range values {
			lowerV := strings.ToLower(v)
			// Skip common system contexts
			if strings.HasPrefix(lowerV, "cn=config") ||
				strings.HasPrefix(lowerV, "cn=schema") ||
				strings.HasPrefix(lowerV, "cn=monitor") ||
				strings.HasPrefix(lowerV, "cn=subschema") {
				continue
			}

			// We return the first "reasonable" context we find.
			// Usually, this will be the main data partition (e.g., dc=example,dc=com)
			client.log.Infow("discovered base DN via namingContexts", "base_dn", v)
			return v, nil
		}

		// If we iterated everything and only found system contexts (unlikely for a user DB),
		// we default to the first one but log a warning.
		client.log.Warnw("only system contexts found in namingContexts, defaulting to first value", "base_dn", values[0])
		return values[0], nil
	}

	return "", fmt.Errorf("base DN not found in rootDSE")
}

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
		// Ensure previous connection is fully closed
		if client.conn != nil {
			client.conn.Close()
		}

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
