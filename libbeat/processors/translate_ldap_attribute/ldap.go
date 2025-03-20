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
	"sync"

	"github.com/go-ldap/ldap/v3"
)

// ldapClient manages a single reusable LDAP connection
type ldapClient struct {
	conn *ldap.Conn
	mu   sync.Mutex
	*ldapConfig
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

// newLDAPClient initializes a new ldapClient with a single connection
func newLDAPClient(config *ldapConfig) (*ldapClient, error) {
	client := &ldapClient{ldapConfig: config}

	// Establish initial connection
	if err := client.connect(); err != nil {
		return nil, err
	}

	return client, nil
}

// connect establishes a new connection to the LDAP server
func (client *ldapClient) connect() error {
	client.mu.Lock()
	defer client.mu.Unlock()

	// Connect with or without TLS based on configuration
	var opts []ldap.DialOpt
	if client.tlsConfig != nil {
		opts = append(opts, ldap.DialWithTLSConfig(client.tlsConfig))
	}
	conn, err := ldap.DialURL(client.address, opts...)
	if err != nil {
		return fmt.Errorf("failed to dial LDAP server: %w", err)
	}

	if client.password != "" {
		err = conn.Bind(client.username, client.password)
	} else {
		err = conn.UnauthenticatedBind(client.username)
	}

	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to bind to LDAP server: %w", err)
	}

	client.conn = conn
	return nil
}

// reconnect checks the connection's health and reconnects if necessary
func (client *ldapClient) reconnect() error {
	client.mu.Lock()
	defer client.mu.Unlock()

	// Check if the connection is still alive
	if client.conn.IsClosing() {
		return client.connect()
	}
	return nil
}

// findObjectBy searches for an object and returns its mapped values.
func (client *ldapClient) findObjectBy(searchBy string) ([]string, error) {
	// Ensure the connection is alive or reconnect if necessary
	if err := client.reconnect(); err != nil {
		return nil, fmt.Errorf("failed to reconnect: %w", err)
	}

	client.mu.Lock()
	defer client.mu.Unlock()

	// Format the filter and perform the search
	filter := fmt.Sprintf("(%s=%s)", client.searchAttr, searchBy)
	searchRequest := ldap.NewSearchRequest(
		client.baseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 1, client.searchTimeLimit, false,
		filter, []string{client.mappedAttr}, nil,
	)

	// Execute search
	result, err := client.conn.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}
	if len(result.Entries) == 0 {
		return nil, fmt.Errorf("no entries found for search attribute %s", searchBy)
	}

	// Retrieve the CN attribute
	cn := result.Entries[0].GetAttributeValues(client.mappedAttr)
	return cn, nil
}

// close closes the LDAP connection
func (client *ldapClient) close() {
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.conn != nil {
		client.conn.Close()
	}
}
