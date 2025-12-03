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
	"github.com/elastic/elastic-agent-libs/logp"
	"golang.org/x/sys/windows/registry"
)

// discoverDomainFromRegistry attempts to read the Windows domain from the registry.
// This works in service contexts where environment variables may not be available.
func discoverDomainFromRegistry(log *logp.Logger) string {
	// Try primary location: HKLM\SYSTEM\CurrentControlSet\Services\Tcpip\Parameters\Domain
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Services\Tcpip\Parameters`, registry.QUERY_VALUE)
	if err != nil {
		log.Debugw("failed to open registry key for domain lookup", "error", err)
		return ""
	}
	defer k.Close()

	// Try "Domain" value first
	domain, _, err := k.GetStringValue("Domain")
	if err == nil && domain != "" {
		log.Infow("discovered domain name from Windows registry", "key", "Domain", "domain", domain)
		return domain
	}

	// Fallback to "DhcpDomain" if Domain is not set
	domain, _, err = k.GetStringValue("DhcpDomain")
	if err == nil && domain != "" {
		log.Infow("discovered domain name from Windows registry", "key", "DhcpDomain", "domain", domain)
		return domain
	}

	log.Debugw("no domain found in Windows registry", "keys_checked", []string{"Domain", "DhcpDomain"})
	return ""
}
