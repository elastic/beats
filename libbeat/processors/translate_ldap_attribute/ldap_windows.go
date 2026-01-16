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
	"errors"
	"fmt"
	"time"

	"github.com/go-ldap/ldap/v3"
	"github.com/go-ldap/ldap/v3/gssapi"
)

// sspiBindTimeout is the maximum time to wait for SSPI bind operations.
// SSPI can hang indefinitely when credentials are unavailable (e.g., local user accounts).
const sspiBindTimeout = 10 * time.Second

var errSSPITimeout = errors.New("SSPI bind timed out - this may indicate the process is running as a local user without Kerberos credentials")

func (client *ldapClient) bindPlatformSpecific(conn *ldap.Conn, spn string) error {
	client.log.Infow("Attempting Windows SSPI Bind", "spn", spn)

	resultCh := make(chan error, 1)

	go func() {
		client.log.Debug("Creating SSPI client")
		sspiClient, err := gssapi.NewSSPIClient()
		if err != nil {
			resultCh <- fmt.Errorf("failed to create SSPI client: %w", err)
			return
		}
		defer sspiClient.DeleteSecContext()

		client.log.Debug("SSPI client created, performing GSSAPIBind")
		err = conn.GSSAPIBind(sspiClient, spn, "")
		if err != nil {
			resultCh <- fmt.Errorf("SSPI bind failed: %w", err)
			return
		}

		resultCh <- nil
	}()

	select {
	case err := <-resultCh:
		if err != nil {
			client.log.Errorw("SSPI bind failed", "error", err)
		} else {
			client.log.Info("Windows SSPI Bind Successful")
		}
		return err
	case <-time.After(sspiBindTimeout):
		client.log.Warnw("SSPI bind timed out", "timeout", sspiBindTimeout, "spn", spn)
		return errSSPITimeout
	}
}
