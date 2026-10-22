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

package secretstores

import (
	"errors"
	"fmt"
	"net/netip"
	"strings"

	conf "github.com/elastic/elastic-agent-libs/config"

	"github.com/elastic/beats/v7/heartbeat/monitors/plugin"
)

// allowedHostsKey is the secret_stores entry setting restricting the hosts
// the secrets of a connection can be sent to.
const allowedHostsKey = "allowed_hosts"

// ErrNotAllowed is wrapped by the errors of monitors that can't use a
// connection because of its allowed_hosts.
var ErrNotAllowed = errors.New("not allowed by " + allowedHostsKey)

// checkAllowedHosts returns an error unless every destination of the resolved
// monitor config cfg is an allowed host of each entry. Hosts are not named in
// errors: resolved hosts can be secrets.
func checkAllowedHosts(entries []entry, monitorType string, destinations plugin.DestinationsFunc, cfg *conf.C) error {
	if len(entries) == 0 {
		return nil
	}
	if destinations == nil {
		return fmt.Errorf("%s: connection %q: %w: monitor type %q is not supported", ConfigKey, entries[0].name, ErrNotAllowed, monitorType)
	}
	hosts, err := destinations(cfg)
	if err != nil {
		return fmt.Errorf("%s: connection %q: %w: %s", ConfigKey, entries[0].name, ErrNotAllowed, err.Error())
	}
	for _, e := range entries {
		for _, h := range hosts {
			if !e.allowedHosts[normalizeHost(h)] {
				return fmt.Errorf("%s: connection %q: %w: the monitor connects to a host that is not allowed", ConfigKey, e.name, ErrNotAllowed)
			}
		}
	}
	return nil
}

// parseAllowedHosts returns the normalized hosts of an allowed_hosts value.
func parseAllowedHosts(raw any) (map[string]bool, error) {
	if raw == nil {
		return nil, nil
	}
	items, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("%s: expected a list of host names or IPs, got %T", allowedHostsKey, raw)
	}
	hosts := make(map[string]bool, len(items))
	for _, item := range items {
		h, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("%s: expected a host name or IP, got %T", allowedHostsKey, item)
		}
		if !validHost(h) {
			return nil, fmt.Errorf("%s: %q must be a host name or IP, without scheme, port or path", allowedHostsKey, h)
		}
		hosts[normalizeHost(h)] = true
	}
	return hosts, nil
}

// validHost reports whether h is a host name or an IP. Wildcards are not supported.
func validHost(h string) bool {
	if h == "" || strings.TrimSpace(h) != h {
		return false
	}
	if _, err := netip.ParseAddr(h); err == nil {
		return true
	}

	return !strings.ContainsAny(h, ":/@?#*[]\\ \t\r\n")
}

// normalizeHost returns the form of a host used to compare it with the
// allowed hosts: lowercase, without the trailing dot of a fully qualified
// name, IPs in their canonical form. A form not normalized here can only
// prevent a match, never make two different hosts match.
func normalizeHost(h string) string {
	h = strings.TrimSuffix(h, ".")
	if ip, err := netip.ParseAddr(h); err == nil {
		return ip.Unmap().String()
	}
	return strings.ToLower(h)
}
