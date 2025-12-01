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

//go:build windows && !requirefips

package translate_ldap_attribute

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/go-ldap/ldap/v3"
	"github.com/go-ldap/ldap/v3/gssapi"
)

// bindWithCurrentUser performs GSSAPI bind using the current Windows user's credentials via SSPI.
func (client *ldapClient) bindWithCurrentUser(conn *ldap.Conn) error {
	client.log.Info("using Windows SSPI authentication with current user credentials")

	// Create SSPI client using current process credentials
	sspiClient, err := gssapi.NewSSPIClient()
	if err != nil {
		return fmt.Errorf("failed to create SSPI client: %w", err)
	}
	defer sspiClient.Close()

	// Extract hostname from LDAP address for SPN
	parsedURL, err := url.Parse(client.address)
	if err != nil {
		return fmt.Errorf("failed to parse LDAP address: %w", err)
	}
	hostname := parsedURL.Hostname()
	if hostname == "" {
		return fmt.Errorf("could not extract hostname from address: %s", client.address)
	}

	// Service Principal Name format: ldap/<hostname>
	servicePrincipal := fmt.Sprintf("ldap/%s", strings.ToLower(hostname))
	client.log.Debugw("performing GSSAPI bind", "spn", servicePrincipal)

	// Perform GSSAPI bind
	err = conn.GSSAPIBind(sspiClient, servicePrincipal, "")
	if err != nil {
		return fmt.Errorf("GSSAPI bind failed: %w", err)
	}

	client.log.Info("GSSAPI bind successful")
	return nil
}
